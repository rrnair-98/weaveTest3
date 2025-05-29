package client

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"weaveTest/internal/proto/generated"
	appError "weaveTest/internal/server/github/client/errors"
)

const (
	httpMethod = "GET"
)

// Paginator provides functionality for paginating search results based on a given search request and context.
// It returns a search response or an application-specific error when processing the request.
type Paginator interface {
	Paginate(ctx context.Context, request *generated.SearchRequest) (*generated.SearchResponse, appError.AppError)
}

// genUrl generates a qualified GitHub API search URL based on search request parameters, page size, and page number.
// It validates the query and constructs the URL with or without a user filter, logging the final URL for debugging.
// Returns the generated URL or an application error in case of invalid input or query validation failure.
func genUrl(request *generated.SearchRequest, pageSize int, pageNum int, logger *zap.Logger) (string, appError.AppError) {
	var query = queryString(request.SearchTerm)
	qualifiedUrl := ""
	if err := query.Validate(); err != nil {
		return "", appError.NewInternalError(err, appError.InvalidQuery, err.Error())
	}
	if request.User == "" {
		qualifiedUrl, _ = query.ToUrlWithMaxPerPage(pageNum, pageSize)
	} else {
		qualifiedUrl, _ = query.ToUrlWithUser(request.User, pageNum, pageSize)
	}
	logger.Debug("url verified: ", zap.String("qualifiedUrl", qualifiedUrl))
	return qualifiedUrl, nil
}

// fetchDataFromRemote retrieves data from a remote server using the provided RequestMaker and URL.
// It takes a context for request handling, a logger for error tracking, and returns a SearchResponse or an AppError.
// This function ensures the response body is read and unmarshalled into a structured format while handling potential errors.
func fetchDataFromRemote(ctx context.Context, url string, logger *zap.Logger, requestMaker RequestMaker) (*generated.SearchResponse, appError.AppError) {
	res, err := requestMaker.Perform(ctx, url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, localErr := io.ReadAll(res.Body)
	if localErr != nil {
		logger.Error("failed to read response body: ", zap.Error(err))
		return nil, appError.NewInternalError(err, appError.InvalidJSONBody, "failed to read response body")
	}
	_, response, err := handleResponseBodyBytes(body, res.StatusCode, url, logger)
	return response, err
}

// handleResponseBodyBytes processes the HTTP response body and status code, logging outcomes and handling errors.
// It unmarshals the response body into a structured format, transforming repository items to search results.
// Returns the total count of results, a SearchResponse object, or an AppError in case of failure.
func handleResponseBodyBytes(body []byte, statusCode int, url string, logger *zap.Logger) (int, *generated.SearchResponse, appError.AppError) {
	if err := handleHttpErrors(statusCode, body, url, logger); err != nil {
		return 0, nil, err
	}
	logger.Debug("successfully fetched data from remote")
	var response CodeSearchResponse
	err := json.Unmarshal(body, &response)
	if err != nil {
		logger.Error("failed to unmarshal response body: ", zap.Error(err))
		return 0, nil, appError.NewInternalError(err, appError.InvalidJSONBody, string(body))
	}
	res := RepositoryItemsToResult(response.RepositoryItems)
	logger.Debug("successfully fetched data from remote", zap.Int("numResults", len(res)))
	return response.TotalCount, &generated.SearchResponse{Results: res}, nil
}

// handleHttpErrors processes HTTP responses and maps specific status codes to detailed, custom error types for logging.
func handleHttpErrors(statusCode int, body []byte, url string, logger *zap.Logger) appError.AppError {
	logger.Debug("handling http errors", zap.Int("statusCode", statusCode))
	if statusCode == http.StatusOK {
		return nil
	}
	switch statusCode {
	// TODO: Wrap errors in a custom error type
	case http.StatusNotAcceptable:
		return appError.NewRemoteError(fmt.Errorf("query string was wrongly formatted: %s", url), statusCode, string(body))
	case http.StatusGatewayTimeout:
		return appError.NewRemoteError(fmt.Errorf("gateway timed out for the search API"), statusCode, "")
	case http.StatusTooManyRequests, http.StatusForbidden:
		return appError.NewRemoteError(fmt.Errorf("rate limit exceeded"), statusCode, string(body))
	case http.StatusUnauthorized:
		return appError.NewRemoteError(fmt.Errorf("unauthorized, the token being used could either have expired or is invalid"), statusCode, string(body))
	case http.StatusUnprocessableEntity:
		// https://docs.github.com/en/rest/search/search?apiVersion=2022-11-28#access-errors-or-missing-search-results
		return appError.NewRemoteError(fmt.Errorf("either the qualifer provided was invalid or the resource specified in the qualifier cant be accessed"), statusCode, string(body))
	default:
		return nil
	}
}
