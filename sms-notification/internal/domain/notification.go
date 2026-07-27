package domain

import "context"

const (
	EventNotificationRequested = "NotificationRequested"
)

type NotificationEvent struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type NotificationSender interface {
	SendEmail(ctx context.Context, to string, subject string, htmlBody string) error
}
