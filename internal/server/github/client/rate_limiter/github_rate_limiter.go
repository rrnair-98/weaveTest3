package rate_limiter

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// GitHubRateLimiter manages rate limiting specifically for GitHub API
type GitHubRateLimiter struct {
	// Configuration
	logger              *zap.Logger
	httpClient          *http.Client
	firstHitUrl         string
	githubAuthToken     string
	blockingRateLimited bool // if set, the ExecuteRequest method will block until the rate limit is reset

	mu          sync.RWMutex
	tokens      int       // current number of available tokens
	maxTokens   int       // maximum number of tokens (from X-RateLimit-Limit)
	resetTime   time.Time // wen the rate limit will reset (from X-RateLimit-Reset)
	initialized bool      // whether the limiter has been initialized with GitHub's limits

	requestChan  chan struct{}     // Channel for requesting a token
	responseChan chan tokenResult  // Channel for receiving token results
	updateChan   chan updateParams // Channel for updating rate limit info
	done         chan struct{}     // Channel for shutdown signal

	initOnce sync.Once
	wg       sync.WaitGroup
}

const (
	firstRequestUrl = "https://api.github.com/search/code?q=Q&per_page=1&page=1"
	bearerFmt       = "Bearer %s" // tokenResult represents the result of a token request
)

type tokenResult struct {
	canProceed bool
	waitTime   time.Duration
	err        error
}

// updateParams contains information for updating rate limits
type updateParams struct {
	remaining int
	limit     int
	resetTime time.Time
}

var (
	githubLimiter     *GitHubRateLimiter
	githubLimiterOnce sync.Once
	githubLimiterMu   sync.RWMutex
)

// NewGitHubRateLimiter creates a new GitHub rate limiter
func NewGitHubRateLimiter(logger *zap.Logger, githubToken string, firstHitUrl string, blockingRateLimited bool) *GitHubRateLimiter {
	if firstHitUrl == "" {
		firstHitUrl = firstRequestUrl
	}
	limiter := &GitHubRateLimiter{
		// configuration
		logger:              logger,
		httpClient:          &http.Client{Timeout: 10 * time.Second},
		firstHitUrl:         "https://api.github.com/search/code?%s",
		githubAuthToken:     githubToken,
		blockingRateLimited: blockingRateLimited,

		tokens:      0,                         // start with 0 tokens until we get the real count
		maxTokens:   10,                        // gitHub's default limit, will be updated
		resetTime:   time.Now().Add(time.Hour), // conservative default
		initialized: false,

		// channels
		requestChan:  make(chan struct{}),
		responseChan: make(chan tokenResult),
		updateChan:   make(chan updateParams, 10), // Buffered to prevent blocking
		done:         make(chan struct{}),
	}

	limiter.wg.Add(1)
	go limiter.worker()

	return limiter
}

// GetGitHubRateLimiter returns the singleton instance of the GitHub rate limiter
func GetGitHubRateLimiter() *GitHubRateLimiter {
	githubLimiterMu.RLock()
	if githubLimiter != nil {
		defer githubLimiterMu.RUnlock()
		return githubLimiter
	}
	githubLimiterMu.RUnlock()

	return githubLimiter
}

// InitGitHubRateLimiter initializes the singleton GitHub rate limiter
func InitGitHubRateLimiter(logger *zap.Logger, githubToken string, blockingRateLimited bool) {
	githubLimiterOnce.Do(func() {
		githubLimiterMu.Lock()
		defer githubLimiterMu.Unlock()

		githubLimiter = NewGitHubRateLimiter(logger, githubToken, firstRequestUrl, blockingRateLimited)
		githubLimiter.blockingRateLimited = blockingRateLimited
	})
}

// worker is the main goroutine that handles token requests and updates
func (rl *GitHubRateLimiter) worker() {
	defer rl.wg.Done()

	// try to initialize rate limits if not already done
	rl.tryInitializeOnce()

	for {
		select {
		case <-rl.requestChan:
			// Handle token request
			rl.mu.RLock()
			canProceed, waitTime := rl.checkTokenAvailability()
			rl.mu.RUnlock()

			if canProceed {
				rl.mu.Lock()
				rl.tokens-- // Decrement token count
				rl.mu.Unlock()
				rl.responseChan <- tokenResult{canProceed: true, waitTime: 0}
			} else {
				// Can't proceed now, return wait time
				rl.responseChan <- tokenResult{canProceed: false, waitTime: waitTime}
			}

		case update := <-rl.updateChan:
			rl.mu.Lock()
			if update.limit > 0 {
				rl.maxTokens = update.limit
			}
			// update tokens if it's a valid value
			if update.remaining >= 0 {
				rl.tokens = update.remaining
			}
			// updating reset time if it's in the future
			if update.resetTime.After(time.Now()) {
				rl.resetTime = update.resetTime.Add(2 * time.Second)
			}
			rl.initialized = true
			rl.mu.Unlock()

			rl.logger.Debug("updated rate limits",
				zap.Int("tokens", update.remaining),
				zap.Int("maxTokens", update.limit),
				zap.Time("resetTime", update.resetTime))

		case <-rl.done:
			return
		}
	}
}

