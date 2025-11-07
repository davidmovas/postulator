package healthcheck

import (
	"context"
	"sync"
	"time"

	settingsSvc "github.com/davidmovas/postulator/internal/domain/settings"
	"github.com/davidmovas/postulator/pkg/logger"
)

type scheduler struct {
	service           Service
	settingsService   settingsSvc.Service
	notifier          Notifier
	visibilityChecker WindowVisibilityChecker
	logger            *logger.Logger

	ticker             *time.Ticker
	stopChan           chan struct{}
	updateIntervalChan chan int
	running            bool
	mu                 sync.RWMutex
}

func NewScheduler(
	service Service,
	settingsService settingsSvc.Service,
	notifier Notifier,
	visibilityChecker WindowVisibilityChecker,
	logger *logger.Logger,
) Scheduler {
	return &scheduler{
		service:           service,
		settingsService:   settingsService,
		notifier:          notifier,
		visibilityChecker: visibilityChecker,
		logger: logger.
			WithScope("scheduler").
			WithScope("healthcheck"),
		stopChan:           make(chan struct{}),
		updateIntervalChan: make(chan int, 1),
	}
}

func (s *scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Warn("Scheduler already running")
		return nil
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting health check scheduler")

	settings, err := s.settingsService.GetHealthCheckSettings(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get health check settings")
		return err
	}

	if !settings.Enabled {
		s.logger.Info("Health check is disabled, scheduler not started")
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return nil
	}

	interval := time.Duration(settings.IntervalMinutes) * time.Minute
	s.ticker = time.NewTicker(interval)

	s.logger.Infof("Health check scheduler started with interval: %v", interval)

	go s.run(ctx)

	return nil
}

func (s *scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		s.logger.Warn("Scheduler not running")
		return nil
	}

	s.logger.Info("Stopping health check scheduler")
	s.running = false

	if s.ticker != nil {
		s.ticker.Stop()
	}

	close(s.stopChan)

	return nil
}

func (s *scheduler) UpdateInterval(intervalMinutes int) error {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		s.logger.Info("Scheduler not running, interval will be applied on next start")
		return nil
	}

	s.logger.Infof("Updating health check interval to %d minutes", intervalMinutes)
	s.updateIntervalChan <- intervalMinutes

	return nil
}

func (s *scheduler) run(ctx context.Context) {
	for {
		select {
		case <-s.stopChan:
			s.logger.Info("Health check scheduler stopped")
			return

		case newInterval := <-s.updateIntervalChan:
			s.logger.Infof("Received interval update: %d minutes", newInterval)
			if s.ticker != nil {
				s.ticker.Stop()
			}
			s.ticker = time.NewTicker(time.Duration(newInterval) * time.Minute)
			s.logger.Info("Ticker updated with new interval")

		case <-s.ticker.C:
			s.logger.Info("Health check tick triggered")
			s.performCheck(ctx)
		}
	}
}

func (s *scheduler) performCheck(ctx context.Context) {
	settings, err := s.settingsService.GetHealthCheckSettings(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get settings during check")
		return
	}

	if !settings.Enabled {
		s.logger.Info("Health check disabled, skipping check")
		return
	}

	isWindowVisible := s.visibilityChecker()
	s.logger.Infof("Window visible: %v", isWindowVisible)

	shouldNotify := settings.NotifyAlways || (settings.NotifyWhenHidden && !isWindowVisible)

	if !shouldNotify {
		s.logger.Info("Notifications disabled for current window state, skipping notifications")
	}

	unhealthy, recovered, err := s.service.CheckAutoHealthSites(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to check auto health sites")
		return
	}

	if shouldNotify {
		if len(unhealthy) > 0 {
			if err = s.notifier.NotifyUnhealthySites(ctx, unhealthy, settings.NotifyWithSound); err != nil {
				s.logger.ErrorWithErr(err, "Failed to send unhealthy notification")
			}
		}

		if settings.NotifyOnRecover && len(recovered) > 0 {
			if err = s.notifier.NotifyRecoveredSites(ctx, recovered, settings.NotifyWithSound); err != nil {
				s.logger.ErrorWithErr(err, "Failed to send recovery notification")
			}
		}
	}
}
