package limiter

import (
	"aegis/internal/remap"
	"aegis/internal/usecase"
	"context"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricRevokeToken = "revoke_token"
)

// limitedCounter tracks request counts for clients with a configured rate limit.
type limitedCounter struct {
	limit   uint32
	counter map[string]*atomic.Uint32
	mu      sync.RWMutex
}

// Increment atomically increases the request count for the specified token.
func (c *limitedCounter) Increment(token string) {
	c.mu.RLock()
	ctr, exists := c.counter[token]
	if exists {
		c.mu.RUnlock()
		ctr.Add(1)
		return
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if ctr, exists = c.counter[token]; exists {
		ctr.Add(1)
		return
	}
	ctr = &atomic.Uint32{}
	c.counter[token] = ctr
	ctr.Add(1)
	slog.Debug("Counter", "token", token, "count", ctr.Load())
}

// RpsLimiter enforces request rate limits per endpoint and revokes tokens for clients exceeding thresholds.
type RpsLimiter struct {
	ctx               context.Context
	endpointCounters  map[string]*remap.ReMap[*limitedCounter]
	mu                sync.RWMutex
	tokenManager      usecase.TokenManager
	metricRevokeToken *prometheus.CounterVec
}

// AddLimit configures a rate limit for the specified HTTP endpoint.
//
// Parameters:
//   - limit: Protection rule containing path, method, and RPS limit.
//
// Behavior:
// 1. Compiles the endpoint path into a regex pattern.
// 2. Associates the regex with a limitedCounter for the HTTP method.
// 3. Logs errors if regex compilation fails.
func (rl *RpsLimiter) AddLimit(limit usecase.Protection) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	method := strings.ToUpper(limit.Method)
	endpointRe, err := regexp.Compile(limit.Path)
	if err != nil {
		slog.Error("Failed to compile regexp",
			slog.String("method", method),
			slog.String("path", limit.Path),
			slog.String("error", err.Error()),
		)
		return
	}
	counters, found := rl.endpointCounters[method]
	if !found {
		counters = remap.NewReMap[*limitedCounter]()
		rl.endpointCounters[limit.Method] = counters
	}
	counters.Put(endpointRe, &limitedCounter{limit: limit.Limit, counter: make(map[string]*atomic.Uint32)})
}

// Count increments the request counter for the specified client token and endpoint.
//
// Parameters:
//   - token: Unique identifier for the client.
//   - path: Requested URL path.
//   - method: HTTP method (e.g., "GET", "POST").
//
// Thread-safety: Uses read locks to minimize contention while accessing shared counters.
func (rl *RpsLimiter) Count(
	token string,
	path string,
	method string,
) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	method = strings.ToUpper(method)
	endpointCounters, found := rl.endpointCounters[method]
	if !found {
		return
	}
	if counters, found := endpointCounters.Find(path); found {
		for _, counter := range counters {
			counter.Increment(token)
		}
	}
}

// revokeByLimits revokes tokens for clients exceeding configured request rate limits.
// This method is typically executed periodically in a background goroutine.
//
// Parameters:
//   - tokenCounters: A pointer to a limitedCounter containing client token usage statistics.
//
// Returns:
//   - uint64: Number of tokens successfully revoked during this invocation.
//
// Functionality:
// 1. Iterates through all tracked tokens in tokenCounters.
// 2. For each token, compares current request count against the configured limit.
// 3. Revokes tokens where usage exceeds the threshold via tokenManager.Revoke().
// 4. Logs debug information for each revoked token including:
//   - Token identifier
//   - Current request rate (rps)
//   - Configured rate limit
func (rl *RpsLimiter) revokeByLimits(tokenCounters *limitedCounter) (revoked uint64) {
	for token, counter := range tokenCounters.counter {
		c := counter.Load()
		if c > tokenCounters.limit {
			slog.Debug("Revoke",
				slog.String("token", token),
				slog.Uint64("rps", uint64(counter.Load())),
				slog.Uint64("limit", uint64(tokenCounters.limit)),
			)
			rl.tokenManager.Revoke(token)
			revoked++
		}
	}
	return
}

// update rotates endpoint counters and revokes tokens for clients exceeding limits.
//
// Behavior:
// 1. Replaces old counters with new empty instances to reset tracking.
// 2. Launches a goroutine to revoke tokens for the previous counters.
// 3. Updates Prometheus metrics with the number of revoked tokens.
func (rl *RpsLimiter) update() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for method, methodCounters := range rl.endpointCounters {
		replacementMethodCounters := remap.NewReMap[*limitedCounter]()
		rl.endpointCounters[method] = replacementMethodCounters
		endpointCounters := methodCounters.Entries()
		for endpointRe, tokensCounters := range endpointCounters {
			go func() {
				revoked := rl.revokeByLimits(tokensCounters)
				rl.metricRevokeToken.WithLabelValues("rps", endpointRe.String()).Add(float64(revoked))
			}()
			replacementMethodCounters.Put(endpointRe, &limitedCounter{limit: tokensCounters.limit, counter: make(map[string]*atomic.Uint32)})
		}
	}
}

// Serve runs a periodic task to update counters and revoke tokens.
//
// This method blocks until the provided context is canceled.
// It uses a 1-second ticker to synchronize updates.
func (rl *RpsLimiter) Serve() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			rl.update()
		case <-rl.ctx.Done():
			return
		}
	}
}

// NewRpsLimiter creates a new RpsLimiter instance with Prometheus metrics integration.
//
// Parameters:
//   - ctx: Context for lifecycle management.
//   - tokenManager: Token manager used to revoke client tokens.
//
// Returns:
//   - *RpsLimiter: Initialized rate limiter with metrics registration.
func NewRpsLimiter(ctx context.Context, tokenManager usecase.TokenManager) *RpsLimiter {
	rl := RpsLimiter{
		ctx:              ctx,
		endpointCounters: map[string]*remap.ReMap[*limitedCounter]{},
		tokenManager:     tokenManager,
	}
	rl.metricRevokeToken = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricRevokeToken,
		},
		[]string{"reason", "path"},
	)
	prometheus.MustRegister(rl.metricRevokeToken)
	return &rl
}
