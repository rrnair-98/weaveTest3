package main

import (
	"go.uber.org/zap"
	"weaveTest/internal/config"
	"weaveTest/internal/server"
	"weaveTest/internal/server/github/client"
	"weaveTest/internal/server/github/client/rate_limiter"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// since we need a logger instance
	err = config.InitEnvFromFile("./.env.json")
	if err != nil {
		logger.Error("failed to load env file", zap.Error(err))
		panic(err)
	}
	e := config.GetEnv()
	rate_limiter.InitGitHubRateLimiter(logger, e.GitToken, e.Paginator.BlockingRateLimited)
	client.InitSinglePagePaginator(logger, nil)
	client.InitMultiPagePaginator(logger, nil, e.Paginator)
	logger.Debug("paginator conf: ", zap.Bool("rateLimiterEnabled", e.Paginator.BlockingRateLimited), zap.String("kind", e.Paginator.Kind), zap.Int("maxPages", e.Paginator.MaxPages), zap.Int("perPage", e.Paginator.PerPage), zap.Bool("fetchAllPages", e.Paginator.FetchAllPages))

	// TODO: add command line args for port and githubToken
	serverInstance := server.NewServerWithDefaultPort(logger)
	serverInstance.Start()
}
