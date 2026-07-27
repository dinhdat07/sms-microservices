package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	simhttp "sms-simulator/internal/server/http"
	simpush "sms-simulator/internal/server/push"
	simssh "sms-simulator/internal/server/ssh"

	"github.com/redis/go-redis/v9"
)

func main() {
	totalIPs := 10000
	if v := os.Getenv("SIMULATOR_IP_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			totalIPs = n
		}
	}
	subnet := envStr("SIMULATOR_SUBNET", "10.1")

	// 1. Setup SSH Server
	sshPort := 2222
	if v := os.Getenv("SIMULATOR_SSH_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			sshPort = p
		}
	}
	err := simssh.StartDummySSHServer(sshPort)
	if err != nil {
		log.Fatalf("Failed to start SSH server: %v", err)
	}

	// 2. Setup AGENT_PUSH routine
	redisAddr := envStr("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	agentEndpoint := envStr("AGENT_HANDLER_URL", "http://sms-agent-handler:8080/api/v1/agent/heartbeat")
	simpush.StartAgentPushWorker(rdb, agentEndpoint)

	// 3. Setup API Server & Auto Flapper
	srv := simhttp.NewServer(totalIPs, subnet)
	if envBool("SIMULATOR_AUTO_FLAP_ENABLED", false) {
		srv.StartAutoFlapper()
	}

	mux := srv.SetupMux()

	log.Printf("Simulator API listening on :8080 (%d IPs, subnet=%s)", totalIPs, subnet)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return defaultVal
}
