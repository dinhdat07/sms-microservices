package elasticsearch

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/assert"
)

// mockRoundTripper intercepts HTTP requests for Elasticsearch
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestESRawUptimeProvider_CalculateRawUptimeStats_NilClient(t *testing.T) {
	calc := NewESRawUptimeProvider(nil, "observations")
	success, total, err := calc.CalculateRawUptimeStats(context.Background(), time.Now(), time.Now())
	
	assert.NoError(t, err)
	assert.Equal(t, int64(0), success)
	assert.Equal(t, int64(0), total)
}

func TestESRawUptimeProvider_CalculateRawUptimeStats_Success(t *testing.T) {
	requestCount := 0
	transport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			requestCount++
			var body string
			if requestCount == 1 {
				// First request: count all
				body = `{"count": 100}`
			} else {
				// Second request: count success
				body = `{"count": 99}`
			}
			res := &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}
			res.Header.Set("X-Elastic-Product", "Elasticsearch")
			return res, nil
		},
	}

	cfg := esv8.Config{
		Transport: transport,
	}
	es, _ := esv8.NewTypedClient(cfg)

	calc := NewESRawUptimeProvider(es, "observations")

	success, total, err := calc.CalculateRawUptimeStats(context.Background(), time.Now(), time.Now())
	
	assert.NoError(t, err)
	assert.Equal(t, int64(99), success)
	assert.Equal(t, int64(100), total)
}

func TestESRawUptimeProvider_CalculateRawUptimeStats_TotalCountError(t *testing.T) {
	transport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, assert.AnError
		},
	}

	cfg := esv8.Config{
		Transport: transport,
	}
	es, _ := esv8.NewTypedClient(cfg)

	calc := NewESRawUptimeProvider(es, "observations")
	success, total, err := calc.CalculateRawUptimeStats(context.Background(), time.Now(), time.Now())
	
	assert.Error(t, err)
	assert.Equal(t, int64(0), success)
	assert.Equal(t, int64(0), total)
}

func TestESRawUptimeProvider_CalculateRawUptimeStats_SuccessCountError(t *testing.T) {
	requestCount := 0
	transport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			requestCount++
			if requestCount == 1 {
				body := `{"count": 100}`
				res := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}
				res.Header.Set("X-Elastic-Product", "Elasticsearch")
				return res, nil
			}
			
			return nil, assert.AnError
		},
	}

	cfg := esv8.Config{
		Transport: transport,
	}
	es, _ := esv8.NewTypedClient(cfg)

	calc := NewESRawUptimeProvider(es, "observations")
	success, total, err := calc.CalculateRawUptimeStats(context.Background(), time.Now(), time.Now())
	
	assert.Error(t, err)
	assert.Equal(t, int64(0), success)
	assert.Equal(t, int64(0), total)
}

