package fingerprint

import (
	"aegis/internal/usecase"
	"slices"
)

type AddressHeadersFingerprinter struct {
}

func NewAddressHeadersFingerprinter() *AddressHeadersFingerprinter {
	return &AddressHeadersFingerprinter{}
}

// Calculate - calculates client fingerprint based on headers fingerprint and IP fingerprint.
func (ahf *AddressHeadersFingerprinter) Calculate(r *usecase.Request) (fp usecase.Fingerprint) {
	ipFingerprint := NewIpFingerprint(r)
	headersFingerprint := NewHeadersFingerprint(r)
	fp = usecase.Fingerprint{
		Type:  usecase.ADDRESS_HEADERS_FP,
		Value: slices.Concat(ipFingerprint.Hash, headersFingerprint.Hash),
	}
	return
}
