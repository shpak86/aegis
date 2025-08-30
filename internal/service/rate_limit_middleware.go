package service

import (
	"aegis/internal/limiter"
	"aegis/internal/usecase"
	"context"
	"log/slog"
)

// Counts requests to the specified paths.
type RateLimitMiddleware struct {
	ctx          context.Context
	next         usecase.Middleware
	rateLimiter  *limiter.RpsLimiter
	tokenManager usecase.TokenManager
}

func (m *RateLimitMiddleware) Handle(request *usecase.Request, response usecase.ResponseSender) (err error) {
	token, _ := m.tokenManager.GetRequestToken(request)
	m.rateLimiter.Count(token, request.Url, request.Method)
	if m.next != nil {
		m.next.Handle(request, response)
	} else {
		response.Send(&usecase.Response{
			Code: 0,
			Body: "",
		})
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

	go m.rateLimiter.Serve()
	slog.Debug("RateLimitMiddleware",
		slog.String("protecting", "true"),
	)
	return &m
}
