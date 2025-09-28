package usecase

const (
	VerdictContinue = iota
	VerdictBan
	VerdictAllow
)

type TokenManager interface {
	ExtractToken(*Request) (string, bool)
	GetChallenge(fp *Fingerprint) ([]byte, error)
	GetToken(fp *Fingerprint, solution []byte) (string, error)
	Validate(*Fingerprint, string) bool
	Revoke(string) bool
}

type ResponseSender interface {
	Send(*Response) error
}

// Fingerprint of some type
type Fingerprint struct {
	Type   int
	Value  []byte
	String string
}

type RequestContext[T any] struct {
	Fingerprint Fingerprint
	Factors     T
}

type FingerprintCalculator[T any] interface {
	Calculate(*T) Fingerprint
}

type Chain[T any] interface {
	Execute(RequestContext[T])
}

type HttpFactors struct {
	Cookies       map[string]string
	Headers       map[string]string
	Method        string
	Path          string
	ClientAddress string
	Token         string
	Body          []byte
}
