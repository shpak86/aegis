package fingerprint

import (
	"aegis/internal/usecase"
	"bytes"
	"hash/crc32"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test suite for fingerprint package
type FingerprintTestSuite struct {
	suite.Suite
}

func (suite *FingerprintTestSuite) SetupTest() {
}

func (suite *FingerprintTestSuite) TestXorString() {
	assert.Equal(suite.T(), byte(0), xorString(""))
	assert.Equal(suite.T(), byte('a'), xorString("a"))
	assert.Equal(suite.T(), byte('a'^'b'^'c'), xorString("abc"))
}

func (suite *FingerprintTestSuite) TestU64ToB() {
	assert.Equal(suite.T(), []byte{0, 0, 0, 0}, u64tob(0))
	assert.Equal(suite.T(), []byte{0x78, 0x56, 0x34, 0x12}, u64tob(0x12345678))
}

func (suite *FingerprintTestSuite) TestNewHeadersFingerprint_AllHeadersPresent() {

	headers := map[string]string{
		"User-Agent":                  "Mozilla/5.0",
		"Sec-CH-UA":                   "Chrome",
		"Sec-CH-UA-Platform":          "Windows",
		"Sec-CH-UA-Mobile":            "?0",
		"Sec-CH-UA-Full-Version-List": "Full Version",
		"Accept-Language":             "en-US",
		"Accept-Encoding":             "gzip",
	}

	cookies := map[string]string{
		"User-Id": "1",
	}
	mockRequest := &usecase.Request{
		ClientAddress: "192.42.49.48",
		Method:        "POST",
		Url:           "/",
		Body:          "content",
		Headers:       headers,
		Cookies:       cookies,
		Metadata:      usecase.Meta{},
	}

	fp := NewHeadersFingerprint(mockRequest)

	assert.Equal(suite.T(), "Mozilla/5.0", fp.UserAgent)
	assert.Equal(suite.T(), "Chrome", fp.SecCHUA)
	assert.Equal(suite.T(), "Windows", fp.SecCHUAPlatform)
	assert.Equal(suite.T(), "?0", fp.SecCHUAMobile)
	assert.Equal(suite.T(), "Full Version", fp.SecCHUAFullVersionList)
	assert.Equal(suite.T(), "en-US", fp.AcceptLanguage)
	assert.Equal(suite.T(), "gzip", fp.AcceptEncoding)

	expectedHash := []byte{
		xorString("Mozilla/5.0"),
		xorString("Chrome"),
		xorString("Windows"),
		xorString("?0"),
		xorString("Full Version"),
		xorString("en-US"),
		xorString("gzip"),
	}
	assert.Equal(suite.T(), expectedHash, fp.Hash)
}

func (suite *FingerprintTestSuite) TestNewHeadersFingerprint_MissingHeaders() {
	headers := map[string]string{}
	cookies := map[string]string{}
	mockRequest := &usecase.Request{
		ClientAddress: "192.42.49.48",
		Method:        "POST",
		Url:           "/",
		Body:          "content",
		Headers:       headers,
		Cookies:       cookies,
		Metadata:      usecase.Meta{},
	}

	fp := NewHeadersFingerprint(mockRequest)

	assert.Empty(suite.T(), fp.UserAgent)
	assert.Empty(suite.T(), fp.SecCHUA)
	assert.Empty(suite.T(), fp.SecCHUAPlatform)
	assert.Empty(suite.T(), fp.SecCHUAMobile)
	assert.Empty(suite.T(), fp.SecCHUAFullVersionList)
	assert.Empty(suite.T(), fp.AcceptLanguage)
	assert.Empty(suite.T(), fp.AcceptEncoding)

	expectedHash := []byte{0, 0, 0, 0, 0, 0, 0}
	assert.Equal(suite.T(), expectedHash, fp.Hash)
}

func (suite *FingerprintTestSuite) TestNewIpFingerprint_ValidIPv4() {
	headers := map[string]string{}
	cookies := map[string]string{}
	mockRequest := &usecase.Request{
		ClientAddress: "192.168.1.1",
		Method:        "POST",
		Url:           "/",
		Body:          "content",
		Headers:       headers,
		Cookies:       cookies,
		Metadata:      usecase.Meta{},
	}

	fp := NewIpFingerprint(mockRequest)

	require.NotNil(suite.T(), fp.Address)
	assert.True(suite.T(), bytes.Equal(fp.Address, net.ParseIP("192.168.1.1")))

	crc := crc32.ChecksumIEEE([]byte("192.168.1.1"))
	expectedHash := u64tob(crc)
	assert.Equal(suite.T(), expectedHash, fp.Hash)
}

func (suite *FingerprintTestSuite) TestNewIpFingerprint_ValidIPv6() {
	headers := map[string]string{}
	cookies := map[string]string{}
	mockRequest := &usecase.Request{
		ClientAddress: "::1",
		Method:        "POST",
		Url:           "/",
		Body:          "content",
		Headers:       headers,
		Cookies:       cookies,
		Metadata:      usecase.Meta{},
	}

	fp := NewIpFingerprint(mockRequest)

	require.NotNil(suite.T(), fp.Address)
	assert.True(suite.T(), bytes.Equal(fp.Address, net.ParseIP("::1")))

	crc := crc32.ChecksumIEEE([]byte("::1"))
	expectedHash := u64tob(crc)
	assert.Equal(suite.T(), expectedHash, fp.Hash)
}

func (suite *FingerprintTestSuite) TestNewIpFingerprint_InvalidAddress() {
	headers := map[string]string{}
	cookies := map[string]string{}
	mockRequest := &usecase.Request{
		ClientAddress: "invalid-ip",
		Method:        "POST",
		Url:           "/",
		Body:          "content",
		Headers:       headers,
		Cookies:       cookies,
		Metadata:      usecase.Meta{},
	}

	fp := NewIpFingerprint(mockRequest)

	assert.Nil(suite.T(), fp.Address)
	crc := crc32.ChecksumIEEE([]byte("invalid-ip"))
	expectedHash := u64tob(crc)
	assert.Equal(suite.T(), expectedHash, fp.Hash)
}

func TestFingerprintSuite(t *testing.T) {
	suite.Run(t, new(FingerprintTestSuite))
}
