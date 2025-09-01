package server

import (
	"aegis/internal/middleware"
	"aegis/internal/usecase"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	MetricAntibotResponse = "antibot_response"
)

// API server
type ApiServer struct {
	address string
	chainer *middleware.Chainer
	server  *http.Server
}

func NewApiServer(address string, chainer *middleware.Chainer) *ApiServer {
	return &ApiServer{address: address, chainer: chainer}
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
	mux.HandleFunc("/api/v1/check", func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Requst body read error", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		var request usecase.Request
		if err := json.Unmarshal(payload, &request); err != nil {
			slog.Error("Request unmarshal error", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.chainer.Execute(&request, NewApiResponseSender(w, metricAntibotResponse))
		slog.Debug("Handle", slog.String("request", fmt.Sprintf("%+v", request)))
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
