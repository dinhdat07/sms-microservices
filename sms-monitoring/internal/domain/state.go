package domain

// ServerState represents the cached status and retry count.
type ServerState struct {
	Status     string
	RetryCount int
}
