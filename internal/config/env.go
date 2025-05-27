package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

var (
	instance *Env
	once     sync.Once
	mu       sync.RWMutex
)

type Env struct {
	GitToken  string     `json:"git_token"`
	Paginator Pagination `json:"pagination"` // if enabled works across all routines except in pagination
}

type Pagination struct {
	Kind          string `json:"kind"`            // either of single or multi, single fetches one page, multi fetches all pages up to max_pages
	MaxPages      int    `json:"max_pages"`       // if FetchAllPages is false, this is the max number of pages to fetch
	PerPage       int    `json:"per_page"`        // Entries per page, max of 100
	FetchAllPages bool   `json:"fetch_all_pages"` // if set ignores MaxPages
	RateLimited   bool   `json:"rate_limited"`    // if set enables the rate limiter
}

func (p *Pagination) IsSinglePage() bool {
	return p.Kind == "single"
}

func (p *Pagination) IsMultiPage() bool {
	return p.Kind == "multi"
}

// InitEnvFromFile initializes the singleton Env instance from the given file path.
// It only loads the config file once, subsequent calls will not reload the config.
// Returns an error if the file cannot be read or parsed.
func InitEnvFromFile(filePath string) error {
	var loadErr error

	once.Do(func() {
		// Read the file content
		data, err := os.ReadFile(filePath)
		if err != nil {
			loadErr = fmt.Errorf("failed to read config file: %w", err)
			return
		}
		var env Env
		if err := json.Unmarshal(data, &env); err != nil {
			loadErr = fmt.Errorf("failed to parse config file: %w", err)
			return
		}
		mu.Lock()
		instance = &env
		mu.Unlock()
	})

	return loadErr
}

func GetEnv() *Env {
	return instance
}
