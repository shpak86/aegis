package server

import (
	"aegis/internal/middleware"
	"aegis/internal/usecase"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	MetricAntibotResponse = "antibot_response"
)

// API server
type ApiServer struct {
	address               string
	chain                 *middleware.Chain[usecase.HttpFactors]
	server                *http.Server
	fingerprintCalculator usecase.FingerprintCalculator[usecase.HttpFactors]
	tokenManager          usecase.TokenManager
}

func NewApiServer(
	address string,
	chain *middleware.Chain[usecase.HttpFactors],
	fingerprintCalculator usecase.FingerprintCalculator[usecase.HttpFactors],
	tokenManager usecase.TokenManager,
) *ApiServer {
	return &ApiServer{
		address:               address,
		chain:                 chain,
		server:                &http.Server{},
		fingerprintCalculator: fingerprintCalculator,
		tokenManager:          tokenManager,
	}
}

func requestContext(r *http.Request) (rc *usecase.RequestContext[usecase.HttpFactors], err error) {
	factors := usecase.HttpFactors{}
	factors.Body, err = io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Requst body read error", slog.String("error", err.Error()))
		return nil, errors.New("requst body read error")
	}
	defer r.Body.Close()
	factors.Method = r.Header.Get("X-Original-Method")
	factors.Path = r.Header.Get("X-Original-Url")
	factors.ClientAddress = r.Header.Get("X-Original-Addr")
	factors.Path = r.Header.Get("X-Original-Url")
	aegisTokens := r.CookiesNamed("AEGIS_TOKEN")
	if len(aegisTokens) == 1 {
		factors.Token = aegisTokens[0].Value
	}
	factors.Cookies = map[string]string{"AEGIS_TOKEN": factors.Token}
	factors.Headers = make(map[string]string)
	for name, values := range r.Header {
		factors.Headers[name] = strings.Join(values, ",")
	}
	rc = &usecase.RequestContext[usecase.HttpFactors]{Factors: factors}
	return
}

// Serve listens and serves REST API of the Antibot
func (s *ApiServer) Serve() error {
	mux := http.NewServeMux()
	metricAntibotResponse := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: MetricAntibotResponse,
		},
		[]string{"code"},
	)
	prometheus.MustRegister(metricAntibotResponse)
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("GET /aegis/token", func(w http.ResponseWriter, r *http.Request) {
		rc, err := requestContext(r)
		if err != nil {
			slog.Error("Get challenge", "error", err, "context", rc)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		fp := s.fingerprintCalculator.Calculate(&rc.Factors)
		payload, err := s.tokenManager.GetChallenge(&fp)
		if err != nil {
			slog.Error("Get challenge", "error", err, "context", rc)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		slog.Debug("GET /aegis/token", "rc", rc)
		w.Write(payload)
	})

	mux.HandleFunc("POST /aegis/token", func(w http.ResponseWriter, r *http.Request) {
		rc, err := requestContext(r)
		if err != nil {
			slog.Error("Get token", "error", err, "context", rc)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		fp := s.fingerprintCalculator.Calculate(&rc.Factors)
		payload, err := s.tokenManager.GetToken(&fp, rc.Factors.Body)
		if err != nil {
			slog.Error("Get token", "error", err, "context", rc)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		slog.Debug("POST /aegis/token", "rc", rc)
		w.Write([]byte(payload))
	})

	mux.HandleFunc("/aegis/handlers/http", func(w http.ResponseWriter, r *http.Request) {
		rc, err := requestContext(r)
		if err != nil {
			slog.Error("HTTP request", "error", err, "context", rc)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		s.chain.Execute(rc, NewHttpResponseSender(w))
	})
	s.server = &http.Server{
		Addr:         s.address,
		Handler:      mux,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		IdleTimeout:  20 * time.Second,
	}
	return s.server.ListenAndServe()
}

func (s *ApiServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
