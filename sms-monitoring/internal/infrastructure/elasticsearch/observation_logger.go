package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"sms-monitoring/internal/config"

	"github.com/elastic/go-elasticsearch/v8"
)

type ObservationLog struct {
	ServerID  string    `json:"server_id"`
	IsSuccess bool      `json:"is_success"`
	Timestamp time.Time `json:"@timestamp"`
}

type ObservationLogger interface {
	LogObservation(ctx context.Context, serverID string, isSuccess bool)
	Shutdown()
}

type bufferedLogger struct {
	client        *elasticsearch.TypedClient
	index         string
	ch            chan ObservationLog
	wg            sync.WaitGroup
	batchSize     int
	flushInterval time.Duration
	retryMax      int
	retryDelay    time.Duration
	closeOnce     sync.Once
}

func NewObservationLogger(client *elasticsearch.TypedClient, index string, cfg config.ObservationLoggerConfig) ObservationLogger {
	l := &bufferedLogger{
		client:        client,
		index:         index,
		ch:            make(chan ObservationLog, cfg.ChannelSize),
		batchSize:     cfg.BatchSize,
		flushInterval: time.Duration(cfg.FlushMs) * time.Millisecond,
		retryMax:      cfg.RetryMax,
		retryDelay:    time.Duration(cfg.RetryDelayMs) * time.Millisecond,
	}
	l.wg.Add(1)
	go l.flusher()
	return l
}

// LogObservation is non-blocking. Drops silently if channel is full.
func (l *bufferedLogger) LogObservation(ctx context.Context, serverID string, isSuccess bool) {
	select {
	case l.ch <- ObservationLog{
		ServerID:  serverID,
		IsSuccess: isSuccess,
		Timestamp: time.Now().UTC(),
	}:
	default:
	}
}

func (l *bufferedLogger) Shutdown() {
	l.closeOnce.Do(func() {
		close(l.ch)
	})
	l.wg.Wait()
}

func (l *bufferedLogger) flusher() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.flushInterval)
	defer ticker.Stop()

	batch := make([]ObservationLog, 0, l.batchSize)

	for {
		select {
		case obs, ok := <-l.ch:
			if !ok {
				if len(batch) > 0 {
					l.flush(batch)
				}
				return
			}
			batch = append(batch, obs)
			if len(batch) >= l.batchSize {
				l.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				l.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

func (l *bufferedLogger) flush(batch []ObservationLog) {
	var buf bytes.Buffer
	for _, obs := range batch {
		action := fmt.Sprintf(`{"create":{"_index":"%s"}}`+"\n", l.index)
		buf.WriteString(action)
		doc, err := json.Marshal(obs)
		if err != nil {
			log.Printf("[ES] observation marshal failed for %s: %v", obs.ServerID, err)
			continue
		}
		buf.Write(doc)
		buf.WriteByte('\n')
	}

	if l.client == nil {
		return
	}

	for i := 0; i < l.retryMax; i++ {
		resp, err := l.client.Bulk().Raw(bytes.NewReader(buf.Bytes())).Do(context.Background())
		if err == nil && !resp.Errors {
			return
		}
		if err != nil {
			log.Printf("[ES] bulk flush error (retry %d/%d): %v", i+1, l.retryMax, err)
		} else {
			log.Printf("[ES] bulk flush had document errors (retry %d/%d)", i+1, l.retryMax)
		}
		time.Sleep(l.retryDelay)
	}
	log.Printf("[ES] bulk flush failed after %d retries", l.retryMax)
}

func InitILMAndDataStream(ctx context.Context, client *elasticsearch.TypedClient, indexName string, retentionDays int) error {
	if client == nil {
		return nil
	}

	policyName := "sms-logs-policy"
	templateName := "sms-logs-template"

	// ILM Policy
	policyBody := fmt.Sprintf(`{
		"policy": {
			"phases": {
				"hot": {
					"actions": {
						"rollover": {
							"max_age": "1d",
							"max_primary_shard_size": "50gb"
						}
					}
				},
				"delete": {
					"min_age": "%dd",
					"actions": {
						"delete": {}
					}
				}
			}
		}
	}`, retentionDays)

	resPolicy, err := client.Ilm.PutLifecycle(policyName).Raw(strings.NewReader(policyBody)).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to put ILM policy: %w", err)
	}
	if !resPolicy.Acknowledged {
		log.Printf("[ES] ILM policy creation was not acknowledged")
	}

	// Index Template for Data Stream
	templateBody := fmt.Sprintf(`{
		"index_patterns": ["%s*"],
		"data_stream": { },
		"template": {
			"settings": {
				"index.lifecycle.name": "%s"
			},
			"mappings": {
				"properties": {
					"@timestamp": {
						"type": "date"
					},
					"server_id": {
						"type": "keyword"
					},
					"is_success": {
						"type": "boolean"
					}
				}
			}
		}
	}`, indexName, policyName)

	resTemplate, err := client.Indices.PutIndexTemplate(templateName).Raw(strings.NewReader(templateBody)).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to put index template: %w", err)
	}
	if !resTemplate.Acknowledged {
		log.Printf("[ES] Index template creation was not acknowledged")
	}

	return nil
}
