package checkers

import (
	"context"
	"time"

	"sms-monitoring/internal/infrastructure/logger"

	probing "github.com/prometheus-community/pro-bing"
)

type ICMPChecker struct {
	privileged bool
	timeout    time.Duration
}

func NewICMPChecker(privileged bool, timeout time.Duration) HealthChecker {
	return &ICMPChecker{
		privileged: privileged,
		timeout:    timeout,
	}
}

func (c *ICMPChecker) Check(ctx context.Context, config ServerConfig) bool {
	ip := config["ipv4"]
	if ip == "" {
		return false
	}

	pinger, err := probing.NewPinger(ip)
	if err != nil {
		logger.Log.Sugar().Errorf("[ICMPChecker] Invalid IP address or resolve failed for IP %s: %v", ip, err)
		return false
	}

	pinger.SetPrivileged(c.privileged)
	pinger.Count = 1
	pinger.Timeout = c.timeout

	errCh := make(chan error, 1)
	go func() {
		errCh <- pinger.Run()
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		return false
	case err := <-errCh:
		if err != nil {
			logger.Log.Sugar().Errorf("[ICMPChecker] Execution failed for IP %s: %v", ip, err)
			return false
		}
	}

	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
}
