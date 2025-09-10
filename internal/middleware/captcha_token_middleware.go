package middleware

import (
	"aegis/internal/captcha"
	"aegis/internal/usecase"
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// Serves /aegis/token endpoint and allows to get challenge and antibot token.
type CaptchaTokenMiddleware struct {
	ctx                    context.Context
	next                   usecase.Middleware
	tokenManager           *captcha.CaptchaTokenManager
	fingerprinter          usecase.FingerprintCalculator
	metricTokenRequest     *prometheus.CounterVec
	metricChallengeRequest *prometheus.CounterVec
}

func (m *CaptchaTokenMiddleware) Handle(request *usecase.Request, response usecase.ResponseSender) (err error) {
	request.Metadata.Fingerprint = m.fingerprinter.Calculate(request)
	if (request.Url == "/aegis/token") && strings.EqualFold(request.Method, methodGet) {
		body, err := m.tokenManager.GetChallenge(&request.Metadata.Fingerprint)
		if err != nil {
			m.metricTokenRequest.WithLabelValues("pow", "unprocessable").Add(1)
			slog.Warn("Unable get captcha",
				slog.String("fp", request.Metadata.Fingerprint.String),
				slog.String("path", request.Url),
				slog.String("error", err.Error()),
			)
			response.Send(&usecase.Response{
				Code: http.StatusInternalServerError,
			})
			return err
		}
		headers := map[string]string{
			"Content-Type":           "text/html; charset=utf-8",
			"Cache-Control":          "no-cache, no-store, must-revalidate",
			"X-Frame-Options":        "DENY",
			"X-Content-Type-Options": "nosniff",
		}
		response.Send(&usecase.Response{
			Code:    http.StatusOK,
			Headers: headers,
			Body:    string(body),
		})
	} else if (request.Url == "/aegis/token") && strings.EqualFold(request.Method, methodPost) {
		token, err := m.tokenManager.GetToken(&request.Metadata.Fingerprint, []byte{}, []byte(request.Body))
		if err != nil {
			slog.Warn("Wrong solution",
				slog.String("fp", request.Metadata.Fingerprint.String),
				slog.String("body", request.Body),
				slog.String("error", err.Error()),
			)
			m.metricTokenRequest.WithLabelValues("pow", "wrong").Add(1)
			response.Send(&usecase.Response{
				Code: http.StatusUnprocessableEntity,
				Headers: map[string]string{
					"Content-Type": "text/html; charset=utf-8",
				},
			})
			return err
		}
		m.metricTokenRequest.WithLabelValues("pow", "success").Add(1)
		slog.Info("Token is issued",
			slog.String("fp", request.Metadata.Fingerprint.String),
			slog.String("token", token),
		)
		headers := map[string]string{
			"Content-Type":           "text/html; charset=utf-8",
			"Cache-Control":          "no-cache, no-store, must-revalidate",
			"X-Frame-Options":        "DENY",
			"X-Content-Type-Options": "nosniff",
		}
		response.Send(&usecase.Response{
			Code:    http.StatusOK,
			Headers: headers,
			Body:    token,
		})
	} else {
		m.next.Handle(request, response)
	}
	return
}

func NewCaptchaTokenMiddleware(
	ctx context.Context,
	next usecase.Middleware,
	fingerprinter usecase.FingerprintCalculator,
	tokenManager *captcha.CaptchaTokenManager,
) *CaptchaTokenMiddleware {
	m := CaptchaTokenMiddleware{
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
