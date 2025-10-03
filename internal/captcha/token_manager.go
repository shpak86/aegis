package captcha

import (
	"aegis/internal/usecase"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	tokenCookie = "AEGIS_TOKEN"
	indexPath   = "/usr/share/aegis/captcha/static/index.html"
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
	complexity      int
	tokens          map[string]*token
	permanentTokens map[string]struct{}
	challenges      map[string]*Challenge
	cmu             sync.RWMutex
	tmu             sync.RWMutex
	parts           [][]byte

	CaptchaManager *CaptchaManager
}

// ExtractToken extracts the AEGIS_TOKEN cookie from the request
// Returns:
//   - string: Token value if present
//   - bool:   True if token was found
func (m *CaptchaTokenManager) ExtractToken(request *usecase.Request) (token string, exists bool) {
	token, exists = request.Cookies[tokenCookie]
	return
}

// GetChallenge generates a new CAPTCHA challenge for the client
// Parameters:
//   - fp: Client fingerprint for tracking
//
// Returns:
//   - []byte: Rendered HTML content with base64-encoded images
//   - error:  Non-nil if image reading fails
func (m *CaptchaTokenManager) GetChallenge(fp *usecase.Fingerprint) (payload []byte, err error) {
	task := m.CaptchaManager.GetChallenge()
	m.cmu.Lock()
	m.challenges[fp.String] = task
	m.cmu.Unlock()

	id := []byte(fmt.Sprintf("%d", task.Id))
	size := 0
	for i := range m.parts {
		size += len(m.parts[i])
	}
	size = size + len([]byte(task.Description))
	size = size + len(id)
	for i := range task.Base64Images {
		size += len(task.Base64Images[i])
	}
	content := make([]byte, 0, size)
	content = append(content, m.parts[0]...)
	content = append(content, []byte(task.Description)...)
	content = append(content, m.parts[1]...)
	for i := 2; i < 2+m.complexity; i++ {
		content = append(content, []byte(task.Base64Images[i-2])...)
		content = append(content, m.parts[i]...)
	}
	content = append(content, id...)
	content = append(content, m.parts[len(m.parts)-1]...)
	slog.Info("Captcha challenge is prepared", "fingerprint", fp.String, "complexity", m.complexity, "id", task.Id, "images", len(task.Base64Images), "solution", task.Solution, "description", task.Description)
	return content, nil
}

// GetToken validates a CAPTCHA solution and generates a new antibot token
// Parameters:
//   - fp:       Client fingerprint
//   - _, body:  Request payload containing the solution
//
// Returns:
//   - string:   Generated antibot token
//   - error:    Non-nil if solution is invalid or challenge not found
func (m *CaptchaTokenManager) GetToken(fp *usecase.Fingerprint, body []byte) (t string, err error) {
	var solution Solution
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
	slog.Info("Token is issued", "fingerprint", fp.String, "token", t, "id", solution.Id)
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
	if _, exists := m.permanentTokens[token]; exists {
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

// NewCaptchaTokenManager creates a new CAPTCHA token manager instance
// Parameters:
//   - complexity: Difficulty level for CAPTCHAs (1-3)
//   - config:     Path to configuration file
//
// Returns:
//   - *CaptchaTokenManager: Initialized manager with preloaded template
func NewCaptchaTokenManager(ctx context.Context, permanentTokens []string, complexity string) *CaptchaTokenManager {
	var complexityLevel int
	switch complexity {
	case "easy":
		complexityLevel = CaptchaComplexityEasy
	case "medium":
		complexityLevel = CaptchaComplexityMedium
	case "hard":
		complexityLevel = CaptchaComplexityHard
	default:
		complexityLevel = CaptchaComplexityMedium
	}
	tm := CaptchaTokenManager{
		CaptchaManager:  NewClassificationCaptchaManager(ctx, complexityLevel),
		tokens:          make(map[string]*token),
		permanentTokens: make(map[string]struct{}),
		challenges:      make(map[string]*Challenge),
		complexity:      complexityLevel,
		parts:           [][]byte{},
	}

	var index = fmt.Sprintf("/usr/share/aegis/captcha/static/index_%s.html", complexity)
	content, err := os.ReadFile(index)
	if err != nil {
		slog.Error("Unable to read template: " + indexPath)
		os.Exit(1)
	}

	buffer := strings.Split(string(content), "{{description}}")
	tm.parts = append(tm.parts, []byte(buffer[0]))
	buffer = strings.Split(buffer[1], "{{image}}")
	for i := 0; i < len(buffer)-1; i++ {
		tm.parts = append(tm.parts, []byte(buffer[i]))
	}
	buffer = strings.Split(buffer[len(buffer)-1], "{{id}}")
	tm.parts = append(tm.parts, []byte(buffer[0]), []byte(buffer[1]))

	for i := range permanentTokens {
		tm.permanentTokens[permanentTokens[i]] = struct{}{}
	}
	return &tm
}
