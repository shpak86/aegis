package middleware

import (
	"aegis/internal/limiter"
	"aegis/internal/remap"
	"aegis/internal/usecase"
	"log/slog"
	"regexp"
)

type PathProtector struct {
	next                  Middleware[usecase.HttpFactors]
	fingerprintCalculator usecase.FingerprintCalculator[usecase.HttpFactors]
	protected             map[string]*remap.ReMap[string]
	rateLimiter           *limiter.RpsLimiter
	tokenManager          usecase.TokenManager
}

func (m *PathProtector) Handle(request *usecase.RequestContext[usecase.HttpFactors], response ResponseSender) {
	var isProtected bool
	if methodPaths, found := m.protected[request.Factors.Method]; found {
		_, isProtected = methodPaths.Find(request.Factors.Path)
	}

	if !isProtected {
		slog.Debug(
			"Unprotected",
			"fingerprint",
			request.Fingerprint,
			"method",
			request.Factors.Method,
			"path",
			request.Factors.Path,
			"verdict",
			"allow",
		)
		response.Allow()
		return
	}

	if len(request.Factors.Token) == 0 {
		slog.Debug(
			"Token is absent",
			"fingerprint",
			request.Fingerprint,
			"method",
			request.Factors.Method,
			"path",
			request.Factors.Path,
			"verdict",
			"deny",
		)
		response.Deny()
		return
	}

	isValid := m.tokenManager.Validate(&request.Fingerprint, request.Factors.Token)
	if !isValid {
		slog.Debug(
			"Token is invalid",
			"fingerprint",
			request.Fingerprint.String,
			"method",
			request.Factors.Method,
			"path",
			request.Factors.Path,
			"token",
			request.Factors.Token,
			"verdict",
			"deny",
		)
		response.Deny()
		return
	}

	m.rateLimiter.Count(request.Factors.Token, request.Factors.Path, request.Factors.Method)

	if m.next != nil {
		m.next.Handle(request, response)
	} else {
		slog.Debug(
			"Path protector is the last chain",
			"fingerprint",
			request.Fingerprint.String,
			"method",
			request.Factors.Method,
			"path",
			request.Factors.Path,
			"token",
			request.Factors.Token,
			"verdict",
			"allow",
		)
		response.Allow()
	}
}

func (m *PathProtector) Bind(next Middleware[usecase.HttpFactors]) {
	m.next = next
}

func NewPathProtector(
	fingerprintCalculator usecase.FingerprintCalculator[usecase.HttpFactors],
	rateLimiter *limiter.RpsLimiter,
	tokenManager usecase.TokenManager,
	protections []usecase.Protection,

) *PathProtector {
	middleware := PathProtector{
		fingerprintCalculator: fingerprintCalculator,
		rateLimiter:           rateLimiter,
		tokenManager:          tokenManager,
		protected:             map[string]*remap.ReMap[string]{},
	}
	for _, protection := range protections {
		endpointRe, err := regexp.Compile(protection.Path)
		if err != nil {
			slog.Error("Failed to compile regexp",
				slog.String("method", protection.Method),
				slog.String("path", protection.Path),
				slog.String("error", err.Error()),
			)
			continue
		}
		pathPattern, exists := middleware.protected[protection.Method]
		if !exists {
			pathPattern = remap.NewReMap[string]()
			middleware.protected[protection.Method] = pathPattern
		}
		pathPattern.Put(endpointRe, protection.Path)
	}
	return &middleware
}