// checkTokenAvailability checks if a token is available or when one will be
func (rl *GitHubRateLimiter) checkTokenAvailability() (bool, time.Duration) {
	now := time.Now()

	// if we have tokens available, the caller doesnt have to wait
	if rl.tokens > 0 {
		return true, 0
	}

	// if we've passed the reset time, the caller doesnt have to wait
	if now.After(rl.resetTime) {
		return true, 0
	}

	waitTime := time.Until(rl.resetTime) + 2000*time.Millisecond // adding buffer of 2 seconds
	rl.logger.Debug("no tokens available, returning wait time", zap.Duration("waitTime", waitTime))
	return false, waitTime
}

// tryInitializeOnce calls tryInitialize by wrapping it in a once block, attempts to initialize the rate limiter from GitHub API
func (rl *GitHubRateLimiter) tryInitializeOnce() {
	rl.initOnce.Do(func() {
		rl.tryInitialize()
	})
}

// try
func (rl *GitHubRateLimiter) tryInitialize() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", rl.firstHitUrl, nil)
	if err != nil {
		rl.logger.Warn("failed to create rate limit request", zap.Error(err))
		return
	}

	resp, err := rl.httpClient.Do(req)
	if err != nil {
		rl.logger.Warn("failed to execute rate limit request", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	rl.parseAndUpdateHeaders(resp.Header)
}

// setHttpClient sets the HTTP client to use for rate limiting requests. To be used only for tests.
func (rl *GitHubRateLimiter) setHttpClient(client *http.Client) {
	rl.httpClient = client
}

// parseAndUpdateHeaders extracts rate limit info from headers and updates the limiter
func (rl *GitHubRateLimiter) parseAndUpdateHeaders(headers http.Header) {
	var remaining, limit int
	var resetTime time.Time

	// parse X-RateLimit-Remaining
	if remainingHeader := headers.Get("X-RateLimit-Remaining"); remainingHeader != "" {
		if val, err := strconv.Atoi(remainingHeader); err == nil {
			remaining = val - 1 // subtract 1 to since github uses an indexed system from 10 ie the token we just used
		} else {
			rl.logger.Warn("invalid X-RateLimit-Remaining header", zap.String("value", remainingHeader), zap.Error(err))
		}
	}

	// parse X-RateLimit-Limit
	if limitHeader := headers.Get("X-RateLimit-Limit"); limitHeader != "" {
		if val, err := strconv.Atoi(limitHeader); err == nil {
			limit = val
		} else {
			rl.logger.Warn("invalid X-RateLimit-Limit header", zap.String("value", limitHeader), zap.Error(err))
		}
	}

	// parse X-RateLimit-Reset
	if resetHeader := headers.Get("X-RateLimit-Reset"); resetHeader != "" {
		if resetUnix, err := strconv.ParseInt(resetHeader, 10, 64); err == nil {
			resetTime = time.Unix(resetUnix, 0)
		} else {
			rl.logger.Warn("invalid X-RateLimit-Reset header", zap.String("value", resetHeader), zap.Error(err))
		}
	}
	rl.logger.Debug("parsed rate limit headers", zap.Int("remaining", remaining), zap.Int("limit", limit), zap.Time("resetTime", resetTime))

	// send update to worker
	if limit > 0 || remaining >= 0 || !resetTime.IsZero() {
		select {
		case rl.updateChan <- updateParams{
			remaining: remaining,
			limit:     limit,
			resetTime: resetTime,
		}:
		default:
			rl.logger.Warn("update channel full, rate limit update skipped")
		}
	}
}

// WaitWithContext waits for rate limiting with context cancellation support
func (rl *GitHubRateLimiter) WaitWithContext(ctx context.Context) error {
	for {
		select {
		case rl.requestChan <- struct{}{}:
			// request sent, wait for response
			rl.logger.Debug("requested token", zap.Time("now", time.Now()))
			select {
			case result := <-rl.responseChan:
				if result.canProceed {
					return nil
				}

				// reed to wait, set up a timer
				rl.logger.Debug("rate limit reached, waiting", zap.Duration("waitTime", result.waitTime))
				timer := time.NewTimer(result.waitTime)
				select {
				case <-timer.C:
					// timer completed, try again
					continue
				case <-ctx.Done():
					// context canceled during wait
					if !timer.Stop() {
						<-timer.C
					}
					return ctx.Err()
				}

			case <-ctx.Done():
				// context canceled while waiting for response
				// need to read the response to avoid blocking the worker
				go func() { <-rl.responseChan }()
				return ctx.Err()
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// UpdateFromResponse updates the rate limiter from a response
func (rl *GitHubRateLimiter) UpdateFromResponse(resp *http.Response) {
	rl.parseAndUpdateHeaders(resp.Header)
}

// Close shuts down the rate limiter
func (rl *GitHubRateLimiter) Close() {
	close(rl.done)
	rl.wg.Wait()
}

// ExecuteRequest executes an HTTP request with rate limiting
func (rl *GitHubRateLimiter) ExecuteRequest(req *http.Request) (*http.Response, error) {

	if rl.blockingRateLimited {
		err := rl.WaitWithContext(req.Context())
		if err != nil {
			return nil, err
		}
	}

	// letting request go through since values have be updated, if not, waiting for the next request
	// TODO: spawn another routine that calls tryInitialize() every time waitTimer ends
	req.Header.Add("Authorization", fmt.Sprintf(bearerFmt, rl.githubAuthToken))
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	rl.logger.Debug("wait over, executing request", zap.String("url", req.URL.String()))
	resp, err := rl.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	rl.UpdateFromResponse(resp)

	return resp, nil
}
