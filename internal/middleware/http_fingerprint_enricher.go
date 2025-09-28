package middleware

import (
	"aegis/internal/usecase"
	"log/slog"
)

type HttpFingerprintEnricher struct {
	next                  Middleware[usecase.HttpFactors]
	fingerprintCalculator usecase.FingerprintCalculator[usecase.HttpFactors]
}

func (m *HttpFingerprintEnricher) Handle(request *usecase.RequestContext[usecase.HttpFactors], response ResponseSender) {
	request.Fingerprint = m.fingerprintCalculator.Calculate(&request.Factors)
	if m.next != nil {
		m.next.Handle(request, response)
	} else {
		slog.Debug(
			"Fingerprint enricher is the last chain",
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

func (m *HttpFingerprintEnricher) Bind(next Middleware[usecase.HttpFactors]) {
	m.next = next
}

func NewHttpFingerprintEnricher(fingerprintCalculator usecase.FingerprintCalculator[usecase.HttpFactors]) *HttpFingerprintEnricher {
	middleware := HttpFingerprintEnricher{
		fingerprintCalculator: fingerprintCalculator,
	}
	return &middleware
}
