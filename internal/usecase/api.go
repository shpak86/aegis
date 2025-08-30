package usecase

type Meta struct {
	Fingerprint Fingerprint
}

type Request struct {
	ClientAddress string            `json:"clientAddress"`
	Method        string            `json:"method"`
	Url           string            `json:"url"`
	Body          string            `json:"body"`
	Headers       map[string]string `json:"headers"`
	Cookies       map[string]string `json:"cookies"`
	Metadata      Meta
}

type Response struct {
	Code    int               `json:"code"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
}

type Endpoint struct {
	Path   string
	Method string
}

type Protection struct {
	Path   string `json:"path"`
	Method string `json:"method"`
	Limit  uint32 `json:"rps"`
}
