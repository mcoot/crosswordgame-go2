package cli

import (
	"os"
	"path/filepath"
)

// Config holds CLI configuration
type Config struct {
	ServerURL string
	Token     string
	TokenFile string
	Output    string
	Verbose   bool
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		ServerURL: getEnvOrDefault("CWGAME_SERVER", "http://localhost:8080"),
		Token:     os.Getenv("CWGAME_TOKEN"),
		TokenFile: getEnvOrDefault("CWGAME_TOKEN_FILE", defaultTokenFile()),
		Output:    "text",
		Verbose:   false,
	}
}

// LoadToken loads the token from file if not already set
func (c *Config) LoadToken() error {
	if c.Token != "" {
		return nil
	}

	data, err := os.ReadFile(c.TokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No token file is fine
		}
		return err
	}

	c.Token = string(data)
	return nil
}

// SaveToken saves the token to the token file
func (c *Config) SaveToken(token string) error {
	c.Token = token

	dir := filepath.Dir(c.TokenFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(c.TokenFile, []byte(token), 0600)
}

func defaultTokenFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cwgame/token"
	}
	return filepath.Join(home, ".cwgame", "token")
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
