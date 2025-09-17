package captcha

import (
	"aegis/internal/usecase"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"os"
	"sync"
	"time"
)

const (
	tokenCookie = "AEGIS_TOKEN"
)

// TokenGenerationError represents errors during token generation operations
type TokenGenerationError struct {
	message string
}

func (e TokenGenerationError) Error() string {
	return e.message
}

// token represents a generated antibot token with metadata
type token struct {
	bytes     []byte
	time      time.Time
	challenge *challenge
}

// String returns the base64-encoded string representation of the token
func (t token) String() string {
	return string(t.bytes)
}

// challenge represents a CAPTCHA challenge associated with a client fingerprint
type challenge struct {
	clientFp  *usecase.Fingerprint
	time      time.Time
	challenge []byte
}

// CaptchaTokenManager manages CAPTCHA challenges and antibot tokens
type CaptchaTokenManager struct {
	complexity     int
	tokens         map[string]*token
	permnentTokens map[string]bool
	challenges     map[string]*CaptchaTask
	cmu            sync.RWMutex
	tmu            sync.RWMutex
	template       *template.Template

	CaptchaManager *ClassificationCaptchaManager
}

// ExtractToken extracts the AEGIS_TOKEN cookie from the request
// Returns:
//   - string: Token value if present
//   - bool:   True if token was found
func (m *CaptchaTokenManager) ExtractToken(request *usecase.Request) (token string, exists bool) {
	token, exists = request.Cookies[tokenCookie]
	return
}

// PageData contains template data for rendering CAPTCHA pages
type PageData struct {
	CaptchaId   uint32
	Description string
	Images      []string
}

// GetChallenge generates a new CAPTCHA challenge for the client
// Parameters:
//   - fp: Client fingerprint for tracking
//
// Returns:
//   - []byte: Rendered HTML content with base64-encoded images
//   - error:  Non-nil if image reading fails
func (m *CaptchaTokenManager) GetChallenge(fp *usecase.Fingerprint) (payload []byte, err error) {
	task := m.CaptchaManager.GetTask()
	m.cmu.Lock()
	m.challenges[fp.String] = task
	m.cmu.Unlock()

	encodedImages := make([]string, len(task.Images))
	for i, imagePath := range task.Images {
		var image []byte
		if image, err = os.ReadFile(imagePath); err != nil {
			return
		}
		encodedImages[i] = base64.StdEncoding.EncodeToString(image)
	}

	var content bytes.Buffer
	m.template.Execute(&content, PageData{
		CaptchaId:   task.Id,
		Description: task.Description,
		Images:      encodedImages,
	})
	return content.Bytes(), nil
}

// GetToken validates a CAPTCHA solution and generates a new antibot token
// Parameters:
//   - fp:       Client fingerprint
//   - _, body:  Request payload containing the solution
//
// Returns:
//   - string:   Generated antibot token
//   - error:    Non-nil if solution is invalid or challenge not found
func (m *CaptchaTokenManager) GetToken(fp *usecase.Fingerprint, _, body []byte) (t string, err error) {
	var solution CaptchaSolution
	err = json.Unmarshal([]byte(body), &solution)
	if err != nil {
		return
	}

	if task, exists := m.challenges[fp.String]; !exists || task.Id != solution.Id {
		err = TokenGenerationError{message: "wrong client"}
		return
	}

	if !m.CaptchaManager.Verify(&solution) {
		err = TokenGenerationError{message: "wrong solution"}
		return
	}
	m.cmu.Lock()
	defer m.cmu.Unlock()
	r := make([]byte, 32)
	rand.Read(r)
	t = base64.StdEncoding.EncodeToString(r)
	m.tmu.Lock()
	defer m.tmu.Unlock()
	m.tokens[t] = &token{bytes: []byte(t), time: time.Now(), challenge: &challenge{clientFp: fp, time: time.Now(), challenge: []byte{}}}
	delete(m.challenges, fp.String)
	return
}

// Validate checks if a token is permanent or valid for the given client fingerprint
// Parameters:
//   - clientFp: Fingerprint to validate against
//   - token:    Token string to verify
//
// Returns:
//   - bool: True if token is permanent or valid and matches the fingerprint
func (m *CaptchaTokenManager) Validate(clientFp *usecase.Fingerprint, token string) bool {
	if _, exists := m.permnentTokens[token]; exists {
		return true
	}
	m.tmu.RLock()
	defer m.tmu.RUnlock()
	storedToken, exists := m.tokens[token]
	if !exists {
		return false
	}
	if !bytes.Equal(storedToken.challenge.clientFp.Value, clientFp.Value) {
		return false
	}
	return true
}

// Revoke removes a token from storage if it exists
// Parameters:
//   - token: Token string to revoke
//
// Returns:
//   - bool: True if token existed and was successfully removed
func (m *CaptchaTokenManager) Revoke(token string) bool {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	_, revoked := m.tokens[token]
	delete(m.tokens, token)
	return revoked
}

// GetComplexity returns the configured CAPTCHA difficulty level
// 1 - easiest, 3 - most complex
func (m *CaptchaTokenManager) GetComplexity() int {
	return m.complexity
}

// Serve runs background tasks for the CAPTCHA manager
func (m *CaptchaTokenManager) Serve() {
	m.CaptchaManager.Serve()
}

// NewCaptchaTokenManager creates a new CAPTCHA token manager instance
// Parameters:
//   - complexity: Difficulty level for CAPTCHAs (1-3)
//   - config:     Path to configuration file
//
// Returns:
//   - *CaptchaTokenManager: Initialized manager with preloaded template
func NewCaptchaTokenManager(permanentTokens []string, complexity int, templatesConfigPath string) *CaptchaTokenManager {
	tm := CaptchaTokenManager{
		CaptchaManager: NewClassificationCaptchaManager(context.Background(), complexity, templatesConfigPath),
		tokens:         make(map[string]*token),
		permnentTokens: make(map[string]bool),
		challenges:     make(map[string]*CaptchaTask),
		template:       template.Must(template.New("captcha").Parse(captchaPage)),
	}
	for i := range permanentTokens {
		tm.permnentTokens[permanentTokens[i]] = true
	}
	return &tm
}
