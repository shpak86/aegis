package api

import (
	"aegis/internal/usecase"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// ApiResponseSender
type ApiResponseSender struct {
	w                     http.ResponseWriter
	metricAntibotResponse *prometheus.CounterVec
}

// Send sends response with antibot verdict to the NGINX
func (rs *ApiResponseSender) Send(response *usecase.Response) (err error) {
	encoder := json.NewEncoder(rs.w)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(response)
	if err != nil {
		rs.w.WriteHeader(http.StatusBadRequest)
		return
	}
	slog.Debug("Response", "response", response)
	rs.metricAntibotResponse.WithLabelValues(fmt.Sprintf("%d", response.Code)).Inc()
	return
}

func NewApiResponseSender(w http.ResponseWriter, metricAntibotResponse *prometheus.CounterVec) *ApiResponseSender {
	return &ApiResponseSender{w, metricAntibotResponse}
}
