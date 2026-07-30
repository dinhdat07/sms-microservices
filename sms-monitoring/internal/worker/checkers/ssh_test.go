package checkers

import (
	"context"
	"os"
	"testing"
	"time"

	"sms-monitoring/internal/infrastructure/security"

	"github.com/stretchr/testify/assert"

)

func TestSSHChecker_Check(t *testing.T) {
	// Set encryption key for decrypt to work
	os.Setenv("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	checker := NewSSHChecker(1 * time.Second)

	t.Run("missing ip", func(t *testing.T) {
		res := checker.Check(context.Background(), ServerConfig{
			"ssh_port": "22",
			"ssh_user": "root",
		})
		assert.False(t, res)
	})

	t.Run("missing port", func(t *testing.T) {
		res := checker.Check(context.Background(), ServerConfig{
			"ipv4":     "1.1.1.1",
			"ssh_user": "root",
		})
		assert.False(t, res)
	})

	t.Run("missing user", func(t *testing.T) {
		res := checker.Check(context.Background(), ServerConfig{
			"ipv4":     "1.1.1.1",
			"ssh_port": "22",
		})
		assert.False(t, res)
	})

	t.Run("decrypt fail", func(t *testing.T) {
		res := checker.Check(context.Background(), ServerConfig{
			"ipv4":     "1.1.1.1",
			"ssh_port": "22",
			"ssh_user": "root",
			"ssh_key":  "invalid-base64",
		})
		assert.False(t, res)
	})

	t.Run("dial fail", func(t *testing.T) {
		res := checker.Check(context.Background(), ServerConfig{
			"ipv4":     "127.0.0.1",
			"ssh_port": "0", // Invalid port to force dial error
			"ssh_user": "root",
		})
		assert.False(t, res)
	})

	t.Run("decrypt success but dial fail", func(t *testing.T) {
		// Valid base64, but maybe invalid ciphertext structure - actually let's use the security package to encrypt it first!
		encKey, _ := security.Encrypt("my-secret-key")
		res := checker.Check(context.Background(), ServerConfig{
			"ipv4":     "127.0.0.1",
			"ssh_port": "0", // Invalid port to force dial error
			"ssh_user": "root",
			"ssh_key":  encKey,
		})
		assert.False(t, res)
	})
}
