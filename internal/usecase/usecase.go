package usecase

const (
	VerdictContinue = iota
	VerdictBan
	VerdictAllow
)

type TokenManager interface {
	ExtractToken(*Request) (string, bool)
	GetChallenge(fp *Fingerprint) ([]byte, error)
	GetToken(fp *Fingerprint, challenge, solution []byte) (string, error)
	Validate(*Fingerprint, string) bool
	Revoke(string) bool
}

type ResponseSender interface {
	Send(*Response) error
}

type Middleware interface {
	Handle(request *Request, response ResponseSender) error
}

const (
	ADDRESS_HEADERS_FP = iota
)

// Fingerprint of some type
type Fingerprint struct {
	Type   int
	Value  []byte
	String string
}

// Calculates a fingerprint
type FingerprintCalculator interface {
	Calculate(r *Request) Fingerprint
}
