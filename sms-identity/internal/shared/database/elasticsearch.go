package database

import (
	"context"
	"fmt"
	"sms-identity/internal/shared/logger"

	esv8 "github.com/elastic/go-elasticsearch/v8"
)

func NewElasticsearchClient(ctx context.Context, addresses []string) (*esv8.TypedClient, error) {
	cfg := esv8.Config{
		Addresses: addresses,
	}

	es, err := esv8.NewTypedClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client: %w", err)
	}

	_, err = es.Info().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect elasticsearch: %w", err)
	}

	logger.Log.Sugar().Info("[Elasticsearch] Connected successfully")

	return es, nil
}