package worker

import (
	"sms-monitoring/internal/infrastructure/logger"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Pinger defines the interface for executing health checks.
type Pinger interface {
	Ping(ip string, timeout time.Duration) bool
}

type ICMPPinger struct {
	privileged bool
}

func NewICMPPinger(privileged bool) Pinger {
	return &ICMPPinger{
		privileged: privileged,
	}
}

func (p *ICMPPinger) Ping(ip string, timeout time.Duration) bool {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		logger.Log.Sugar().Errorf("[Pinger] Invalid IP address or resolve failed for IP %s: %v", ip, err)
		return false
	}

	pinger.SetPrivileged(p.privileged)
	pinger.Count = 1
	pinger.Timeout = timeout

	err = pinger.Run()
	if err != nil {
		logger.Log.Sugar().Errorf("[Pinger] Execution failed for IP %s: %v", ip, err)
		return false
	}

	stats := pinger.Statistics()
	// Return true if received at least one packet
	return stats.PacketsRecv > 0
}
