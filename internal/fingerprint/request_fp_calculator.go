package fingerprint

import (
	"aegis/internal/fingerprint/hfp"
	"aegis/internal/fingerprint/ipfp"
	"aegis/internal/usecase"
	"fmt"
	"slices"
)

type RequestFingerprintCalculator struct {
}

func (c *RequestFingerprintCalculator) Calculate(factors *usecase.HttpFactors) usecase.Fingerprint {
	ipFingerprint := ipfp.Calculate(factors.ClientAddress)
	headersFingerprint := hfp.Calculate(factors.Headers)
	// Accept headers depend on context so they useless in the request fingerprint
	hash := slices.Concat(ipFingerprint.Hash, headersFingerprint.Hash[:5])
	fp := usecase.Fingerprint{
		Type:   0,
		Value:  hash,
		String: fmt.Sprintf("%x", hash),
	}
	return fp
}

func NewRequestFingerprintCalculator() *RequestFingerprintCalculator {
	return &RequestFingerprintCalculator{}
}
