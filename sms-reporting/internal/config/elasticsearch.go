package config

import "os"

type ElasticsearchConfig struct {
	URL         string
	ServerIndex string
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
