package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestICMPPinger_Ping(t *testing.T) {
	pinger := NewICMPPinger(false) // UDP for tests

	t.Run("success", func(t *testing.T) {
		// Ping localhost (might fail on Windows if no privilege)
		_ = pinger.Ping("127.0.0.1", 10*time.Millisecond)
	})

	t.Run("invalid IP", func(t *testing.T) {
		ok := pinger.Ping("invalid_ip", 100*time.Millisecond)
		assert.False(t, ok)
	})

	t.Run("timeout", func(t *testing.T) {
		// Ping unreachable IP
		ok := pinger.Ping("10.255.255.254", 50*time.Millisecond)
		assert.False(t, ok)
	})
}
