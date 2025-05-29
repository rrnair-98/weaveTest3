package client

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"weaveTest/internal/proto/generated"
	appError "weaveTest/internal/server/github/client/errors"
)

const badQueryFmt = "https://github.com/search?%s"

type MockedRequestMaker struct {
	RequestMaker
	logger          *zap.Logger
	responseMap     map[string]*http.Response
	responseCounter int
}

func (mrm *MockedRequestMaker) Perform(ctx context.Context, url string) (*http.Response, appError.AppError) {
	response := mrm.responseMap[url]
	return response, nil
}

func NewMockedRequestMaker(logger *zap.Logger, responses map[string]*http.Response) *MockedRequestMaker {
	return &MockedRequestMaker{
		logger:          logger,
		responseMap:     responses,
		responseCounter: 0,
	}
}

func createResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func readFileContentsAsString(filePath string, t *testing.T) string {
	if filePath == "" {
		t.Log("filePath is empty, returning empty string")
		return ""
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}
	// t.Log("file contents: ", string(content))
	return string(content)
}

type searchRequestWithUrl struct {
	searchRequest    *generated.SearchRequest
	url              string
	responseBody     string
	expectedResponse *generated.SearchResponse
}

func getSearchRequestWithUrl(user string, query string, pageNumber int, perPage int, filePath string, t *testing.T) *searchRequestWithUrl {
	helloSearchRequest := &generated.SearchRequest{
		User:       user,
		SearchTerm: query,
	}
	q, err := queryString(query).ToUrlWithUser(user, pageNumber, perPage)
	if err != nil {
		// we want to simulate a bad call to the server
		q = url.QueryEscape(fmt.Sprintf(badQueryFmt, q))
	}
	fileContents := readFileContentsAsString(filePath, t)

	// writing contents to searchResponse
	var searchResponse *generated.SearchResponse
	if fileContents != "" {
		var codeSearchResponse CodeSearchResponse
		err := json.Unmarshal([]byte(fileContents), &codeSearchResponse)
		if err != nil {
			t.Fatalf("Error unmarshalling file contents: %v", err)
		}
		searchResponse = &generated.SearchResponse{Results: RepositoryItemsToResult(codeSearchResponse.RepositoryItems)}
	}
	return &searchRequestWithUrl{
		searchRequest:    helloSearchRequest,
		url:              q,
		responseBody:     fileContents,
		expectedResponse: searchResponse,
	}
}
