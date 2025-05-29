package github

import (
	"context"
	"go.uber.org/zap"
	"weaveTest/internal/config"
	"weaveTest/internal/proto/generated"
	internal "weaveTest/internal/server/github/client"
	"weaveTest/internal/server/github/client/errors"
)

type RepositoryDataFetcher struct {
	logger    zap.Logger
	paginator internal.Paginator
}

func NewDataFetcher(logger zap.Logger) *RepositoryDataFetcher {
	// uses default single page paginator
	env := config.GetEnv()
	if env.Paginator.IsMultiPage() {
		return NewDataFetcherWithPagination(logger, internal.GetMultiPagePaginator())
	}
	return NewDataFetcherWithPagination(logger, internal.GetSinglePagePaginator())
}

func NewDataFetcherWithPagination(logger zap.Logger, paginator internal.Paginator) *RepositoryDataFetcher {
	return &RepositoryDataFetcher{
		logger:    logger,
		paginator: paginator,
	}
}

func (dataFetcher *RepositoryDataFetcher) Fetch(ctx context.Context, request *generated.SearchRequest) (*generated.SearchResponse, errors.AppError) {
	return dataFetcher.paginator.Paginate(ctx, request)
}
