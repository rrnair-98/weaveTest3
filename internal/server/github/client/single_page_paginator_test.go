package client

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"net/http"
	"reflect"
	"testing"
	"weaveTest/internal/proto/generated"
)

func TestSinglePagePaginator_Paginate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	responseMap := make(map[string]*http.Response)
	srHello := getSearchRequestWithUrl("", "hello", 1, 30, "fixtures/hello_query_first_response.json", t)
	srWorld := getSearchRequestWithUrl("", "world", 1, 30, "fixtures/world_search_response.json", t)
	srHelloWithEmptyQuery := getSearchRequestWithUrl("", "", 1, 30, "", t)
	srHelloWithInvalidQuery := getSearchRequestWithUrl("", "hello AND hello OR world OR amazing OR foobar OR err AND data", 1, 30, "", t)

	fmt.Println(srHelloWithInvalidQuery.url)
	fmt.Println(srHelloWithEmptyQuery.url)
	responseMap[srHello.url] = createResponse(http.StatusOK, srHello.responseBody)
	responseMap[srWorld.url] = createResponse(http.StatusOK, srWorld.responseBody)
	responseMap[srHelloWithInvalidQuery.url] = createResponse(http.StatusBadRequest, "")
	responseMap[srHelloWithEmptyQuery.url] = createResponse(http.StatusUnprocessableEntity, "")
	// TODO: add more tests
	mockedRequestMaker := NewMockedRequestMaker(logger, responseMap)
	singlePagePaginator := newSinglePagePaginator(logger, mockedRequestMaker)
	// genResponse, err := singlePagePaginator.Paginate(t.Context(), srHello.searchRequest)

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
			got, err := singlePagePaginator.Paginate(context.Background(), tt.searchRequestWithUrl.searchRequest)
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
