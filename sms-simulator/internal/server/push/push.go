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

func StartAgentPushWorker(rdb redis.UniversalClient, endpoint string, masterKey string) {
	go func() {
		ctx := context.Background()
		client := &http.Client{Timeout: 2 * time.Second}

		var pushServerIDs []string
		
		refreshServers := func() {
			serverIDs, err := rdb.SMembers(ctx, "server:all_ids").Result()
			if err != nil {
				return
			}
			var newIDs []string
			for _, id := range serverIDs {
				method, _ := rdb.HGet(ctx, fmt.Sprintf("server:info:%s", id), "health_check_method").Result()
				if method == "AGENT_PUSH" {
					newIDs = append(newIDs, id)
				}
			}
			pushServerIDs = newIDs
			log.Printf("Refreshed AGENT_PUSH servers. Found %d servers.", len(pushServerIDs))
		}

		log.Println("Discovering AGENT_PUSH servers in Redis...")
		refreshServers()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		tickCount := 0

		for range ticker.C {
			tickCount++
			if tickCount%6 == 0 { // Every 30 seconds (6 * 5s)
				refreshServers()
			}
			
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
				
				req.Header.Set("X-Master-Key", masterKey)

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
