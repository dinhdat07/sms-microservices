package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"sms-simulator/internal/seed"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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

	// Assuming the simulator host name from within docker network is "sms-simulator"
	simulatorHost := os.Getenv("SIMULATOR_HOST")
	if simulatorHost == "" {
		simulatorHost = "sms-simulator"
	}

	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	return seed.RunSeed(db, rdb, count, simulatorHost)
}
