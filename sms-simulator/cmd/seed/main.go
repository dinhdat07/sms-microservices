package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Raw server struct to bypass domain dependencies
type Server struct {
	ServerID      string    `gorm:"primaryKey;column:server_id"`
	ServerName    string    `gorm:"column:server_name"`
	IPv4          string    `gorm:"column:ipv4"`
	CurrentStatus string    `gorm:"column:current_status"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (Server) TableName() string {
	return "management_schema.servers"
}

// Raw Redis cache struct
type CacheItem struct {
	ID         string `json:"id"`
	IPv4       string `json:"ipv4"`
	Status     string `json:"status"`
	RetryCount int    `json:"retry_count"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
}

func run() error {
	count := 10000
	if v := os.Getenv("SIMULATOR_IP_COUNT"); v != "" {
		count, _ = strconv.Atoi(v)
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:15432/sms?sslmode=disable"
	}
	
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// 1. Setup Postgres
	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}

	log.Println("Cleaning up previous simulation servers...")
	result := db.Where("server_name LIKE ?", "sim-%").Delete(&Server{})
	if result.Error != nil {
		return fmt.Errorf("cleanup old sim servers: %w", result.Error)
	}
	log.Printf("Deleted %d old simulation servers\n", result.RowsAffected)

	// 2. Setup Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx := context.Background()
	keys, _ := rdb.Keys(ctx, "server:*").Result()
	if len(keys) > 0 {
		rdb.Del(ctx, keys...)
	}
	log.Printf("Redis: flushed %d server keys\n", len(keys))

	// 3. Seed Data
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
		redisPipeline := rdb.Pipeline()

		for j := i; j < end; j++ {
			ip := fmt.Sprintf("%s.%d.%d", subnet, octet3, octet4)
			id := uuid.New().String()
			name := fmt.Sprintf("sim-%s", id[:8])

			// Add to DB batch
			batch = append(batch, &Server{
				ServerID:      id,
				ServerName:    name,
				IPv4:          ip,
				CurrentStatus: "ONLINE",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			})

			// Add to Redis pipeline (New Schema: Hash for info, Set for all IDs)
			redisKey := fmt.Sprintf("server:info:%s", id)
			redisPipeline.HSet(ctx, redisKey, map[string]interface{}{
				"ipv4":        ip,
				"status":      "ONLINE",
				"retry_count": 0,
			})
			redisPipeline.SAdd(ctx, "server:all_ids", id)

			octet4++
			if octet4 > 254 {
				octet4 = 1
				octet3++
			}
		}

		// Execute DB Insert
		if err := db.Create(batch).Error; err != nil {
			return fmt.Errorf("batch create at offset %d: %w", i, err)
		}

		// Execute Redis Pipeline
		if _, err := redisPipeline.Exec(ctx); err != nil {
			log.Printf("WARN: redis batch upsert at offset %d: %v\n", i, err)
		}

		log.Printf("Seeded %d/%d servers\n", end, count)
	}

	log.Printf("Seed complete: %d servers in DB + Redis\n", count)
	return nil
}
