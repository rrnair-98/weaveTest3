package client

import (
	"context"
	"go.uber.org/zap"
	"io"
	"sync"
	"weaveTest/internal/config"
	"weaveTest/internal/proto/generated"
	appError "weaveTest/internal/server/github/client/errors"
)

var (
	instance *MultiPagePaginator
	mppOnce  sync.Once
	mppMu    sync.RWMutex
)

// GetMultiPagePaginator returns the singleton instance of MultiPagePaginator
func GetMultiPagePaginator(logger *zap.Logger, requestMaker RequestMaker) *MultiPagePaginator {
	mppOnce.Do(func() {
		mppMu.Lock()
		if requestMaker == nil {
			requestMaker = NewDefaultRequestMaker(logger)
		}
		instance = &MultiPagePaginator{
			logger:       logger,
			requestMaker: requestMaker,
		}
		mppMu.Unlock()
	})
	return instance
}

// InitMultiPagePaginator initializes the singleton instance of MultiPagePaginator
func InitMultiPagePaginator(logger *zap.Logger, requestMaker RequestMaker) {
	GetMultiPagePaginator(logger, requestMaker)
}

func DefaultMultiPagePaginator(logger *zap.Logger) *MultiPagePaginator {
	return GetMultiPagePaginator(logger, nil)
}

type MultiPagePaginator struct {
	logger       *zap.Logger
	requestMaker RequestMaker
}

func (m *MultiPagePaginator) Paginate(ctx context.Context, request *generated.SearchRequest) (*generated.SearchResponse, appError.AppError) {
	env := config.GetEnv()
	maxConfigPages := env.Paginator.MaxPages
	perPage := env.Paginator.PerPage

	// If perPage is not set or invalid, use default
	if perPage <= 0 || perPage > 100 {
		perPage = defaultPerPage
	}

	// Generate URL for the first page
	firstUrl, err := genUrl(request, perPage, defaultPageNumber, m.logger)
	if err != nil {
		m.logger.Error("failed to generate qualified url", zap.Error(err))
		return nil, err
	}

	// Make the first request
	res, err := m.requestMaker.Perform(ctx, firstUrl)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Read and parse the first response
	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		m.logger.Error("failed to read response body", zap.Error(readErr))
		return nil, appError.NewInternalError(readErr, appError.PaginationFailed, "failed to read response body")
	}

	totalCount, searchResponse, err := handleResponseBodyBytes(body, res.StatusCode, firstUrl, m.logger)
	if err != nil {
		return nil, err
	}

	// Calculate the total number of pages based on TotalCount
	totalPages := (totalCount + perPage - 1) / perPage
	m.logger.Debug("total pages calculated", zap.Int("totalPages", totalPages))

	pagesToFetch := totalPages
	if maxConfigPages > 0 && pagesToFetch > maxConfigPages {
		pagesToFetch = maxConfigPages
	}

	m.logger.Debug("pagination details",
		zap.Int("totalCount", totalCount),
		zap.Int("perPage", perPage),
		zap.Int("totalPages", totalPages),
		zap.Int("configMaxPages", maxConfigPages),
		zap.Int("pagesToFetch", pagesToFetch))

	// If only one page or no more pages, return early
	if pagesToFetch <= 1 {
		return searchResponse, nil
	}

	// Fetch the remaining pages
	pageNum := 0
	for pageNum = 2; pageNum <= pagesToFetch; pageNum++ {
		// Generate URL for the current page
		pageUrl, err := genUrl(request, perPage, pageNum, m.logger)
		if err != nil {
			m.logger.Error("failed to generate url for page", zap.Int("page", pageNum), zap.Error(err))
			break
		}
		m.logger.Debug("fetching page", zap.Int("page", pageNum), zap.String("url", pageUrl))

		// Request the current page
		pageRes, err := m.requestMaker.Perform(ctx, pageUrl)
		if err != nil {
			m.logger.Error("failed to fetch page", zap.Int("page", pageNum), zap.Error(err))
			break
		}

		pageBody, readErr := io.ReadAll(pageRes.Body)
		if readErr != nil {
			pageRes.Body.Close()
			m.logger.Error("failed to read page response", zap.Int("page", pageNum), zap.Error(readErr))
			break
		}
		_, paginatedResponse, err := handleResponseBodyBytes(pageBody, pageRes.StatusCode, pageUrl, m.logger)
		if err != nil {
			m.logger.Error("failed to handle page response", zap.Int("page", pageNum), zap.Error(err))
			break
		}
		pageRes.Body.Close()
		searchResponse.Results = append(searchResponse.Results, paginatedResponse.Results...)
	}

	m.logger.Debug("pagination completed", zap.Bool("fetcedAll", pageNum == pagesToFetch),
		zap.Int("numPages", len(searchResponse.Results)))
	return searchResponse, nil
}
