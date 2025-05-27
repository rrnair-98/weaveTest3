package client

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"weaveTest/internal/config"
	appError "weaveTest/internal/server/github/client/errors"
)

// RequestMaker defines an interface for performing HTTP requests, returning an HTTP response or an application error.
type RequestMaker interface {
	Perform(ctx context.Context, url string) (*http.Response, appError.AppError)
}

// DefaultRequestMaker is a type responsible for performing HTTP requests using a provided logger for logging.
type DefaultRequestMaker struct {
	logger *zap.Logger
}

// Perform sends an HTTP GET request to the specified URL and returns the HTTP response or an appError upon failure.
func (r *DefaultRequestMaker) Perform(ctx context.Context, url string) (*http.Response, appError.AppError) {
	return performRequest(ctx, url, r.logger)
}

// NewDefaultRequestMaker initializes a DefaultRequestMaker with the specified logger for logging HTTP request actions.
func NewDefaultRequestMaker(logger *zap.Logger) *DefaultRequestMaker {
	return &DefaultRequestMaker{
		logger: logger,
	}
}

// performRequest sends an HTTP GET request to the specified URL, handling rate limits, and logs errors if any occur.
// It returns an *http.Response object or an appError if the request fails.
// ctx is the context to handle request cancellation or timeout.
// url is the target endpoint for the HTTP request.
// logger is used for structured logging during the lifecycle of the request.
func performRequest(ctx context.Context, url string, logger *zap.Logger) (*http.Response, appError.AppError) {
	env := config.GetEnv()
	client := &http.Client{}
	req, err := http.NewRequest(httpMethod, url, nil)
	if err != nil {
		logger.Error("failed to create httpClient", zap.Error(err))
		return nil, appError.NewInternalError(err, appError.InvalidHttpClient, "failed to create httpClient")
	}
	// dont need to set this accept header, since text_matches is not being used right now
	// req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf(bearerFmt, env.GitToken))
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	logger.Debug("performing http request for url: ", zap.String("url", url))
	if env.Paginator.RateLimited {
		rateLimiter := GetRateLimiter()
		logger.Debug("rate limiter wait started")
		err := rateLimiter.WaitWithContext(ctx)
		logger.Debug("rate limiter wait completed")
		if err != nil {
			logger.Error("rate limiter failed, limit exceeded", zap.Error(err))
			return nil, appError.NewRemoteError(err, http.StatusTooManyRequests, "rate limiter failed, limit exceeded")
		}
	}
	res, err := client.Do(req)
	if err != nil {
		logger.Error("failed to perform http request: ", zap.Error(err))
		return nil, appError.NewInternalError(err, appError.InvalidHttpClient, "failed to perform http request")
	}
	return res, nil
}
