package app

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"

	"sms-reporting/internal/infrastructure/database"
	"sms-reporting/internal/infrastructure/logger"
	"sms-reporting/internal/service"
)

const schedulerLockKey = "lock:reporting_scheduler"

type Scheduler struct {
	cron             *cron.Cron
	reportingService service.ReportingService
	redisClient      redis.UniversalClient
	cronSpec         string
	adminEmail       string
}

func NewScheduler(reportingService service.ReportingService, redisClient redis.UniversalClient, cronSpec string, adminEmail string) *Scheduler {
	if cronSpec == "" {
		cronSpec = "0 0 * * *"
	}
	if adminEmail == "" {
		adminEmail = "admin@example.com"
	}
	return &Scheduler{
		cron:             cron.New(cron.WithLocation(time.Local)),
		reportingService: reportingService,
		redisClient:      redisClient,
		cronSpec:         cronSpec,
		adminEmail:       adminEmail,
	}
}

func (s *Scheduler) Start() error {
	logger.Log.Sugar().Infof("Starting daily reporting scheduler with cron spec: %s", s.cronSpec)

	_, err := s.cron.AddFunc(s.cronSpec, func() {
		acquired, err := database.AcquireLock(context.Background(), s.redisClient, schedulerLockKey, 30*time.Second)
		if err != nil {
			logger.Log.Sugar().Errorf("Failed to acquire scheduler lock: %v", err)
			return
		}
		if !acquired {
			logger.Log.Sugar().Info("Another instance already triggered the daily report. Skipping.")
			return
		}

		logger.Log.Sugar().Info("Triggering daily report generation (leader elected)...")

		now := time.Now()
		startTime := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		if err := s.reportingService.ExecuteDailyMaintenance(context.Background(), startTime, endTime); err != nil {
			logger.Log.Sugar().Errorf("Failed to execute daily maintenance: %v", err)
		}

		err = s.reportingService.RequestReport(context.Background(), s.adminEmail, startTime.Format("2006-01-02"), startTime.Format("2006-01-02"))
		if err != nil {
			logger.Log.Sugar().Errorf("Failed to request daily report: %v", err)
		} else {
			logger.Log.Sugar().Info("Daily report request enqueued successfully")
		}
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


