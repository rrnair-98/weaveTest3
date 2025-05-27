package client

import (
	"fmt"
	"net/url"
	"strings"
)

// allowedQualifiers defines a list of supported query qualifiers for filtering search results in specific contexts.
var allowedQualifiers = []string{
	"in:file,path",
	"in:path,file",
	"in:file",
	"in:path",
	"user",
	"org",
	"repo",
	"path",
	"path",
	"language",
	"size",
	"filename",
	"extension",
}

var allowedOperators = []string{
	"AND",
	"OR",
	"NOT",
}

var replacementArray = make([]string, (len(allowedQualifiers)+len(allowedOperators))*2)

func init() {
	initReplacementArray()
}

func initReplacementArray() {
	j := 0
	for i, qualifier := range allowedQualifiers {
		replacementArray[i*2] = qualifier + ":"
		replacementArray[i*2+1] = ""

	}
	opIter := 0
	for i := j; i < j+3; i++ {
		replacementArray[i*2] = allowedOperators[opIter]
		replacementArray[i*2+1] = ""
		opIter++
	}
}

// allowedKeyWords represents a list of query params that this API supports,
// the assumption is that only the qualifier string will be passed to the GRPC server
var allowedKeywords = []string{
	"q",
	"sort",
	"order",
	"per_page",
	"page",
}

const (
	userQueryStringFmt                = "https://api.github.com/search/code?q=%s+%s&per_page=%d&page=%d"
	userQueryDefaultFmt               = "user:%s"
	gitUrlFmtWithPageNumberAndPerPage = "https://api.github.com/search/code?q=%s&per_page=%d&page=%d"
	defaultPerPage                    = 30
	defaultPageNumber                 = 1
)

// queryString is a wrapper around a string that represents a query string for github code search API.
// The query string is a string that can be appended to the gitUrlFmt to form a valid url.
// Content in q=() has to be URI encoded always.
// content after q= cannot exceed 255 chars(in my testing, the API ignores this), without qualifiers and operators being counted
// allowed keywords for code search are defined in allowedKeywords
// allowed qualifiers are defined in allowedQualifiers
// allowed operators are defined in allowedOperators.
// One thing to remember is that the API performs a case-insensitive search.
// Q, sort(default=indexed), order=(asc/desc| default=desc), per_page, page,
// q is required and can not be empty. The others are optional and have default values.
type queryString string

// Validate checks the validity of the query string by verifying it is not empty, complies with operator limits, and length constraints.
func (q queryString) Validate() error {
	if err := q.IsEmpty(); err != nil {
		return err
	}
	if err := q.hasValidNumAndsOrsNots(); err != nil {
		return err
	}
	if err := q.satisfiesQueryLenCapacity(); err != nil {
		return err
	}
	return nil
}

// IsEmpty checks if the query string is empty and returns an error if it is. It ensures the query string is not blank.
func (q queryString) IsEmpty() error {
	if q == "" {
		return fmt.Errorf("query string cant be empty")
	}
	return nil
}

// ToUrlWithMaxPerPage generates a GitHub API search URL with page and per-page parameters based on the query string.
// Returns the URL or an error if the query string validation fails. Defaults to predefined values for invalid inputs.
func (q queryString) ToUrlWithMaxPerPage(pageNumber int, perPage int) (string, error) {
	if err := q.Validate(); err != nil {
		return "", err
	}
	if pageNumber <= 0 {
		pageNumber = defaultPageNumber
	}
	if perPage <= 0 || perPage > 100 {
		perPage = defaultPerPage
	}
	return fmt.Sprintf(gitUrlFmtWithPageNumberAndPerPage, url.QueryEscape(string(q)), perPage, pageNumber), nil
}

// ToUrlWithUser generates a GitHub API search URL with the provided user, page number, and results per page.
// Returns the URL or an error if the query string validation fails.
// Defaults to predefined values for page number and per page if they are invalid.
// If no user is provided, the URL is generated without the user filter.
func (q queryString) ToUrlWithUser(user string, pageNumber int, perPage int) (string, error) {
	if err := q.Validate(); err != nil {
		return "", err
	}
	if pageNumber <= 0 {
		pageNumber = defaultPageNumber
	}
	if perPage <= 0 || perPage > 100 {
		perPage = defaultPerPage
	}
	if user == "" {
		return fmt.Sprintf(gitUrlFmtWithPageNumberAndPerPage, url.QueryEscape(string(q)), perPage, pageNumber), nil
	}
	userStr := fmt.Sprintf(userQueryDefaultFmt, user)
	return fmt.Sprintf(userQueryStringFmt, url.QueryEscape(string(q)), url.QueryEscape(userStr), perPage, pageNumber), nil
}

// hasValidNumAndsOrsNots checks if the query string contains more than 5 boolean operators (AND, OR, NOT).
// Mentioned here https://docs.github.com/en/rest/search/search?apiVersion=2022-11-28#limitations-on-query-length
func (q queryString) hasValidNumAndsOrsNots() error {
	queryStr := strings.ToUpper(string(q))

	// Count occurrences of each operator
	andCount := strings.Count(queryStr, " AND ")
	orCount := strings.Count(queryStr, " OR ")
	notCount := strings.Count(queryStr, " NOT ")
	totalOperators := andCount + orCount + notCount
	if totalOperators > 5 {
		return fmt.Errorf("query string contains %d boolean operators (AND, OR, NOT), exceeding the maximum of 5", totalOperators)
	}
	return nil
}

// satisfiesQueryLenCapacity checks if the query string length exceeds 255 characters after removing specific replacements.
// Returns an error if the processed query exceeds the allowed length; otherwise, it returns nil.
func (q queryString) satisfiesQueryLenCapacity() error {
	// The API ignores this, but it is a good practice to check for it.
	strippedQuery := strings.NewReplacer(replacementArray...).Replace(string(q))
	if len(strippedQuery) > 255 {
		return fmt.Errorf("query string exceeds the maximum of 255 characters")
	}
	return nil
}
