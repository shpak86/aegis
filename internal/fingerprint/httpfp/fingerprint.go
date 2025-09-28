package httpfp

import (
	"aegis/internal/fingerprint/hfp"
	"aegis/internal/fingerprint/ipfp"
	"aegis/internal/usecase"
	"fmt"
	"slices"
)

func Calculate(r *usecase.HttpFactors) (fp usecase.Fingerprint) {
	ipFingerprint := ipfp.Calculate(r.ClientAddress)
	headersFingerprint := hfp.Calculate(r.Headers)
	hash := slices.Concat(ipFingerprint.Hash, headersFingerprint.Hash)
	fp = usecase.Fingerprint{
		Type:   0,
		Value:  hash,
		String: fmt.Sprintf("%x", hash),
	}
	return
}
