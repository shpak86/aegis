package middleware

import (
	"aegis/internal/limiter"
	"aegis/internal/usecase"
	"context"
)

// Counts requests to the specified paths.
type RateLimitMiddleware struct {
	ctx          context.Context
	next         usecase.Middleware
	rateLimiter  *limiter.RpsLimiter
	tokenManager usecase.TokenManager
}

func (m *RateLimitMiddleware) Handle(request *usecase.Request, response usecase.ResponseSender) (err error) {
	token, exists := m.tokenManager.GetRequestToken(request)
	if !exists {
		response.Send(&usecase.ResponseChallenge)
		return
	}
	// Just counting, revoking is inside rateLimiter
	m.rateLimiter.Count(token, request.Url, request.Method)
	if m.next != nil {
		m.next.Handle(request, response)
	} else {
		response.Send(&usecase.ResponseContinue)
	}
	return
}

func NewRateLimitMiddleware(
	ctx context.Context,
	next usecase.Middleware,
	tokenManager usecase.TokenManager,
	rateLimiter *limiter.RpsLimiter,
) *RateLimitMiddleware {
	m := RateLimitMiddleware{
		ctx:          ctx,
		next:         next,
		rateLimiter:  rateLimiter,
		tokenManager: tokenManager,
	}
	return &m
}
