package service

import (
	"aegis/internal/fingerprint"
	"aegis/internal/limiter"
	"aegis/internal/pow"
	"aegis/internal/usecase"
	"context"
)

// Build a chain of handlers
func BasicAntibotChainer(ctx context.Context, complexity int, protections []usecase.Protection) *Chainer {

	fingerprinter := fingerprint.NewAddressHeadersFingerprinter()
	tokenManager := pow.NewPowTokenManager(complexity)
	rateLimiter := limiter.NewRpsLimiter(ctx, tokenManager)
	endpoints := []usecase.Endpoint{}
	for _, p := range protections {
		rateLimiter.AddLimit(usecase.Limit(p))
		endpoints = append(endpoints, usecase.Endpoint{Path: p.Path, Method: p.Method})
	}

	rateLimitMiddleware := NewRateLimitMiddleware(ctx, nil, tokenManager, rateLimiter)

	tokenValidationMiddleware := NewTokenValidationMiddleware(ctx, rateLimitMiddleware, tokenManager, fingerprinter, endpoints)

	tokenMiddleware := NewPowTokenMiddleware(ctx, tokenValidationMiddleware, fingerprinter, tokenManager)

	chainer := NewChainer(ctx)
	chainer.Add(tokenMiddleware)
	chainer.Add(tokenValidationMiddleware)
	chainer.Add(rateLimitMiddleware)
	return chainer
}
