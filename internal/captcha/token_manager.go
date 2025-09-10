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

type TokenGenerationError struct {
	message string
}

func (e TokenGenerationError) Error() string {
	return e.message
}

type token struct {
	bytes     []byte
	time      time.Time
	challenge *challenge
}

func (t token) String() string {
	return string(t.bytes)
}

type challenge struct {
	clientFp  *usecase.Fingerprint
	time      time.Time
	challenge []byte
}

type CaptchaTokenManager struct {
	complexity int
	tokens     map[string]*token
	challenges map[string]*CaptchaTask
	cmu        sync.RWMutex
	tmu        sync.RWMutex
	template   *template.Template

	CaptchaManager *ClassificationCaptchaManager
}

// Extracts token from request
func (m *CaptchaTokenManager) ExtractToken(request *usecase.Request) (token string, exists bool) {
	token, exists = request.Cookies[tokenCookie]
	return
}

type PageData struct {
	CaptchaId   uint32
	Description string
	Images      []string
}

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

	// t, _ := m.template.Clone()
	var content bytes.Buffer
	m.template.Execute(&content, PageData{
		CaptchaId:   task.Id,
		Description: task.Description,
		Images:      encodedImages,
	})
	return content.Bytes(), nil
}

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

// Validates token and returns true if the token is valid.
func (m *CaptchaTokenManager) Validate(clientFp *usecase.Fingerprint, token string) bool {
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

// Revoke token if it exists. Returns true is token exists ant was revoked.
func (m *CaptchaTokenManager) Revoke(token string) bool {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	_, revoked := m.tokens[token]
	delete(m.tokens, token)
	return revoked
}

// Get complexity returns the level of challange complexity. 1 - the easiest, 4 - most complex
func (m *CaptchaTokenManager) GetComplexity() int {
	return m.complexity
}

func (m *CaptchaTokenManager) Serve() {
	m.CaptchaManager.Serve()
}

func NewCaptchaTokenManager(complexity int, config string) *CaptchaTokenManager {
	tm := CaptchaTokenManager{
		CaptchaManager: NewClassificationCaptchaManager(context.Background(), complexity, config),
		tokens:         make(map[string]*token),
		challenges:     make(map[string]*CaptchaTask),
		template:       template.Must(template.New("captcha").Parse(captchaPage)),
	}
	return &tm
}
