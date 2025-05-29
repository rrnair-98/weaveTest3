package client

import "net/http"

type GithubClient interface {
	Fetch(url string) (http.Response, error)
}
