package client

import (
	"context"
	"go.uber.org/zap"
	"sync"
	"time"
)

var (
	rateLimiterInstance *RateLimiter
	rateLimiterOnce     sync.Once
	rateLimiterMu       sync.RWMutex
)

// RateLimiter manages API request rate limiting
type RateLimiter struct {
	mu           sync.Mutex
	requestTimes []time.Time
	maxRequests  int
	timeWindow   time.Duration
	requestChan  chan struct{}
	releaseChan  chan struct{}
	done         chan struct{}
	wg           sync.WaitGroup
	logger       *zap.Logger
}

// GetRateLimiter returns the singleton instance of the rate limiter
func GetRateLimiter() *RateLimiter {
	rateLimiterMu.RLock()
	if rateLimiterInstance != nil {
		defer rateLimiterMu.RUnlock()
		return rateLimiterInstance
	}
	rateLimiterMu.RUnlock()

	return rateLimiterInstance
}

// InitRateLimiter initializes the rate limiter with custom parameters
// This should be called early in your application startup
func InitRateLimiter(maxRequests int, timeWindow time.Duration, logger *zap.Logger) {
	rateLimiterOnce.Do(func() {
		rateLimiterMu.Lock()
		defer rateLimiterMu.Unlock()

		rateLimiterInstance = newRateLimiter(maxRequests, timeWindow, logger)
	})
}

// newRateLimiter creates a new rate limiter instance
func newRateLimiter(maxRequests int, timeWindow time.Duration, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		maxRequests:  maxRequests,
		timeWindow:   timeWindow,
		requestTimes: make([]time.Time, 0, maxRequests),
		requestChan:  make(chan struct{}),
		releaseChan:  make(chan struct{}),
		done:         make(chan struct{}),
		logger:       logger,
	}

	logger.Debug("starting worker goroutine")
	// Start the worker goroutine
	rl.wg.Add(1)
	go rl.worker()

	return rl
}

// worker processes incoming request tokens and manages the rate limiting
func (rl *RateLimiter) worker() {
	defer rl.wg.Done()

	for {
		select {
		case <-rl.requestChan:
			rl.logger.Debug("waiting for chan")
			delay := rl.calculateDelay()
			if delay > 0 {
				time.Sleep(delay)
			}

			rl.logger.Debug("adding request time: ", zap.Time("time", time.Now()))
			rl.mu.Lock()
			now := time.Now()
			rl.requestTimes = append(rl.requestTimes, now)

			rl.logger.Debug("removing old request times: ", zap.Int("numOldTimes", len(rl.requestTimes)))
			cutoff := now.Add(-rl.timeWindow)
			i := 0
			for i < len(rl.requestTimes) && rl.requestTimes[i].Before(cutoff) {
				i++
			}
			if i > 0 {
				rl.requestTimes = rl.requestTimes[i:]
			}
			rl.mu.Unlock()

			rl.logger.Debug("signalling to chan for routine to start execution")
			rl.releaseChan <- struct{}{}

		case <-rl.done:
			return
		}
	}
}

// calculateDelay determines how long to wait before allowing the next request
func (rl *RateLimiter) calculateDelay() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.requestTimes) < rl.maxRequests {
		return 0
	}

	// If we've reached the max requests, calculate how long until the oldest
	// request falls outside the time window
	oldestAllowed := time.Now().Add(-rl.timeWindow)
	if rl.requestTimes[0].After(oldestAllowed) {
		rl.logger.Debug("oldest request is still within the time window, returning diff")
		// The oldest request is still within the time window
		return rl.requestTimes[0].Add(rl.timeWindow).Sub(time.Now())
	}
	rl.logger.Debug("oldest request is outside the time window, returning 0")
	return 0
}

// Wait blocks until a request can be made according to rate limits
func (rl *RateLimiter) Wait() {
	rl.requestChan <- struct{}{}
	<-rl.releaseChan
}

// WaitWithContext blocks until a request can be made or context is canceled
func (rl *RateLimiter) WaitWithContext(ctx context.Context) error {
	select {
	case rl.requestChan <- struct{}{}:
		select {
		case <-rl.releaseChan:
			return nil
		case <-ctx.Done():
			rl.logger.Debug("draining release chan")
			// Need to drain the request to avoid leaking it
			go func() {
				<-rl.releaseChan
			}()
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ExecuteWithRateLimit executes the given function while respecting rate limits
func (rl *RateLimiter) ExecuteWithRateLimit(ctx context.Context, fn func() error) error {
	err := rl.WaitWithContext(ctx)
	if err != nil {
		return err
	}

	return fn()
}

// Close shuts down the rate limiter
func (rl *RateLimiter) Close() {
	close(rl.done)
	rl.wg.Wait()
}

// ShutdownRateLimiter closes the singleton rate limiter instance
func ShutdownRateLimiter() {
	rateLimiterMu.Lock()
	defer rateLimiterMu.Unlock()

	if rateLimiterInstance != nil {
		rateLimiterInstance.Close()
		rateLimiterInstance = nil
	}

	// Reset the once so it can be initialized again if needed
	rateLimiterOnce = sync.Once{}
}
