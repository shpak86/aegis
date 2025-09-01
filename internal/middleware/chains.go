package middleware

import (
	"aegis/internal/fingerprint"
	"aegis/internal/limiter"
	"aegis/internal/token_manager"
	"aegis/internal/usecase"
	"context"
)

// DefaultProtectionChain creates and returns a pre-configured middleware chain for request protection,
// combining Proof of Work (PoW) token validation, rate limiting, and endpoint-specific protections.
//
// Parameters:
//   - ctx: Context for lifecycle management and cancellation.
//   - complexity: Target computational complexity for PoW token generation.
//   - protections: List of protection rules defining paths/methods to secure.
//
// Returns:
//   - *Chainer: A middleware chain containing:
//     1. PoW token middleware for client authentication.
//     2. Token validation middleware to verify tokens and fingerprints.
//     3. Rate-limiting middleware to enforce request quotas.
//
// The chain applies protections in this order:
// 1. Token generation (PoW) → 2. Token validation → 3. Rate limiting.
// Rate limits are configured using the provided protections list.
func DefaultProtectionChain(ctx context.Context, complexity string, protections []usecase.Protection) *Chainer {

	fingerprinter := fingerprint.NewAddressHeadersFingerprinter()

	var complexityLevel int
	switch complexity {
	case "easy":
		complexityLevel = 1
	case "medium":
		complexityLevel = 2
	case "hard":
		complexityLevel = 3
	default:
		complexityLevel = 2
	}
	tokenManager := token_manager.NewShaChallengeTokenManager(complexityLevel)

	rateLimiter := limiter.NewRpsLimiter(ctx, tokenManager)
	for _, p := range protections {
		rateLimiter.AddLimit(usecase.Protection(p))
	}
	go rateLimiter.Serve()

	rateLimitMiddleware := NewRateLimitMiddleware(ctx, nil, tokenManager, rateLimiter)
	endpointProtectionMiddleware := NewEndpointProtectionMiddleware(ctx, rateLimitMiddleware, tokenManager, fingerprinter, protections)
	tokenMiddleware := NewTokenMiddleware(ctx, endpointProtectionMiddleware, fingerprinter, tokenManager)

	chainer := NewChainer(ctx)
	chainer.Add(tokenMiddleware)
	chainer.Add(endpointProtectionMiddleware)
	chainer.Add(rateLimitMiddleware)
	return chainer
}
