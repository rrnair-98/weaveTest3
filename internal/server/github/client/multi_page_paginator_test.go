package client

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"net/http"
	"reflect"
	"testing"
	"weaveTest/internal/config"
	"weaveTest/internal/proto/generated"
)

func TestMultiPagePaginator_Paginate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	srHello := getSearchRequestWithUrl("", "hello", 1, 30, "fixtures/hello_query_first_response.json", t)
	srHelloSecond := getSearchRequestWithUrl("", "hello", 2, 30, "fixtures/hello_query_second_response.json", t)
	srWorld := getSearchRequestWithUrl("", "world", 1, 30, "fixtures/world_search_paginated_response.json", t)
	srHelloWithEmptyQuery := getSearchRequestWithUrl("", "", 1, 30, "", t)
	srHelloWithInvalidQuery := getSearchRequestWithUrl("", "hello AND hello OR world OR amazing OR foobar OR err AND data", 1, 30, "", t)
	// since the responses will be appended
	srHello.expectedResponse.Results = append(srHello.expectedResponse.Results, srHelloSecond.expectedResponse.Results...)

	fmt.Println(srHelloWithInvalidQuery.url)
	fmt.Println(srHelloWithEmptyQuery.url)

	responseMap := make(map[string]*http.Response)
	t.Log(srHello.url)
	responseMap[srHello.url] = createResponse(http.StatusOK, srHello.responseBody)
	res, ok := responseMap[srHello.url]
	t.Log("res== nil", res == nil, " ok ", ok)
	responseMap[srHelloSecond.url] = createResponse(http.StatusOK, srHelloSecond.responseBody)
	responseMap[srWorld.url] = createResponse(http.StatusOK, srWorld.responseBody)
	responseMap[srHelloWithInvalidQuery.url] = createResponse(http.StatusBadRequest, "")
	responseMap[srHelloWithEmptyQuery.url] = createResponse(http.StatusUnprocessableEntity, "")

	mockedRequestMaker := NewMockedRequestMaker(logger, responseMap)
	multiPagePaginator := newMultiPagePaginator(logger, mockedRequestMaker, config.Pagination{
		Kind:                "multi",
		MaxPages:            2,
		PerPage:             30,
		FetchAllPages:       false,
		BlockingRateLimited: false,
	})

	tests := []struct {
		name                 string
		searchRequestWithUrl *searchRequestWithUrl
		want                 *generated.SearchResponse
		wantErr              bool
		errorGrpcCode        codes.Code
	}{
		{
			name:                 "hello world search",
			searchRequestWithUrl: srHello,
			want:                 srHello.expectedResponse,
			wantErr:              false,
			errorGrpcCode:        codes.OK,
		},
		{
			name:                 "world search",
			searchRequestWithUrl: srWorld,
			want:                 srWorld.expectedResponse,
			wantErr:              false,
			errorGrpcCode:        codes.OK,
		},
		{
			name:                 "hello world search with invalid query",
			searchRequestWithUrl: srHelloWithInvalidQuery,
			want:                 nil,
			wantErr:              true,
			errorGrpcCode:        codes.InvalidArgument,
		},
		{
			name:                 "hello world search with empty query",
			searchRequestWithUrl: srHelloWithEmptyQuery,
			want:                 nil,
			wantErr:              true,
			errorGrpcCode:        codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.name)
			got, err := multiPagePaginator.Paginate(context.Background(), tt.searchRequestWithUrl.searchRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("SinglePagePaginator.Paginate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("SinglePagePaginator.Paginate() responses dont match = %v, want %v", got, tt.want)
				}
			}
			if err != nil {
				t.Log("status: %s \n" + err.GrpcStatus().String())
			}
			if tt.wantErr && err != nil && err.GrpcStatus() != tt.errorGrpcCode {
				t.Errorf("SinglePagePaginator.Paginate() grpc codes dont match error code = %v, want %v", err.GrpcStatus(), tt.errorGrpcCode)
			}
		})
	}

}
