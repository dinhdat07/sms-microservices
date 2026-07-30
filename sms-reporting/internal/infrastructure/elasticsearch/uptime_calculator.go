package elasticsearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sms-reporting/internal/domain"
	"sms-reporting/internal/infrastructure/logger"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/count"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

// ESUptimeCalculator implements domain.UptimeCalculator using Elasticsearch.
type ESRawUptimeProvider struct {
	client *esv8.TypedClient
	index  string
}

func NewESRawUptimeProvider(client *esv8.TypedClient, index string) domain.RawUptimeProvider {
	return &ESRawUptimeProvider{
		client: client,
		index:  index,
	}
}

func (c *ESRawUptimeProvider) CalculateRawUptimeStats(ctx context.Context, startTime time.Time, endTime time.Time) (int64, int64, error) {
	if c.client == nil {
		return 0, 0, nil
	}

	startStr := startTime.Format("2006-01-02T15:04:05Z")
	endStr := endTime.Format("2006-01-02T15:04:05Z")

	// Total Observations
	totalCountReq, err := c.client.Count().
		Index(c.index).
		Request(&count.Request{
			Query: &types.Query{
				Range: map[string]types.RangeQuery{
					"timestamp": types.DateRangeQuery{
						Gte: &startStr,
						Lte: &endStr,
					},
				},
			},
		}).Do(ctx)

	if err != nil {
		return 0, 0, fmt.Errorf("failed to count total observations: %w", err)
	}

	if totalCountReq.Count == 0 {
		return 0, 0, nil
	}

	// Success Observations
	successCountReq, err := c.client.Count().
		Index(c.index).
		Request(&count.Request{
			Query: &types.Query{
				Bool: &types.BoolQuery{
					Must: []types.Query{
						{
							Range: map[string]types.RangeQuery{
								"timestamp": types.DateRangeQuery{
									Gte: &startStr,
									Lte: &endStr,
								},
							},
						},
						{
							Term: map[string]types.TermQuery{
								"is_success": {Value: true},
							},
						},
					},
				},
			},
		}).Do(ctx)

	if err != nil {
		return 0, 0, fmt.Errorf("failed to count success observations: %w", err)
	}

	return successCountReq.Count, totalCountReq.Count, nil
}

func (c *ESRawUptimeProvider) CleanupOldData(ctx context.Context, olderThan time.Time) error {
	cleanupQuery := fmt.Sprintf(`{
		"query": {
			"range": {
				"timestamp": {
					"lt": "%s"
				}
			}
		}
	}`, olderThan.UTC().Format(time.RFC3339Nano))

	res, err := c.client.DeleteByQuery(c.index).Raw(strings.NewReader(cleanupQuery)).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute _delete_by_query: %w", err)
	}

	logger.Log.Sugar().Infof("Elasticsearch cleanup executed. Deleted docs: %d", res.Deleted)
	return nil
}
