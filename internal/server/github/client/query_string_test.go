package client

import "testing"

func TestQueryString_ToUrlWithMaxPerPage(t *testing.T) {
	tests := []struct {
		name       string
		query      queryString
		pageNumber int
		perPage    int
		want       string
		wantErr    bool
	}{
		{
			name:       "Valid query with default pagination",
			query:      queryString("test"),
			pageNumber: 1,
			perPage:    30,
			want:       "https://api.github.com/search/code?q=test&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Valid query with custom pagination",
			query:      queryString("hello world"),
			pageNumber: 2,
			perPage:    50,
			want:       "https://api.github.com/search/code?q=hello+world&per_page=50&page=2",
			wantErr:    false,
		},
		{
			name:       "Query with special characters and pagination",
			query:      queryString("test&query=value"),
			pageNumber: 3,
			perPage:    25,
			want:       "https://api.github.com/search/code?q=test%26query%3Dvalue&per_page=25&page=3",
			wantErr:    false,
		},
		{
			name:       "Invalid page number (negative)",
			query:      queryString("test"),
			pageNumber: -1,
			perPage:    30,
			want:       "https://api.github.com/search/code?q=test&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Invalid page number (zero)",
			query:      queryString("test"),
			pageNumber: 0,
			perPage:    30,
			want:       "https://api.github.com/search/code?q=test&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Invalid per_page (negative)",
			query:      queryString("test"),
			pageNumber: 1,
			perPage:    -10,
			want:       "https://api.github.com/search/code?q=test&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Invalid per_page (zero)",
			query:      queryString("test"),
			pageNumber: 1,
			perPage:    0,
			want:       "https://api.github.com/search/code?q=test&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Invalid per_page (too large)",
			query:      queryString("test"),
			pageNumber: 1,
			perPage:    101,
			want:       "https://api.github.com/search/code?q=test&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Maximum allowed per_page",
			query:      queryString("test"),
			pageNumber: 1,
			perPage:    100,
			want:       "https://api.github.com/search/code?q=test&per_page=100&page=1",
			wantErr:    false,
		},
		{
			name:       "Empty query",
			query:      queryString(""),
			pageNumber: 1,
			perPage:    30,
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.query.ToUrlWithMaxPerPage(tt.pageNumber, tt.perPage)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToUrlWithMaxPerPage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ToUrlWithMaxPerPage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryString_ToUrlWithUser_WithPagination(t *testing.T) {
	tests := []struct {
		name       string
		query      queryString
		user       string
		pageNumber int
		perPage    int
		want       string
		wantErr    bool
	}{
		{
			name:       "Valid query with user and default pagination",
			query:      queryString("test"),
			user:       "github-user",
			pageNumber: 1,
			perPage:    30,
			want:       "https://api.github.com/search/code?q=test+user%3Agithub-user&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Valid query with user and custom pagination",
			query:      queryString("hello world"),
			user:       "github-org",
			pageNumber: 3,
			perPage:    50,
			want:       "https://api.github.com/search/code?q=hello+world+user%3Agithub-org&per_page=50&page=3",
			wantErr:    false,
		},
		{
			name:       "Invalid page number with valid user",
			query:      queryString("test"),
			user:       "octocat",
			pageNumber: -1,
			perPage:    30,
			want:       "https://api.github.com/search/code?q=test+user%3Aoctocat&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Invalid per_page with valid user",
			query:      queryString("test"),
			user:       "octocat",
			pageNumber: 1,
			perPage:    150,
			want:       "https://api.github.com/search/code?q=test+user%3Aoctocat&per_page=30&page=1",
			wantErr:    false,
		},
		{
			name:       "Empty user with valid pagination",
			query:      queryString("test"),
			user:       "",
			pageNumber: 2,
			perPage:    40,
			want:       "https://api.github.com/search/code?q=test&per_page=40&page=2",
			wantErr:    false,
		},
		{
			name:       "Maximum allowed pagination values",
			query:      queryString("test"),
			user:       "github",
			pageNumber: 100,
			perPage:    100,
			want:       "https://api.github.com/search/code?q=test+user%3Agithub&per_page=100&page=100",
			wantErr:    false,
		},
		{
			name:       "Query with qualifiers, user and pagination",
			query:      queryString("language:go filename:main.go"),
			user:       "google",
			pageNumber: 5,
			perPage:    20,
			want:       "https://api.github.com/search/code?q=language%3Ago+filename%3Amain.go+user%3Agoogle&per_page=20&page=5",
			wantErr:    false,
		},
		{
			name:       "Invalid query",
			query:      queryString(""),
			user:       "testuser",
			pageNumber: 1,
			perPage:    30,
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.query.ToUrlWithUser(tt.user, tt.pageNumber, tt.perPage)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToUrlWithUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ToUrlWithUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}
