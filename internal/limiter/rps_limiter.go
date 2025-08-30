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

// Container of counters for the specified paths
type limitedCounter struct {
	limit   uint32
	counter map[string]*atomic.Uint32
	mu      sync.RWMutex
}

// Increments token for the specified path
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

// Counts clients requests and revokes tokens of the clients who are exceed a limit.
type RpsLimiter struct {
	ctx               context.Context
	endpointCounters  map[string]*remap.ReMap[*limitedCounter]
	mu                sync.RWMutex
	tokenManager      usecase.TokenManager
	metricRevokeToken *prometheus.CounterVec
}

// Adds RPS limit
func (rl *RpsLimiter) AddLimit(limit usecase.Limit) {
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

// Count client request. Client is represented by token.
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
	counters, _ := endpointCounters.Find(path)
	for _, counter := range counters {
		counter.Increment(token)
	}
}

// Revoke tokens of the clients who exceed limits
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

// Update counters and revoke tokens of the clients who exceed limits
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

// Serve counters
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
