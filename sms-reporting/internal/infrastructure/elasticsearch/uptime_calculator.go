package elasticsearch

import (
	"context"
	"fmt"
	"time"

	"sms-reporting/internal/domain"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/count"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

// ESUptimeCalculator implements domain.UptimeCalculator using Elasticsearch.
type ESUptimeCalculator struct {
	esClient *esv8.TypedClient
	esIndex  string
}

func NewESUptimeCalculator(esClient *esv8.TypedClient, esIndex string) domain.UptimeCalculator {
	return &ESUptimeCalculator{esClient: esClient, esIndex: esIndex}
}

func (c *ESUptimeCalculator) CalculateUptime(ctx context.Context, startTime, endTime time.Time) (float64, error) {
	if c.esClient == nil {
		return 0, nil
	}

	startStr := startTime.Format("2006-01-02T15:04:05Z")
	endStr := endTime.Format("2006-01-02T15:04:05Z")

	// Total Observations
	totalCountReq, err := c.esClient.Count().
		Index(c.esIndex).
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
		return 0, fmt.Errorf("failed to count total observations: %w", err)
	}

	if totalCountReq.Count == 0 {
		return 0, nil
	}

	// Success Observations
	successCountReq, err := c.esClient.Count().
		Index(c.esIndex).
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
		return 0, fmt.Errorf("failed to count success observations: %w", err)
	}

	return (float64(successCountReq.Count) / float64(totalCountReq.Count)) * 100, nil
}
