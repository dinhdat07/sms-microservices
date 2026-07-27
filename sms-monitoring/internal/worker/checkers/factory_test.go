package checkers

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestCheckerFactory_GetChecker(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	timeouts := CheckerTimeouts{
		ICMP:      1 * time.Second,
		SSH:       2 * time.Second,
		AgentPull: 3 * time.Second,
	}

	factory := NewHealthCheckerFactory(rdb, false, timeouts)

	t.Run("Get SSH checker", func(t *testing.T) {
		checker := factory.GetChecker("SSH")
		assert.IsType(t, &SSHChecker{}, checker)
	})

	t.Run("Get AGENT_PULL checker", func(t *testing.T) {
		checker := factory.GetChecker("AGENT_PULL")
		assert.IsType(t, &AgentPullChecker{}, checker)
	})

	t.Run("Get ICMP checker", func(t *testing.T) {
		checker := factory.GetChecker("ICMP")
		assert.IsType(t, &ICMPChecker{}, checker)
	})

	t.Run("Get default checker (ICMP)", func(t *testing.T) {
		checker := factory.GetChecker("UNKNOWN")
		assert.IsType(t, &ICMPChecker{}, checker)
	})
}
