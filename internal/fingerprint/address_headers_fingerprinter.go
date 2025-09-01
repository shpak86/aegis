package fingerprint

import (
	"aegis/internal/usecase"
	"crypto/sha1"
	"strings"
)

type AddressHeadersFingerprinter struct {
}

func NewAddressHeadersFingerprinter() *AddressHeadersFingerprinter {
	return &AddressHeadersFingerprinter{}
}

// Calculate - calculates client fingerprint using client address and headers.
// Next headers are used:
// - User-Agent
// - Accept
// - Accept-Encoding
// - Sec-Fetch-Dest
// - Sec-Fetch-Mode
// - Sec-Fetch-Site
// - sec-ch-ua
// - sec-ch-ua-mobile
// - sec-ch-ua-platform
func (ahf *AddressHeadersFingerprinter) Calculate(r *usecase.Request) (fp usecase.Fingerprint) {
	b := strings.Builder{}
	b.WriteString(r.ClientAddress)
	if h, exists := r.Headers["User-Agent"]; exists {
		b.WriteString(h)
	} else {
		b.WriteString(".")
	}
	if _, exists := r.Headers["Accept"]; exists {
		b.WriteString("AC")
	} else {
		b.WriteString(".")
	}
	if h, exists := r.Headers["Accept-Encoding"]; exists {
		b.WriteString(h)
	} else {
		b.WriteString(".")
	}
	if h, exists := r.Headers["Accept-Language"]; exists {
		b.WriteString(h)
	} else {
		b.WriteString(".")
	}
	if _, exists := r.Headers["Sec-Fetch-Dest"]; exists {
		b.WriteString("SFD")
	} else {
		b.WriteString(".")
	}
	if _, exists := r.Headers["Sec-Fetch-Mode"]; exists {
		b.WriteString("SFM")
	} else {
		b.WriteString(".")
	}
	if _, exists := r.Headers["Sec-Fetch-Site"]; exists {
		b.WriteString("SFS")
	} else {
		b.WriteString(".")
	}
	if h, exists := r.Headers["sec-ch-ua"]; exists {
		b.WriteString(h)
	} else {
		b.WriteString(".")
	}
	if h, exists := r.Headers["sec-ch-ua-mobile"]; exists {
		b.WriteString(h)
	} else {
		b.WriteString(".")
	}
	if h, exists := r.Headers["sec-ch-ua-platform"]; exists {
		b.WriteString(h)
	} else {
		b.WriteString(".")
	}

	fp = usecase.Fingerprint{
		Type:  usecase.ADDRESS_HEADERS_FP,
		Value: make([]byte, 20),
	}
	hash := sha1.Sum([]byte(b.String()))
	copy(fp.Value, hash[:])
	return
}
