package config

import "os"

type ElasticsearchConfig struct {
	URL         string
	ServerIndex string
}

type ObservationLoggerConfig struct {
	ChannelSize  int
	BatchSize    int
	FlushMs      int
	RetryMax     int
	RetryDelayMs int
}

func LoadElasticsearchConfig() ElasticsearchConfig {
	url := os.Getenv("ELASTICSEARCH_URL")
	if url == "" {
		url = "http://elasticsearch:9200"
	}

	serverIndex := os.Getenv("ELASTICSEARCH_SERVER_INDEX")
	if serverIndex == "" {
		serverIndex = "sms_observation_logs"
	}

	return ElasticsearchConfig{
		URL:         url,
		ServerIndex: serverIndex,
	}
}

func LoadObservationLoggerConfig() ObservationLoggerConfig {
	channelSize, _ := GetEnvInt("ES_OBSERVATION_CHANNEL_SIZE", 10000)
	batchSize, _ := GetEnvInt("ES_OBSERVATION_BATCH_SIZE", 500)
	flushMs, _ := GetEnvInt("ES_OBSERVATION_FLUSH_MS", 2000)
	retryMax, _ := GetEnvInt("ES_OBSERVATION_RETRY_MAX", 3)
	retryDelayMs, _ := GetEnvInt("ES_OBSERVATION_RETRY_DELAY_MS", 500)

	return ObservationLoggerConfig{
		ChannelSize:  channelSize,
		BatchSize:    batchSize,
		FlushMs:      flushMs,
		RetryMax:     retryMax,
		RetryDelayMs: retryDelayMs,
	}
}
