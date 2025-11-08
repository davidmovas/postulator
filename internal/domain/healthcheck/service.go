package healthcheck

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/pkg/logger"
)

type service struct {
	siteService sites.Service
	repo        Repository
	logger      *logger.Logger
}

func NewService(
	siteService sites.Service,
	repo Repository,
	logger *logger.Logger,
) Service {
	return &service{
		siteService: siteService,
		repo:        repo,
		logger: logger.
			WithScope("service").
			WithScope("healthcheck"),
	}
}

func (s *service) CheckSiteHealth(ctx context.Context, site *entities.Site) (*entities.HealthCheckHistory, error) {
	s.logger.Infof("Checking health for site: %s (ID: %d)", site.Name, site.ID)

	healthCheck, err := s.siteService.CheckHealth(ctx, site.ID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to perform health check")
		return nil, err
	}

	history := &entities.HealthCheckHistory{
		SiteID:         site.ID,
		CheckedAt:      time.Now(),
		Status:         healthCheck.Status,
		ResponseTimeMs: int(healthCheck.ResponseTime.Milliseconds()),
		StatusCode:     healthCheck.Code,
		ErrorMessage:   healthCheck.Error,
	}

	if err = s.repo.SaveHistory(ctx, history); err != nil {
		s.logger.ErrorWithErr(err, "Failed to save health check history")
		return history, err
	}

	return history, nil
}

func (s *service) CheckAutoHealthSites(ctx context.Context) (unhealthy []*entities.Site, recovered []*entities.Site, err error) {
	s.logger.Info("Starting auto health check for all sites")

	allSites, err := s.siteService.ListSites(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get sites list")
		return nil, nil, err
	}

	var sitesToCheck []*entities.Site
	for _, site := range allSites {
		if site.Status == entities.StatusActive && site.AutoHealthCheck {
			sitesToCheck = append(sitesToCheck, site)
		}
	}

	if len(sitesToCheck) == 0 {
		s.logger.Info("No sites with auto health check enabled")
		return nil, nil, nil
	}

	s.logger.Infof("Found %d sites for auto health check", len(sitesToCheck))

	var unhealthySites []*entities.Site
	var recoveredSites []*entities.Site

	for _, site := range sitesToCheck {
		var lastCheck *entities.HealthCheckHistory
		lastCheck, err = s.repo.GetLastCheckBySite(ctx, site.ID)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to get last check for site")
			continue
		}

		wasUnhealthy := lastCheck != nil && lastCheck.Status == entities.HealthUnhealthy

		var history *entities.HealthCheckHistory
		history, err = s.CheckSiteHealth(ctx, site)
		if err != nil {
			s.logger.ErrorWithErr(err, "Health check failed for site")
			continue
		}

		isUnhealthyNow := history.Status == entities.HealthUnhealthy

		if isUnhealthyNow && !wasUnhealthy {
			// Сайт только что упал
			unhealthySites = append(unhealthySites, site)
		} else if !isUnhealthyNow && wasUnhealthy {
			// Сайт восстановился
			recoveredSites = append(recoveredSites, site)
		} else if isUnhealthyNow {
			// Сайт всё ещё down - добавляем в unhealthy для notifier (он сам решит уведомлять или нет)
			unhealthySites = append(unhealthySites, site)
		}
	}

	s.logger.Infof("Health check completed: %d unhealthy, %d recovered", len(unhealthySites), len(recoveredSites))

	return unhealthySites, recoveredSites, nil
}

func (s *service) GetSiteHistory(ctx context.Context, siteID int64, limit int) ([]*entities.HealthCheckHistory, error) {
	history, err := s.repo.GetHistoryBySite(ctx, siteID, limit)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site history")
		return nil, err
	}

	return history, nil
}
