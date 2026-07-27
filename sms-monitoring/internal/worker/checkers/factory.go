package checkers

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type ServerConfig map[string]string

type HealthChecker interface {
	Check(ctx context.Context, config ServerConfig) bool
}

type HealthCheckerFactory interface {
	GetChecker(method string) HealthChecker
}

type checkerFactory struct {
	icmp      HealthChecker
	ssh       HealthChecker
	agentPull HealthChecker
	agentPush HealthChecker
}

type CheckerTimeouts struct {
	ICMP      time.Duration
	SSH       time.Duration
	AgentPull time.Duration
}

func NewHealthCheckerFactory(rdb redis.UniversalClient, privileged bool, timeouts CheckerTimeouts) HealthCheckerFactory {
	return &checkerFactory{
		icmp:      NewICMPChecker(privileged, timeouts.ICMP),
		ssh:       NewSSHChecker(timeouts.SSH),
		agentPull: NewAgentPullChecker(timeouts.AgentPull),
		agentPush: NewAgentPushChecker(rdb),
	}
}

func (f *checkerFactory) GetChecker(method string) HealthChecker {
	switch method {
	case "SSH":
		return f.ssh
	case "AGENT_PULL":
		return f.agentPull
	case "AGENT_PUSH":
		return f.agentPush
	case "ICMP":
		fallthrough
	default:
		return f.icmp
	}
}
