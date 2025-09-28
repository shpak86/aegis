package sha_challenge

import (
	"aegis/internal/usecase"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"html/template"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	tokenCookie = "AEGIS_TOKEN"
	indexPath   = "/usr/share/aegis/sha-challenge/static/index.html"
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

type ShaChallengeTokenManager struct {
	complexity      int
	tokens          map[string]*token
	permanentTokens map[string]struct{}
	challenges      map[string]*challenge
	cmu             sync.RWMutex
	tmu             sync.RWMutex
	template        *template.Template
}

type pageData struct {
	Challenge string
}

// Extracts token from request
func (m *ShaChallengeTokenManager) ExtractToken(request *usecase.Request) (token string, exists bool) {
	token, exists = request.Cookies[tokenCookie]
	return
}

func (m *ShaChallengeTokenManager) GetChallenge(fp *usecase.Fingerprint) ([]byte, error) {
	c := challenge{clientFp: fp, time: time.Now(), challenge: make([]byte, m.complexity)}
	rand.Read(c.challenge)
	m.cmu.Lock()
	defer m.cmu.Unlock()
	m.challenges[string(c.challenge)] = &c
	var content bytes.Buffer
	challengeString := base64.StdEncoding.EncodeToString(c.challenge)
	err := m.template.Execute(&content, pageData{
		Challenge: challengeString,
	})
	slog.Info("SHA challenge is prepared", "fingerprint", fp.String, "complexity", m.complexity, "challenge", challengeString)
	return content.Bytes(), err
}

// Checks the solution for the specified fingerprint. If the soluiton is correct the new token will be returned.
// If solution is incorrect or some internal error occured, false will be returned.
func (m *ShaChallengeTokenManager) GetToken(fp *usecase.Fingerprint, payload []byte) (t string, err error) {
	if len(payload) < 10 {
		err = TokenGenerationError{message: "wrong solution"}
		return
	}
	message, err := base64.StdEncoding.DecodeString(string(payload))
	challenge, solution := message[:m.complexity], message[m.complexity:]
	m.cmu.Lock()
	defer m.cmu.Unlock()
	challengeString := string(challenge)
	c, exists := m.challenges[challengeString]
	if !exists {
		err = TokenGenerationError{message: "wrong challenge"}
		return
	}
	if !bytes.Equal(fp.Value, c.clientFp.Value) {
		err = TokenGenerationError{message: "wrong client"}
		return
	}
	solutionHash := sha512.Sum512(solution)
	for i, b := range challenge {
		if solutionHash[i] != b {
			err = TokenGenerationError{message: "wrong solution"}
			return
		}
	}
	r := make([]byte, 32)
	rand.Read(r)
	t = base64.StdEncoding.EncodeToString(r)
	m.tmu.Lock()
	defer m.tmu.Unlock()
	m.tokens[t] = &token{bytes: []byte(t), time: time.Now(), challenge: c}
	slog.Info("Token is issued", "fingerprint", fp.String, "token", t, "challenge", challenge, "solution", solution)
	delete(m.challenges, challengeString)
	return
}

// Validates token and returns true if the token is valid.
func (m *ShaChallengeTokenManager) Validate(clientFp *usecase.Fingerprint, token string) bool {
	m.tmu.RLock()
	defer m.tmu.RUnlock()
	if _, exists := m.permanentTokens[token]; exists {
		return true
	}
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
func (m *ShaChallengeTokenManager) Revoke(token string) bool {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	_, revoked := m.tokens[token]
	delete(m.tokens, token)
	slog.Debug("Revoked token", "token", token)
	return revoked
}

// Get complexity returns the level of challange complexity. 1 - the easiest, 4 - most complex
func (m *ShaChallengeTokenManager) GetComplexity() int {
	return m.complexity
}

func (m *ShaChallengeTokenManager) Serve(ctx context.Context) {
	m.cmu.Lock()
	defer m.cmu.Unlock()
	ticker := time.NewTicker(time.Second)
	select {
	case <-ticker.C:
		ticker.Stop()
		return
	case <-ctx.Done():
		for k, v := range m.challenges {
			if time.Since(v.time) > time.Minute {
				delete(m.challenges, k)
			}
		}
	}
}

func NewShaChallengeTokenManager(permanentTokens []string, complexity string) *ShaChallengeTokenManager {
	var complexityLevel int
	switch complexity {
	case "easy":
		complexityLevel = 1
	case "medium":
		complexityLevel = 2
	case "hard":
		complexityLevel = 3
	default:
		complexityLevel = 2
	}
	pageContent, err := os.ReadFile(indexPath)
	if err != nil {
		slog.Error("Unable to read template: " + indexPath)
		os.Exit(1)
	}
	tm := ShaChallengeTokenManager{
		complexity:      complexityLevel,
		tokens:          make(map[string]*token),
		challenges:      make(map[string]*challenge),
		template:        template.Must(template.New("sha-challenge").Parse(string(pageContent))),
		permanentTokens: make(map[string]struct{}),
	}
	for i := range permanentTokens {
		tm.permanentTokens[permanentTokens[i]] = struct{}{}
	}
	return &tm
}
