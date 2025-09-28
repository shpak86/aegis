package config

import (
	"encoding/json"
	"math"
	"os"
	"strings"
)

// ProtectionConfig defines rate-limiting rules for specific HTTP endpoints.
type ProtectionConfig struct {
	Path   string `json:"path"`   // URL path to protect (e.g., "/api/v1/login")
	Method string `json:"method"` // HTTP method to protect (e.g., "POST")
	Limit  uint32 `json:"rps"`    // Maximum requests per second allowed
}

// VerificationConfig specifies client verification requirements.
type VerificationConfig struct {
	Type       string `json:"type"`       // Verification method type (default: "js-challenge")
	Complexity string `json:"complexity"` // Computational difficulty for verification
}

// Config contains global application configuration loaded from JSON.
type Config struct {
	Address string `json:"address"` // Server listen address (e.g., ":8080")

	Logger struct {
		Level string `json:"level"` // Logging verbosity level (e.g., "info", "debug")
	} `json:"logger"`

	Protections  []ProtectionConfig `json:"protections"`  // List of endpoint protection rules
	Verification VerificationConfig `json:"verification"` // Client verification settings

	PermanentTokens []string `json:"permanent_tokens"` // List of permanent tokens
}

// Load reads and parses a JSON configuration file into the receiver.
//
// Parameters:
//   - file: Path to the JSON configuration file.
//
// Returns:
//   - error: Non-nil if file reading/parsing fails.
//
// Processing steps:
// 1. Reads the entire file content.
// 2. Unmarshals JSON into the Config structure.
// 3. Sets default values:
//   - Sets Verification.Type to "js-challenge" if empty.
//
// 4. Normalizes protection rules:
//   - Sets Limit=MaxUint32 if zero (unlimited).
//   - Converts Method to uppercase (case-insensitive HTTP methods).
func (c *Config) Load(file string) (err error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, c)
	if err != nil {
		return
	}

	if c.Logger.Level == "" {
		c.Logger.Level = "INFO"
	} else {
		c.Logger.Level = strings.ToUpper(c.Logger.Level)
	}

	if c.Address == "" {
		c.Address = "localhost:2048"
	}

	if c.Verification.Type == "" {
		c.Verification.Type = "js-challenge"
	}

	for i := range c.Protections {
		if c.Protections[i].Limit == 0 {
			c.Protections[i].Limit = math.MaxUint32
		}
		c.Protections[i].Method = strings.ToUpper(c.Protections[i].Method)
	}
	return
}
