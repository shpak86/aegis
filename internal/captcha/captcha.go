package captcha

type Captcha interface {
	GetTask() []byte
	Validate(solution []byte)
}

const (
	CaptchaComplexityEasy   = 4
	CaptchaComplexityMedium = 6
	CaptchaComplexityHard   = 8
)
