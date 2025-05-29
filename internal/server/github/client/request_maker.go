package client

import (
	"context"
	"go.uber.org/zap"
	"net/http"
	appError "weaveTest/internal/server/github/client/errors"
	"weaveTest/internal/server/github/client/rate_limiter"
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
// Ctx is the context to handle request cancellation or timeout.
// Url is the target endpoint for the HTTP request.
// Logger is used for structured logging during the lifecycle of the request.
func performRequest(ctx context.Context, url string, logger *zap.Logger) (*http.Response, appError.AppError) {
	rateLimiter := rate_limiter.GetGitHubRateLimiter()
	logger.Debug("rate limiter enabled", zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, httpMethod, url, nil)
	if err != nil {
		logger.Error("failed to create request", zap.Error(err))
		return nil, appError.NewInternalError(err, appError.InvalidHttpClient, "failed to create request")
	}

	res, err := rateLimiter.ExecuteRequest(req)
	if err != nil {
		logger.Error("failed to perform http request", zap.Error(err))
		return nil, appError.NewInternalError(err, appError.InvalidHttpClient, "failed to perform http request")
	}

	return res, nil

}
