package client

import (
	"weaveTest/internal/proto/generated"
)

type CodeSearchResponse struct {
	TotalCount        int              `json:"total_count"`
	IncompleteResults bool             `json:"incomplete_results"`
	RepositoryItems   []RepositoryItem `json:"items"`
}

type RepositoryItem struct {
	Name    string     `json:"name"`
	URL     string     `json:"url"`
	GitURL  string     `json:"git_url"`
	HTMLURL string     `json:"html_url"`
	Score   float64    `json:"score"`
	Repo    Repository `json:"repository"`
}

func RepositoryItemsToResult(repoItems []RepositoryItem) []*generated.Result {
	var results = make([]*generated.Result, len(repoItems))
	for i, item := range repoItems {
		results[i] = &generated.Result{
			FileUrl: item.HTMLURL,
			Repo:    item.Repo.HTMLURL,
		}
	}
	return results
}

type Repository struct {
	ID          int    `json:"id"`
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
	Fork        bool   `json:"fork"`
	URL         string `json:"url"`
}

// ErrorResponses

type HttpErrorResponse struct {
	Message          string      `json:"message"`
	Errors           []ErrorItem `json:"errors"`
	DocumentationURL string      `json:"documentation_url"`
	Status           string      `json:"status"`
}

type ErrorItem struct {
	Resource string `json:"resource"`
	Field    string `json:"field"`
	Code     string `json:"code"`
}
