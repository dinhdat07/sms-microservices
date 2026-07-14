package domain

import "context"

type ReportNotifier interface {
	SendReportEmail(ctx context.Context, toEmail string, subject string, htmlBody string) error
}
