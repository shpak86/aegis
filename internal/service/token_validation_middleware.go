package service

import (
	"aegis/internal/remap"
	"aegis/internal/usecase"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricTokenValidation = "token_validation"
)

// Validates antibot token for specified paths and restricts access to these paths for clients without the token.
type TokenValidationMiddleware struct {
	ctx                   context.Context
	next                  usecase.Middleware
	tokenManager          usecase.TokenManager
	fingerprinter         usecase.FingerprintCalculator
	protectingEndpoints   map[string]*remap.ReMap[string]
	metricTokenValidation *prometheus.CounterVec
}

func (m *TokenValidationMiddleware) Handle(request *usecase.Request, response usecase.ResponseSender) (err error) {
	method := strings.ToUpper(request.Method)
	var isProtected bool
	if protectingEndpoints, found := m.protectingEndpoints[method]; found {

		_, isProtected = protectingEndpoints.Find(request.Url)
	}
	if !isProtected {
		response.Send(&usecase.Response{
			Code: 0,
			Body: "",
		})
		slog.Debug("Unprotected",
			slog.String("address", request.ClientAddress),
			slog.String("path", request.Url),
			slog.String("method", method),
		)
		return
	}
	token, exists := m.tokenManager.GetRequestToken(request)
	if !exists {
		response.Send(&usecase.Response{
			Code:    http.StatusFound,
			Headers: map[string]string{"Location": "/aegis/challenge/index.html"},
			Body:    "Forbidden",
		})
		m.metricTokenValidation.WithLabelValues("forbidden", request.Url).Inc()
		slog.Debug("Forbidden. Token is absent.",
			slog.String("address", request.ClientAddress),
			slog.String("path", request.Url),
			slog.String("method", method),
		)
		return
	}
	isValidToken := m.tokenManager.Validate(&request.Metadata.Fingerprint, token)
	if !isValidToken {
		response.Send(&usecase.Response{
			Code:    http.StatusFound,
			Headers: map[string]string{"Location": "/aegis/challenge/index.html"},
			Body:    "Forbidden",
		})
		m.metricTokenValidation.WithLabelValues("forbidden", request.Url).Inc()
		slog.Debug("Forbidden. Invalid token.",
			slog.String("address", request.ClientAddress),
			slog.String("fp", request.Metadata.Fingerprint.String()),
			slog.String("token", token),
			slog.String("path", request.Url),
			slog.String("method", method),
		)
		return
	}
	m.metricTokenValidation.WithLabelValues("success", request.Url).Inc()
	m.next.Handle(request, response)
	return
}

func NewTokenValidationMiddleware(
	ctx context.Context,
	next usecase.Middleware,
	tokenManager usecase.TokenManager,
	fingerprinter usecase.FingerprintCalculator,
	protectingEndpoints []usecase.Endpoint,
) *TokenValidationMiddleware {
	m := TokenValidationMiddleware{
		ctx:                 ctx,
		next:                next,
		tokenManager:        tokenManager,
		fingerprinter:       fingerprinter,
		protectingEndpoints: map[string]*remap.ReMap[string]{},
	}

	for _, endpoint := range protectingEndpoints {
		method := strings.ToUpper(endpoint.Method)
		endpointRe, err := regexp.Compile(endpoint.Path)
		if err != nil {
			slog.Error("Failed to compile regexp",
				slog.String("method", method),
				slog.String("path", endpoint.Path),
				slog.String("error", err.Error()),
			)
			continue
		}
		protectingEndpoints, exists := m.protectingEndpoints[method]
		if !exists {
			protectingEndpoints = remap.NewReMap[string]()
			m.protectingEndpoints[method] = protectingEndpoints
		}
		protectingEndpoints.Put(endpointRe, endpoint.Path)
	}
	m.metricTokenValidation = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricTokenValidation,
		},
		[]string{"result", "path"},
	)
	prometheus.MustRegister(m.metricTokenValidation)
	slog.Debug("TokenValidationMiddleware",
		slog.String("protecting", fmt.Sprintf("%v", protectingEndpoints)),
	)
	return &m
}
