package service

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"sync"

	"sms-reporting/internal/domain"
	"sms-reporting/internal/infrastructure/logger"
	"sms-reporting/internal/repository"

	"go.uber.org/zap"
)

//go:embed templates/status_report.html
var statusReportTemplate string

type TemplateData struct {
	StartDate      string
	EndDate        string
	PeriodDays     int
	TotalServers   int64
	OnlineServers  int64
	OfflineServers int64
	UptimePercent  float64
	OnlinePercent  float64
	OfflinePercent float64
	HealthLabel    string
}

type ReportingWorker interface {
	Start(ctx context.Context)
	Stop()
	EnqueueReport(req *domain.ReportRequest)
}

type reportingWorkerImpl struct {
	repo        repository.ReportingRepository
	uptimeCalc  domain.UptimeCalculator
	jobQueue    chan *domain.ReportRequest
	workerCount int
	notifier    domain.ReportNotifier
	wg          sync.WaitGroup
}

func NewReportingWorker(repo repository.ReportingRepository, uptimeCalc domain.UptimeCalculator, workerCount int, jobQueueSize int, notifier domain.ReportNotifier) ReportingWorker {
	if workerCount <= 0 {
		workerCount = 5 // Default to 5 concurrent workers
	}
	if jobQueueSize <= 0 {
		jobQueueSize = 100 // Default to 100 capacity
	}
	return &reportingWorkerImpl{
		repo:        repo,
		uptimeCalc:  uptimeCalc,
		jobQueue:    make(chan *domain.ReportRequest, jobQueueSize), // Buffered queue
		workerCount: workerCount,
		notifier:    notifier,
	}
}

func (w *reportingWorkerImpl) Start(ctx context.Context) {
	logger.Log.Sugar().Infof("[ReportingWorker] Starting pool with %d workers", w.workerCount)
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(ctx, i)
	}
}

func (w *reportingWorkerImpl) Stop() {
	logger.Log.Sugar().Info("[ReportingWorker] Stopping pool...")
	close(w.jobQueue)
	w.wg.Wait()
	logger.Log.Sugar().Info("[ReportingWorker] All workers stopped.")
}

func (w *reportingWorkerImpl) EnqueueReport(req *domain.ReportRequest) {
	w.jobQueue <- req
}

func (w *reportingWorkerImpl) worker(ctx context.Context, id int) {
	defer w.wg.Done()
	logger.Log.Info("ReportingWorker Started", zap.Int("worker_id", id))
	for req := range w.jobQueue {
		w.processReport(ctx, req, id)
	}
	logger.Log.Info("ReportingWorker Stopped", zap.Int("worker_id", id))
}

func (w *reportingWorkerImpl) processReport(ctx context.Context, req *domain.ReportRequest, workerID int) {
	logger.Log.Info("Processing report", zap.Int("worker_id", workerID), zap.String("report_id", req.ID.String()))

	err := w.repo.UpdateReportStatus(ctx, req.ID.String(), domain.ReportStatusProcessing)
	if err != nil {
		logger.Log.Error("Failed to update status to PROCESSING", zap.Int("worker_id", workerID), zap.Error(err))
		return
	}

	err = w.doWork(ctx, req)

	finalStatus := domain.ReportStatusCompleted
	if err != nil {
		logger.Log.Error("Report failed", zap.Int("worker_id", workerID), zap.String("report_id", req.ID.String()), zap.Error(err))
		finalStatus = domain.ReportStatusFailed
	}

	err = w.repo.UpdateReportStatus(ctx, req.ID.String(), finalStatus)
	if err != nil {
		logger.Log.Error("Failed to update final status", zap.Int("worker_id", workerID), zap.Error(err))
	} else {
		logger.Log.Info("Finished report", zap.Int("worker_id", workerID), zap.String("report_id", req.ID.String()))
	}
}

func (w *reportingWorkerImpl) doWork(ctx context.Context, req *domain.ReportRequest) error {
	// 1. Get Server Stats from LOCAL reporting_servers table
	totalServers, err := w.repo.GetServerCountByStatus(ctx, "")
	if err != nil {
		return err
	}
	onlineServers, err := w.repo.GetServerCountByStatus(ctx, "ONLINE")
	if err != nil {
		return err
	}
	offlineServers, err := w.repo.GetServerCountByStatus(ctx, "OFFLINE")
	if err != nil {
		return err
	}

	// 2. Get Uptime from Elasticsearch
	uptimePercent, err := w.uptimeCalc.CalculateUptime(ctx, req.StartTime, req.EndTime)
	if err != nil {
		return err
	}

	// 3. Render and Send HTML Email using template
	tmpl, err := template.New("status_report").Parse(statusReportTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	data := TemplateData{
		StartDate:      req.StartTime.Format("2006-01-02"),
		EndDate:        req.EndTime.Format("2006-01-02"),
		PeriodDays:     int(req.EndTime.Sub(req.StartTime).Hours()/24) + 1,
		TotalServers:   totalServers,
		OnlineServers:  onlineServers,
		OfflineServers: offlineServers,
		UptimePercent:  uptimePercent,
		OnlinePercent:  percentage(onlineServers, totalServers),
		OfflinePercent: percentage(offlineServers, totalServers),
		HealthLabel:    healthLabel(uptimePercent),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}
	htmlStr := buf.String()

	logger.Log.Sugar().Infof("[ReportingWorker] Sending email to %s", req.RequestorEmail)

	// Call internal Notification via domain.ReportNotifier interface
	if w.notifier != nil {
		err := w.notifier.SendReportEmail(ctx, req.RequestorEmail, "Server Status Report", htmlStr)
		if err != nil {
			logger.Log.Sugar().Errorf("[ReportingWorker] Error sending email: %v", err)
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	return nil
}

func percentage(value, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) * 100 / float64(total)
}

func healthLabel(uptime float64) string {
	switch {
	case uptime >= 99:
		return "Healthy"
	case uptime >= 95:
		return "Attention Recommended"
	default:
		return "Action Required"
	}
}
