package utils

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

// Computes a bit-wise XOR for the string.
// Returns 0 for empty string
func XorString(in string) (out byte) {
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
func Uint64ToByte(val uint32) (r []byte) {
	r = make([]byte, 4)
	for i := uint64(0); i < 4; i++ {
		r[i] = byte((val >> (i * 8)) & 0xff)
	}
	return
}
