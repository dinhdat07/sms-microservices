package checkers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestICMPChecker_Check(t *testing.T) {
	checker := NewICMPChecker(false, 1*time.Millisecond)

	t.Run("missing ip", func(t *testing.T) {
		res := checker.Check(context.Background(), ServerConfig{})
		assert.False(t, res)
	})

	t.Run("invalid ip format", func(t *testing.T) {
		res := checker.Check(context.Background(), ServerConfig{"ipv4": "invalid-ip-!@#$"})
		assert.False(t, res)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		res := checker.Check(ctx, ServerConfig{"ipv4": "127.0.0.1"})
		assert.False(t, res)
	})

	t.Run("timeout or no reply", func(t *testing.T) {
		// 192.0.2.1 is TEST-NET-1, should drop packets and timeout
		res := checker.Check(context.Background(), ServerConfig{"ipv4": "192.0.2.1"})
		assert.False(t, res)
	})
}
