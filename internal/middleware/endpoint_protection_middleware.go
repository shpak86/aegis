package middleware

import (
	"aegis/internal/remap"
	"aegis/internal/usecase"
	"context"
	"log/slog"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricEndpointProtection = "endpoint_protection"
)

// Validates antibot token for specified paths and restricts access to these paths for clients without the token.
type EndpointProtectionMiddleware struct {
	ctx                      context.Context
	next                     usecase.Middleware
	tokenManager             usecase.TokenManager
	fingerprinter            usecase.FingerprintCalculator
	protectingEndpoints      map[string]*remap.ReMap[string]
	metricEndpointProtection *prometheus.CounterVec
}

func (m *EndpointProtectionMiddleware) Handle(request *usecase.Request, response usecase.ResponseSender) (err error) {
	method := strings.ToUpper(request.Method)
	var isProtected bool
	if protectingEndpoints, found := m.protectingEndpoints[method]; found {
		_, isProtected = protectingEndpoints.Find(request.Url)
	}
	if !isProtected {
		slog.Debug("Unprotected",
			slog.String("fp", request.Metadata.Fingerprint.String),
			slog.String("path", request.Url),
			slog.String("method", method),
		)
		response.Send(&usecase.ResponseContinue)
		return
	}
	token, exists := m.tokenManager.ExtractToken(request)
	if !exists {
		m.metricEndpointProtection.WithLabelValues("forbidden", request.Url).Inc()
		slog.Info("Token is absent.",
			slog.String("fp", request.Metadata.Fingerprint.String),
			slog.String("path", request.Url),
			slog.String("method", method),
		)
		response.Send(&usecase.ResponseChallenge)
		return
	}
	isValidToken := m.tokenManager.Validate(&request.Metadata.Fingerprint, token)
	if !isValidToken {
		m.metricEndpointProtection.WithLabelValues("forbidden", request.Url).Inc()
		slog.Info("Invalid token.",
			slog.String("fp", request.Metadata.Fingerprint.String),
			slog.String("token", token),
			slog.String("path", request.Url),
			slog.String("method", method),
		)
		response.Send(&usecase.ResponseChallenge)
		return
	}
	m.metricEndpointProtection.WithLabelValues("success", request.Url).Inc()
	m.next.Handle(request, response)
	return
}

func NewEndpointProtectionMiddleware(
	ctx context.Context,
	next usecase.Middleware,
	tokenManager usecase.TokenManager,
	fingerprinter usecase.FingerprintCalculator,
	protections []usecase.Protection,
) *EndpointProtectionMiddleware {
	m := EndpointProtectionMiddleware{
		ctx:                 ctx,
		next:                next,
		tokenManager:        tokenManager,
		fingerprinter:       fingerprinter,
		protectingEndpoints: map[string]*remap.ReMap[string]{},
	}

	for _, protection := range protections {
		method := strings.ToUpper(protection.Method) //todo
		endpointRe, err := regexp.Compile(protection.Path)
		if err != nil {
			slog.Error("Failed to compile regexp",
				slog.String("method", method),
				slog.String("path", protection.Path),
				slog.String("error", err.Error()),
			)
			continue
		}
		protectingEndpoints, exists := m.protectingEndpoints[method]
		if !exists {
			protectingEndpoints = remap.NewReMap[string]()
			m.protectingEndpoints[method] = protectingEndpoints
		}
		protectingEndpoints.Put(endpointRe, protection.Path)
	}
	m.metricEndpointProtection = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricEndpointProtection,
		},
		[]string{"result", "path"},
	)
	prometheus.MustRegister(m.metricEndpointProtection)
	return &m
}
