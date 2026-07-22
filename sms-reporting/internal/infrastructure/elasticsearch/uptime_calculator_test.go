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

func TestESUptimeCalculator_CalculateUptime_NilClient(t *testing.T) {
	calc := NewESUptimeCalculator(nil, "observations")
	uptime, err := calc.CalculateUptime(context.Background(), time.Now(), time.Now())
	
	assert.NoError(t, err)
	assert.Equal(t, float64(0), uptime)
}

func TestESUptimeCalculator_CalculateUptime_Success(t *testing.T) {
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
	// mock the elastictransport Logger to avoid nil pointer panic inside TypedClient when error occurs sometimes
	// (removed because logger is unexported in the newer version)

	calc := NewESUptimeCalculator(es, "observations")

	uptime, err := calc.CalculateUptime(context.Background(), time.Now(), time.Now())
	
	assert.NoError(t, err)
	assert.Equal(t, 99.0, uptime)
}

func TestESUptimeCalculator_CalculateUptime_TotalCountError(t *testing.T) {
	transport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, assert.AnError
		},
	}

	cfg := esv8.Config{
		Transport: transport,
	}
	es, _ := esv8.NewTypedClient(cfg)

	calc := NewESUptimeCalculator(es, "observations")
	uptime, err := calc.CalculateUptime(context.Background(), time.Now(), time.Now())
	
	assert.Error(t, err)
	assert.Equal(t, float64(0), uptime)
}

func TestESUptimeCalculator_CalculateUptime_ZeroTotalCount(t *testing.T) {
	transport := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			body := `{"count": 0}`
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

	calc := NewESUptimeCalculator(es, "observations")
	uptime, err := calc.CalculateUptime(context.Background(), time.Now(), time.Now())
	
	assert.NoError(t, err)
	assert.Equal(t, float64(0), uptime)
}

func TestESUptimeCalculator_CalculateUptime_SuccessCountError(t *testing.T) {
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

	calc := NewESUptimeCalculator(es, "observations")
	uptime, err := calc.CalculateUptime(context.Background(), time.Now(), time.Now())
	
	assert.Error(t, err)
	assert.Equal(t, float64(0), uptime)
}
