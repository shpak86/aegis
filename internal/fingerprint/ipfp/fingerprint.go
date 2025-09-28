package ipfp

import (
	"aegis/internal/fingerprint/utils"
	"hash/crc32"
)

// IpFingerprint represents a client fingerprint derived from their IP address.
type IpFingerprint struct {
	// CRC32 hash of the raw client address string
	Hash []byte
}

func Calculate(address string) *IpFingerprint {
	f := IpFingerprint{}
	f.Hash = utils.Uint64ToByte(crc32.ChecksumIEEE([]byte(address)))
	return &f
}
