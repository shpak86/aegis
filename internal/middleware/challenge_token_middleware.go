package middleware

import (
	"aegis/internal/sha_challenge"
	"aegis/internal/usecase"
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	endpointToken          = "/aegis/token"
	endpointChallenge      = "/aegis/challenge/index.html"
	MetrcTokenRequest      = "token_request"
	MetricChallengeRequest = "challenge_request"
	methodGet              = "GET"
	methodPost             = "POST"
)

// Serves /aegis/token endpoint and allows to get challenge and antibot token.
type ChallengeTokenMiddleware struct {
	ctx                    context.Context
	next                   usecase.Middleware
	tokenManager           usecase.TokenManager
	fingerprinter          usecase.FingerprintCalculator
	metricTokenRequest     *prometheus.CounterVec
	metricChallengeRequest *prometheus.CounterVec
}

func (m *ChallengeTokenMiddleware) Handle(request *usecase.Request, response usecase.ResponseSender) (err error) {
	request.Metadata.Fingerprint = m.fingerprinter.Calculate(request)
	if request.Url == endpointToken && strings.EqualFold(request.Method, methodGet) {
		challenge, _ := m.tokenManager.GetChallenge(&request.Metadata.Fingerprint)
		body := base64.StdEncoding.EncodeToString(challenge)
		response.Send(&usecase.Response{
			Code: http.StatusOK,
			Body: body,
		})
		m.metricChallengeRequest.WithLabelValues("pow").Add(1)
		slog.Info("Challenge request",
			slog.String("fp", request.Metadata.Fingerprint.String),
			slog.String("challenge", body),
		)
	} else if request.Url == endpointToken && strings.EqualFold(request.Method, methodPost) {
		var payload []byte
		payload, err = base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			response.Send(&usecase.Response{
				Code: http.StatusUnprocessableEntity,
				Body: "Challenge solution is expected",
			})
			m.metricTokenRequest.WithLabelValues("pow", "unprocessable").Add(1)
			slog.Info("Unprocessable challenge solution",
				slog.String("fp", request.Metadata.Fingerprint.String),
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
			slog.Info("Wrong challenge solution",
				slog.String("fp", request.Metadata.Fingerprint.String),
				slog.String("solution", request.Body),
			)
			return err
		}
		response.Send(&usecase.Response{
			Code: http.StatusOK,
			Body: antibotToken,
		})
		m.metricTokenRequest.WithLabelValues("pow", "success").Add(1)
		slog.Info("Token is issued",
			slog.String("fp", request.Metadata.Fingerprint.String),
			slog.String("token", antibotToken),
		)
	} else if (request.Url == endpointChallenge) && strings.EqualFold(request.Method, methodGet) {
		headers := map[string]string{
			"Content-Type":           "text/html; charset=utf-8",
			"Cache-Control":          "no-cache, no-store, must-revalidate",
			"X-Frame-Options":        "DENY",
			"X-Content-Type-Options": "nosniff",
		}
		response.Send(&usecase.Response{
			Code:    http.StatusOK,
			Headers: headers,
			Body:    sha_challenge.ShaChallengeIndex,
		})
	} else {
		m.next.Handle(request, response)
	}
	return
}

func NewChallengeTokenMiddleware(
	ctx context.Context,
	next usecase.Middleware,
	fingerprinter usecase.FingerprintCalculator,
	tokenManager usecase.TokenManager,
) *ChallengeTokenMiddleware {
	m := ChallengeTokenMiddleware{
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
	return &m
}
