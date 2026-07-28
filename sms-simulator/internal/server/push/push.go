package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"sms-simulator/internal/nft"

	"github.com/redis/go-redis/v9"
)

func StartAgentPushWorker(rdb redis.UniversalClient, endpoint string) {
	go func() {
		ctx := context.Background()
		client := &http.Client{Timeout: 2 * time.Second}

		// Find all AGENT_PUSH servers once to avoid heavy Redis queries
		var pushServerIDs []string

		log.Println("Discovering AGENT_PUSH servers in Redis...")
		serverIDs, err := rdb.SMembers(ctx, "server:all_ids").Result()
		if err == nil {
			for _, id := range serverIDs {
				method, _ := rdb.HGet(ctx, fmt.Sprintf("server:info:%s", id), "health_check_method").Result()
				if method == "AGENT_PUSH" {
					pushServerIDs = append(pushServerIDs, id)
				}
			}
		}

		log.Printf("Found %d AGENT_PUSH servers. Starting push loop every 5s", len(pushServerIDs))

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			downSet, _ := nft.ListDownIPs()

			for _, id := range pushServerIDs {
				ip, _ := rdb.HGet(ctx, fmt.Sprintf("server:info:%s", id), "ipv4").Result()
				if downSet[ip] {
					continue // Server is offline, do not push heartbeat
				}

				payload, _ := json.Marshal(map[string]interface{}{
					"server_id": id,
					"timestamp": time.Now().Unix(),
				})

				req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
				req.Header.Set("Content-Type", "application/json")
				
				req.Header.Set("X-Master-Key", "0123456789abcdef0123456789abcdef")

				go func(r *http.Request) {
					resp, err := client.Do(r)
					if err == nil {
						resp.Body.Close()
					}
				}(req)
			}
		}
	}()
}
