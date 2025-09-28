package fingerprint

import (
	"aegis/internal/fingerprint/httpfp"
	"aegis/internal/usecase"
)

type RequestFingerprintCalculator struct {
}

func (c *RequestFingerprintCalculator) Calculate(factors *usecase.HttpFactors) usecase.Fingerprint {
	return httpfp.Calculate(factors)
}

func NewRequestFingerprintCalculator() *RequestFingerprintCalculator {
	return &RequestFingerprintCalculator{}
}
