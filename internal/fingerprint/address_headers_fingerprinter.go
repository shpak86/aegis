package fingerprint

import (
	"aegis/internal/usecase"
	"fmt"
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
	hash := slices.Concat(ipFingerprint.Hash, headersFingerprint.Hash)
	fp = usecase.Fingerprint{
		Type:   usecase.ADDRESS_HEADERS_FP,
		Value:  hash,
		String: fmt.Sprintf("%x", hash),
	}
	return
}
