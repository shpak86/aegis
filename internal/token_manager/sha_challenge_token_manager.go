package token_manager

import (
	"aegis/internal/usecase"
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"sync"
	"time"
)

const (
	tokenCookie = "AB135"
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
	complexity int
	tokens     map[string]*token
	challenges map[string]*challenge
	cmu        sync.RWMutex
	tmu        sync.RWMutex
}

func (m *ShaChallengeTokenManager) GetRequestToken(request *usecase.Request) (token string, exists bool) {
	token, exists = request.Cookies[tokenCookie]
	return
}

func (m *ShaChallengeTokenManager) GetChallenge(fp *usecase.Fingerprint) []byte {
	c := challenge{clientFp: fp, time: time.Now(), challenge: make([]byte, m.complexity)}
	rand.Read(c.challenge)
	m.cmu.Lock()
	defer m.cmu.Unlock()
	m.challenges[string(c.challenge)] = &c
	return c.challenge
}

// Checks the solution for the specified fingerprint. If the soluiton is correct the new token will be returned.
// If solution is incorrect or some internal error occured, false will be returned.
func (m *ShaChallengeTokenManager) GetToken(fp *usecase.Fingerprint, challenge, solution []byte) (t string, err error) {
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
	delete(m.challenges, challengeString)
	return
}

// Validates token and returns true if the token is valid.
func (m *ShaChallengeTokenManager) Validate(clientFp *usecase.Fingerprint, token string) bool {
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
func (m *ShaChallengeTokenManager) Revoke(token string) bool {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	_, revoked := m.tokens[token]
	delete(m.tokens, token)
	return revoked
}

// Get complexity returns the level of challange complexity. 1 - the easiest, 4 - most complex
func (m *ShaChallengeTokenManager) GetComplexity() int {
	return m.complexity
}

func NewShaChallengeTokenManager(complexity int) *ShaChallengeTokenManager {
	tm := ShaChallengeTokenManager{
		complexity: complexity,
		tokens:     make(map[string]*token),
		challenges: make(map[string]*challenge),
	}
	return &tm
}
