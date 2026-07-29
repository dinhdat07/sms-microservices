package checkers

import (
	"context"
	"net/http"
	"time"

	"sms-monitoring/internal/infrastructure/logger"
)

type AgentPullChecker struct {
	httpClient *http.Client
}

func NewAgentPullChecker(timeout time.Duration) HealthChecker {
	return &AgentPullChecker{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *AgentPullChecker) Check(ctx context.Context, config ServerConfig) bool {
	endpoint := config["agent_endpoint"]
	if endpoint == "" {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Log.Sugar().Errorf("[AgentPullChecker] Request failed to %s: %v", endpoint, err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
