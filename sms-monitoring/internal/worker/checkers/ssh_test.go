package checkers

import (
	"context"
	"os"
	"testing"
	"time"

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
}
