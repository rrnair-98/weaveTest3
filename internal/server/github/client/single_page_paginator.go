package client

import (
	"context"
	"go.uber.org/zap"
	"sync"
	"weaveTest/internal/proto/generated"
	appError "weaveTest/internal/server/github/client/errors"
)

var (
	singleInstance *SinglePagePaginator
	sppOnce        sync.Once
	sppMu          sync.RWMutex
)

// SinglePagePaginator is responsible for handling pagination for single-page search requests.
// It uses a RequestMaker to perform HTTP requests and logs activities using a zap.Logger.
type SinglePagePaginator struct {
	logger       *zap.Logger
	requestMaker RequestMaker
}

// GetSinglePagePaginator returns the singleton instance of SinglePagePaginator
func GetSinglePagePaginator(logger *zap.Logger, maker RequestMaker) *SinglePagePaginator {
	sppOnce.Do(func() {
		sppMu.Lock()
		defer sppMu.Unlock()
		if maker == nil {
			maker = NewDefaultRequestMaker(logger)
		}
		singleInstance = &SinglePagePaginator{
			logger:       logger,
			requestMaker: maker,
		}
	})

	return singleInstance
}

// DefaultSinglePagePaginator initializes and returns a default SinglePagePaginator with logger and default RequestMaker.
func DefaultSinglePagePaginator(logger *zap.Logger) *SinglePagePaginator {
	return GetSinglePagePaginator(logger, nil)
}

// Paginate executes a single-page pagination for search requests, fetching and returning search results or an AppError.
func (s *SinglePagePaginator) Paginate(ctx context.Context, request *generated.SearchRequest) (*generated.SearchResponse, appError.AppError) {
	queryEscapedUrl, err := genUrl(request, defaultPerPage, defaultPageNumber, s.logger)
	if err != nil {
		s.logger.Error("failed to generate qualified url", zap.Error(err))
		return nil, err
	}
	return fetchDataFromRemote(ctx, queryEscapedUrl, s.logger, s.requestMaker)
}
