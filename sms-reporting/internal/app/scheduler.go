package app

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/google/uuid"

	"sms-reporting/internal/domain"
	"sms-reporting/internal/infrastructure/logger"
	"sms-reporting/internal/service"
)

type Scheduler struct {
	cron            *cron.Cron
	reportingWorker service.ReportingWorker
	cronSpec        string
	adminEmail      string
}

func NewScheduler(worker service.ReportingWorker, cronSpec string, adminEmail string) *Scheduler {
	if cronSpec == "" {
		cronSpec = "0 0 * * *" // Default: Midnight every day
	}
	if adminEmail == "" {
		adminEmail = "admin@example.com"
	}
	return &Scheduler{
		cron:            cron.New(cron.WithLocation(time.Local)),
		reportingWorker: worker,
		cronSpec:        cronSpec,
		adminEmail:      adminEmail,
	}
}

func (s *Scheduler) Start() error {
	logger.Log.Sugar().Infof("Starting daily reporting scheduler with cron spec: %s", s.cronSpec)

	_, err := s.cron.AddFunc(s.cronSpec, func() {
		logger.Log.Sugar().Info("Triggering daily report generation...")
		
		now := time.Now()
		startTime := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		
		req, err := domain.NewReportRequest(s.adminEmail, startTime, endTime, "cron-"+uuid.New().String())
		if err != nil {
			logger.Log.Sugar().Errorf("Failed to create daily report request: %v", err)
			return
		}

		s.reportingWorker.EnqueueReport(req)
		logger.Log.Sugar().Info("Daily report request enqueued successfully.")
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		logger.Log.Sugar().Info("Stopping daily reporting scheduler...")
		s.cron.Stop()
	}
}
