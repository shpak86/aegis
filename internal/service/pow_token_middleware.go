package service

import (
	"aegis/internal/pow"
	"aegis/internal/usecase"
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetrcTokenRequest      = "token_request"
	MetricChallengeRequest = "challenge_request"
)

// Serves /token endpoint and allows to get challenge and antibot token.
type PowTokenMiddleware struct {
	ctx                    context.Context
	next                   usecase.Middleware
	tokenManager           *pow.PowTokenManager
	fingerprinter          usecase.FingerprintCalculator
	metricTokenRequest     *prometheus.CounterVec
	metricChallengeRequest *prometheus.CounterVec
}

func (m *PowTokenMiddleware) Handle(request *usecase.Request, response usecase.ResponseSender) (err error) {
	request.Metadata.Fingerprint = m.fingerprinter.Calculate(request)
	if request.Url == "/aegis/token" && strings.EqualFold(request.Method, "GET") {
		challenge := m.tokenManager.GetChallenge(&request.Metadata.Fingerprint)
		body := base64.StdEncoding.EncodeToString(challenge)
		response.Send(&usecase.Response{
			Code: http.StatusOK,
			Body: body,
		})
		m.metricChallengeRequest.WithLabelValues("pow").Add(1)
		slog.Debug("Challenge request",
			slog.String("address", request.ClientAddress),
			slog.String("fp", request.Metadata.Fingerprint.String()),
			slog.String("body", body),
		)
	} else if request.Url == "/aegis/token" && strings.EqualFold(request.Method, "POST") {
		var payload []byte
		payload, err = base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			response.Send(&usecase.Response{
				Code: http.StatusUnprocessableEntity,
				Body: "Challenge solution is expected",
			})
			m.metricTokenRequest.WithLabelValues("pow", "unprocessable").Add(1)
			slog.Debug("Unprocessable challenge solution",
				slog.String("address", request.ClientAddress),
				slog.String("fp", request.Metadata.Fingerprint.String()),
				slog.String("solution", request.Body),
			)
			return
		}
		antibotToken, err := m.tokenManager.GetToken(&request.Metadata.Fingerprint, payload[:2], payload[2:])
		if err != nil {
			response.Send(&usecase.Response{
				Code: http.StatusUnauthorized,
				Body: "Wrong solution",
			})
			m.metricTokenRequest.WithLabelValues("pow", "wrong").Add(1)
			slog.Debug("Wrong challenge solution",
				slog.String("address", request.ClientAddress),
				slog.String("fp", request.Metadata.Fingerprint.String()),
				slog.String("solution", request.Body),
			)
			return err
		}
		response.Send(&usecase.Response{
			Code: http.StatusOK,
			Body: antibotToken,
		})
		m.metricTokenRequest.WithLabelValues("pow", "success").Add(1)
		slog.Debug("Token is issued",
			slog.String("address", request.ClientAddress),
			slog.String("fp", request.Metadata.Fingerprint.String()),
			slog.String("token", antibotToken),
		)
	} else if (request.Url == "/aegis/challenge/index.html") && strings.EqualFold(request.Method, "GET") {
		headers := map[string]string{}
		headers["Content-Type"] = "text/html; charset=utf-8"
		response.Send(&usecase.Response{
			Code:    http.StatusOK,
			Headers: headers,
			Body:    pow.IndexHtml,
		})
	} else {
		m.next.Handle(request, response)
	}
	return
}

func NewPowTokenMiddleware(
	ctx context.Context,
	next usecase.Middleware,
	fingerprinter usecase.FingerprintCalculator,
	tokenManager *pow.PowTokenManager,
) *PowTokenMiddleware {
	m := PowTokenMiddleware{
		ctx:           ctx,
		tokenManager:  tokenManager,
		fingerprinter: fingerprinter,
		next:          next,
	}
	m.metricTokenRequest = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetrcTokenRequest,
		},
		[]string{"type", "result"},
	)
	m.metricChallengeRequest = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricChallengeRequest,
		},
		[]string{"type"},
	)
	prometheus.MustRegister(m.metricTokenRequest)
	prometheus.MustRegister(m.metricChallengeRequest)
	slog.Debug("PowTokenMiddleware", slog.Int("complexity", tokenManager.GetComplexity()))
	return &m
}
