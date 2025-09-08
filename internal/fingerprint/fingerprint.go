package fingerprint

import (
	"aegis/internal/usecase"
	"hash/crc32"
	"net"
	"strings"
)

const (
	HeaderUserAgent               = "user-agent"
	HeaderAcceptLanguage          = "accept-language"
	HeaderAcceptEncoding          = "accept-encoding"
	HeaderAccept                  = "accept"
	HeaderSecCHUA                 = "sec-ch-ua"
	HeaderSecCHUAPlatform         = "sec-ch-ua-platform"
	HeaderSecCHUAMobile           = "sec-ch-ua-mobile"
	HeaderSecCHUAFullVersionList  = "sec-ch-ua-full-version-list"
	HeaderDNT                     = "dnt"
	HeaderReferer                 = "referer"
	HeaderCacheControl            = "cache-control"
	HeaderConnection              = "connection"
	HeaderUpgradeInsecureRequests = "upgrade-insecure-requests"
	HeaderHost                    = "host"
	HeaderContentType             = "content-type"
	HeaderCookie                  = "cookie"
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

// Computes a bit-wise XOR for the string.
// Returns 0 for empty string
func xorString(in string) (out byte) {
	if len(in) == 0 {
		return
	}
	out = in[0]
	for i := 1; i < len(in); i++ {
		out = out ^ in[i]
	}
	return
}

// Converts uint32 uint32 to 4-byte slice
func u64tob(val uint32) []byte {
	r := make([]byte, 4)
	for i := uint64(0); i < 4; i++ {
		r[i] = byte((val >> (i * 8)) & 0xff)
	}
	return r
}

// NewHeadersFingerprint creates a HeadersFingerprint by extracting relevant HTTP headers
// from the request and computing a XOR-based hash.
//
// Parameters:
//   - request: The HTTP request containing headers to analyze.
//
// Returns:
//   - *HeadersFingerprint: A new fingerprint object with collected headers and computed hash.
//
// Behavior:
// 1. Extracts headers matching predefined constants (case-insensitive).
// 2. Computes XOR hash for each collected header (7 fields total).
// 3. Combines hashes into a single 7-byte array.
func NewHeadersFingerprint(request *usecase.Request) *HeadersFingerprint {
	f := HeadersFingerprint{}
	for header, value := range request.Headers {
		switch strings.ToLower(header) {
		case HeaderUserAgent:
			f.UserAgent = value
		case HeaderSecCHUA:
			f.SecCHUA = value
		case HeaderSecCHUAPlatform:
			f.SecCHUAPlatform = value
		case HeaderSecCHUAMobile:
			f.SecCHUAMobile = value
		case HeaderSecCHUAFullVersionList:
			f.SecCHUAFullVersionList = value
		case HeaderAcceptLanguage:
			f.AcceptLanguage = value
		case HeaderAcceptEncoding:
			f.AcceptEncoding = value
		}
	}
	f.Hash = []byte{
		xorString(f.UserAgent),
		xorString(f.SecCHUA),
		xorString(f.SecCHUAPlatform),
		xorString(f.SecCHUAMobile),
		xorString(f.SecCHUAFullVersionList),
		xorString(f.AcceptLanguage),
		xorString(f.AcceptEncoding),
	}
	return &f
}

// IpFingerprint represents a client fingerprint derived from their IP address.
type IpFingerprint struct {
	// Parsed IP address as byte slice
	Address []byte
	// CRC32 hash of the raw client address string
	Hash []byte
}

// NewIpFingerprint creates an IpFingerprint by parsing the client's IP address
// and computing a CRC32 hash of the raw address string.
//
// Parameters:
//   - request: The HTTP request containing the client's IP address.
//
// Returns:
//   - *IpFingerprint: A new fingerprint object with parsed IP and hash.
//
// Behavior:
// 1. Parses the client address string to a []byte using net.ParseIP.
// 2. Computes CRC32-IEEE checksum of the raw address string.
// 3. Converts checksum to 4-byte big-endian representation.
func NewIpFingerprint(request *usecase.Request) *IpFingerprint {
	f := IpFingerprint{}
	f.Address = net.ParseIP(request.ClientAddress)
	f.Hash = u64tob(crc32.ChecksumIEEE([]byte(request.ClientAddress)))
	return &f
}
