package hfp

import (
	"aegis/internal/fingerprint/utils"
	"strings"
)

// HeadersFingerprint represents a client fingerprint derived from HTTP headers.
// Fields:
// - User-Agent
// - Client hints (Sec-CH-UA family)
// - Accept headers
type HeadersFingerprint struct {
	// User-Agent
	UserAgent string
	// Client hints
	SecCHUA                string
	SecCHUAPlatform        string
	SecCHUAMobile          string
	SecCHUAFullVersionList string
	// Accept
	AcceptLanguage string
	AcceptEncoding string
	Hash           []byte
}

func Calculate(headers map[string]string) *HeadersFingerprint {
	f := HeadersFingerprint{}
	if len(headers) != 0 {
		for header, value := range headers {
			switch strings.ToLower(header) {
			case utils.HeaderUserAgent:
				f.UserAgent = value
			case utils.HeaderSecCHUA:
				f.SecCHUA = value
			case utils.HeaderSecCHUAPlatform:
				f.SecCHUAPlatform = value
			case utils.HeaderSecCHUAMobile:
				f.SecCHUAMobile = value
			case utils.HeaderSecCHUAFullVersionList:
				f.SecCHUAFullVersionList = value
			case utils.HeaderAcceptLanguage:
				f.AcceptLanguage = value
			case utils.HeaderAcceptEncoding:
				f.AcceptEncoding = value
			}
		}
		f.Hash = []byte{
			utils.XorString(f.UserAgent),
			utils.XorString(f.SecCHUA),
			utils.XorString(f.SecCHUAPlatform),
			utils.XorString(f.SecCHUAMobile),
			utils.XorString(f.SecCHUAFullVersionList),
			utils.XorString(f.AcceptLanguage),
			utils.XorString(f.AcceptEncoding),
		}
	}
	return &f
}
