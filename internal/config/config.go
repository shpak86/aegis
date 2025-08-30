package config

type ProtectionConfig struct {
	Path   string `json:"path"`
	Method string `json:"method"`
	Limit  uint32 `json:"rps"`
}

type TokenConfig struct {
	Complexity int `json:"complexity"`
}

type Config struct {
	Address string `json:"address"`
	Logger  struct {
		Level string `json:"level"`
	} `json:"logger"`
	Protections []ProtectionConfig `json:"protections"`
	Token       TokenConfig        `json:"token"`
}

// todo config validation
