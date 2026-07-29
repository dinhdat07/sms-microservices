package seed

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Server struct {
	ServerID          string    `gorm:"primaryKey;column:server_id"`
	ServerName        string    `gorm:"column:server_name"`
	IPv4              string    `gorm:"column:ipv4"`
	CurrentStatus     string    `gorm:"column:current_status"`
	HealthCheckMethod string    `gorm:"column:health_check_method"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

type ReportingServer struct {
	ServerID  string    `gorm:"primaryKey;column:server_id"`
	Name      string    `gorm:"column:name"`
	IPv4      string    `gorm:"column:ipv4"`
	Status    string    `gorm:"column:status"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (ReportingServer) TableName() string {
	return "reporting_schema.reporting_servers"
}

func (Server) TableName() string {
	return "management_schema.servers"
}

func RunSeed(db *gorm.DB, rdb redis.UniversalClient, count int, simulatorHost string) error {
	log.Println("Cleaning up previous simulation servers...")

	var oldSimServerIDs []string
	db.Model(&Server{}).Where("server_name LIKE ?", "sim-%").Pluck("server_id", &oldSimServerIDs)

	result := db.Where("server_name LIKE ?", "sim-%").Delete(&Server{})
	if result.Error != nil {
		return fmt.Errorf("cleanup old sim servers: %w", result.Error)
	}
	log.Printf("Deleted %d old simulation servers from management\n", result.RowsAffected)

	resultReporting := db.Where("name LIKE ?", "sim-%").Delete(&ReportingServer{})
	if resultReporting.Error != nil {
		return fmt.Errorf("cleanup old sim servers reporting: %w", resultReporting.Error)
	}
	log.Printf("Deleted %d old simulation servers from reporting\n", resultReporting.RowsAffected)

	ctx := context.Background()

	if len(oldSimServerIDs) > 0 {
		var redisKeys []string
		for _, id := range oldSimServerIDs {
			redisKeys = append(redisKeys, fmt.Sprintf("server:info:%s", id))
			rdb.SRem(ctx, "server:all_ids", id)
		}
		rdb.Del(ctx, redisKeys...)
	}
	log.Printf("Redis: flushed %d simulation server keys\n", len(oldSimServerIDs))

	batchSize := 500
	subnet := "10.1"
	octet3 := 0
	octet4 := 1

	log.Printf("Seeding %d servers in batches of %d...\n", count, batchSize)

	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}

		var batch []*Server
		var reportingBatch []*ReportingServer
		redisPipeline := rdb.Pipeline()

		for j := i; j < end; j++ {
			ip := fmt.Sprintf("%s.%d.%d", subnet, octet3, octet4)
			id := uuid.New().String()
			name := fmt.Sprintf("sim-%s", id[:8])

			// 70% ICMP, 10% SSH, 10% AGENT_PULL, 10% AGENT_PUSH
			method := "ICMP"
			rnd := rand.Intn(100)
			if rnd < 10 {
				method = "SSH"
			} else if rnd < 20 {
				method = "AGENT_PULL"
			} else if rnd < 30 {
				method = "AGENT_PUSH"
			}

			batch = append(batch, &Server{
				ServerID:          id,
				ServerName:        name,
				IPv4:              ip,
				CurrentStatus:     "UNKNOWN",
				HealthCheckMethod: method,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			})

			reportingBatch = append(reportingBatch, &ReportingServer{
				ServerID:  id,
				Name:      name,
				IPv4:      ip,
				Status:    "UNKNOWN",
				UpdatedAt: time.Now(),
			})

			infoMap := map[string]interface{}{
				"id":                  id,
				"ipv4":                ip,
				"status":              "UNKNOWN",
				"retry_count":         0,
				"health_check_method": method,
			}

			switch method {
			case "SSH":
				infoMap["ssh_port"] = "2222"
				infoMap["ssh_user"] = "sim"
				infoMap["ssh_key"] = ""
			case "AGENT_PULL":
				infoMap["agent_endpoint"] = fmt.Sprintf("http://%s:8080/health", simulatorHost)
			case "AGENT_PUSH":
				redisPipeline.ZAdd(ctx, "monitoring:agent:heartbeats", redis.Z{
					Score:  float64(time.Now().Unix()),
					Member: id,
				})
			}

			redisKey := fmt.Sprintf("server:info:%s", id)
			redisPipeline.HSet(ctx, redisKey, infoMap)
			redisPipeline.SAdd(ctx, "server:all_ids", id)

			octet4++
			if octet4 > 254 {
				octet4 = 1
				octet3++
			}
		}

		if err := db.Create(batch).Error; err != nil {
			return fmt.Errorf("batch create at offset %d: %w", i, err)
		}

		if err := db.Create(reportingBatch).Error; err != nil {
			return fmt.Errorf("reporting batch create at offset %d: %w", i, err)
		}

		if _, err := redisPipeline.Exec(ctx); err != nil {
			log.Printf("WARN: redis batch upsert at offset %d: %v\n", i, err)
		}

		log.Printf("Seeded %d/%d servers\n", end, count)
	}

	log.Printf("Seed complete: %d servers in DB + Redis\n", count)
	return nil
}
