package usecase

import "fmt"

const (
	VerdictContinue = iota
	VerdictBan
	VerdictAllow
)

type TokenManager interface {
	GetRequestToken(*Request) (string, bool)
	Validate(*Fingerprint, string) bool
	Revoke(string) bool
}

type ResponseSender interface {
	Send(*Response) error
}

type Middleware interface {
	Handle(request *Request, response ResponseSender) error
}

type Limit struct {
	Path   string
	Method string
	Limit  uint32
}

const (
	ADDRESS_HEADERS_FP = iota
)

// Fingerprint of some type
type Fingerprint struct {
	Type  int
	Value []byte
}

// String returns a string representation of the fingerprint
func (fp Fingerprint) String() string {
	return fmt.Sprintf("%x", fp.Value)
}

// Calculates a fingerprint
type FingerprintCalculator interface {
	Calculate(r *Request) Fingerprint
}
