package main

import (
	"go.uber.org/zap"
	"time"
	"weaveTest/internal/config"
	"weaveTest/internal/server"
	"weaveTest/internal/server/github/client"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// since we need a logger instance
	client.InitRateLimiter(9, time.Minute, logger)
	err = config.InitEnvFromFile("./.env.json")
	if err != nil {
		logger.Error("failed to load env file", zap.Error(err))
		panic(err)
	}
	e := config.GetEnv()
	logger.Debug("paginator conf: ", zap.Bool("rateLimiterEnabled", e.Paginator.RateLimited), zap.String("kind", e.Paginator.Kind), zap.Int("maxPages", e.Paginator.MaxPages), zap.Int("perPage", e.Paginator.PerPage), zap.Bool("fetchAllPages", e.Paginator.FetchAllPages))
	if err != nil {
		logger.Error("failed to load env file", zap.Error(err))
		panic(err)
	}

	// TODO: add command line args for port and githubToken
	serverInstance := server.NewServerWithDefaultPort(logger)
	serverInstance.Start()
}
