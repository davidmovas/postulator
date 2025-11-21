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
	ctx                context.Context
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
	s.ctx = ctx

	if s.running {
		s.mu.Unlock()
		s.logger.Warn("Scheduler already running")
		return nil
	}
	s.mu.Unlock()

	s.logger.Info("Starting job scheduler")

	settings, err := s.settingsService.GetHealthCheckSettings(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get health check settings")
		return err
	}

	if !settings.Enabled {
		s.logger.Info("Health check scheduler is disabled in settings; waiting for enable")
		return nil
	}

	interval := time.Duration(settings.IntervalMinutes) * time.Minute

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.ticker = time.NewTicker(interval)
	s.stopChan = make(chan struct{})
	s.running = true

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
		return nil
	}

	if intervalMinutes <= 0 {
		intervalMinutes = 1
	}

	select {
	case s.updateIntervalChan <- intervalMinutes:
	default:
		select {
		case <-s.updateIntervalChan:
		default:
		}
		s.updateIntervalChan <- intervalMinutes
	}

	return nil
}

func (s *scheduler) ApplySettings(ctx context.Context, enabled bool, intervalMinutes int) error {
	if intervalMinutes <= 0 {
		intervalMinutes = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx != nil {
		s.ctx = ctx
	}
	if s.ctx == nil {
		s.ctx = context.Background()
	}

	if !enabled {
		if s.running {
			s.logger.Info("Disabling health check scheduler")
			s.running = false

			if s.ticker != nil {
				s.ticker.Stop()
				s.ticker = nil
			}

			if s.stopChan != nil {
				close(s.stopChan)
				s.stopChan = make(chan struct{})
			}
		}
		return nil
	}

	if !s.running {
		s.logger.Infof("Enabling health check scheduler with interval: %d minutes", intervalMinutes)

		if s.ticker != nil {
			s.ticker.Stop()
		}

		s.ticker = time.NewTicker(time.Duration(intervalMinutes) * time.Minute)

		if s.stopChan == nil {
			s.stopChan = make(chan struct{})
		} else {
			select {
			case <-s.stopChan:
				s.stopChan = make(chan struct{})
			default:
			}
		}

		s.running = true

		go s.run(s.ctx)

		s.logger.Info("Health check scheduler enabled and started")
		return nil
	}

	s.logger.Infof("Updating health check interval to: %d minutes", intervalMinutes)
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.ticker = time.NewTicker(time.Duration(intervalMinutes) * time.Minute)

	return nil
}

func (s *scheduler) run(ctx context.Context) {
	s.logger.Debug("Health check run loop started")

	for {
		select {
		case <-s.stopChan:
			s.logger.Debug("Health check run loop stopped")
			return

		case newInterval := <-s.updateIntervalChan:
			s.logger.Infof("Updating interval to: %d minutes", newInterval)
			s.mu.Lock()
			if s.ticker != nil {
				s.ticker.Stop()
			}
			s.ticker = time.NewTicker(time.Duration(newInterval) * time.Minute)
			s.mu.Unlock()

		case <-s.ticker.C:
			s.performCheck(ctx)
		}
	}
}

func (s *scheduler) performCheck(ctx context.Context) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return
	}

	settings, err := s.settingsService.GetHealthCheckSettings(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get settings during check")
		return
	}

	if !settings.Enabled {
		return
	}

	unhealthy, recovered, err := s.service.CheckAutoHealthSites(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to check auto health sites")
		return
	}

	isWindowVisible := s.visibilityChecker()
	shouldNotify := settings.NotifyAlways || (settings.NotifyWhenHidden && !isWindowVisible)

	if shouldNotify {
		if len(unhealthy) > 0 {
			if err = s.notifier.NotifyUnhealthySites(ctx, unhealthy, settings.NotifyWithSound); err != nil {
				s.logger.ErrorWithErr(err, "Failed to send unhealthy notification")
			}
		}

		if len(unhealthy) > 0 && len(recovered) > 0 && settings.NotifyOnRecover {
			time.Sleep(2 * time.Second)
		}

		if settings.NotifyOnRecover && len(recovered) > 0 {
			if err = s.notifier.NotifyRecoveredSites(ctx, recovered, settings.NotifyWithSound); err != nil {
				s.logger.ErrorWithErr(err, "Failed to send recovery notification")
			}
		}
	}
}
