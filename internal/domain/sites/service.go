package sites

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/infra/secret"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	wpClient wp.Client
	secret   *secret.Manager
	repo     Repository
	logger   *logger.Logger
}

func NewService(wpClient wp.Client, secretManager *secret.Manager, repo Repository, logger *logger.Logger) Service {
	return &service{
		wpClient: wpClient,
		secret:   secretManager,
		repo:     repo,
		logger: logger.
			WithScope("service").
			WithScope("sites"),
	}
}

func (s *service) CreateSite(ctx context.Context, site *entities.Site) error {
	if err := s.validateSite(site); err != nil {
		return err
	}

	encryptedPassword, err := s.secret.Encrypt(site.WPPassword)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to encrypt WordPress password")
		return errors.Internal(err)
	}

	site.WPPassword = encryptedPassword

	site.Status = entities.StatusActive
	site.HealthStatus = entities.HealthUnknown
	now := time.Now()
	site.CreatedAt = now
	site.UpdatedAt = now

	if err = s.repo.Create(ctx, site); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create site")
		return err
	}

	s.logger.Infof("Site created successfully %d", site.ID)
	return nil
}

func (s *service) GetSite(ctx context.Context, id int64) (*entities.Site, error) {
	site, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site")
		return nil, err
	}

	return site, nil
}

func (s *service) GetSiteWithPassword(ctx context.Context, id int64) (*entities.Site, error) {
	site, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site with password")
		return nil, err
	}

	if site.WPPassword != "" {
		var decryptedPassword string
		decryptedPassword, err = s.secret.Decrypt(site.WPPassword)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to decrypt WordPress password")
			return nil, errors.Internal(err)
		}
		site.WPPassword = decryptedPassword
	}

	return site, nil
}

func (s *service) ListSites(ctx context.Context) ([]*entities.Site, error) {
	sites, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list sites")
		return nil, err
	}

	s.logger.Debug("Sites listed")
	return sites, nil
}

func (s *service) UpdateSite(ctx context.Context, site *entities.Site) error {
	if err := s.validateSite(site); err != nil {
		return err
	}

	existingSite, err := s.repo.GetByID(ctx, site.ID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get existing site for update")
		return err
	}

	if site.WPPassword != "" && site.WPPassword != existingSite.WPPassword {
		var encryptedPassword string
		encryptedPassword, err = s.secret.Encrypt(site.WPPassword)
		if err != nil {
			s.logger.ErrorWithErr(err, "Failed to encrypt WordPress password during update")
			return errors.Internal(err)
		}
		site.WPPassword = encryptedPassword
	} else {
		site.WPPassword = existingSite.WPPassword
	}

	site.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, site); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update site")
		return err
	}

	return nil
}

func (s *service) UpdateSitePassword(ctx context.Context, id int64, password string) error {
	existingSite, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for password update")
		return err
	}

	encryptedPassword, err := s.secret.Encrypt(password)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to encrypt WordPress password")
		return errors.Internal(err)
	}

	existingSite.WPPassword = encryptedPassword
	existingSite.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, existingSite); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update site password")
		return err
	}

	return nil
}

func (s *service) DeleteSite(ctx context.Context, id int64) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for deletion")
		return err
	}

	if err = s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete site")
		return err
	}

	s.logger.Info("Site deleted successfully")
	return nil
}

func (s *service) CheckHealth(ctx context.Context, siteID int64) error {
	site, err := s.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site for health check")
		return err
	}

	healthStatus, err := s.performHealthCheck(ctx, site)
	if err != nil {
		if updateErr := s.repo.UpdateHealthStatus(ctx, siteID, healthStatus, time.Now()); updateErr != nil {
			s.logger.ErrorWithErr(updateErr, "Failed to update health status")
			return updateErr
		}

		return errors.SiteUnreachable(site.URL, err)
	}

	if updateErr := s.repo.UpdateHealthStatus(ctx, siteID, healthStatus, time.Now()); updateErr != nil {
		s.logger.ErrorWithErr(updateErr, "Failed to update health status")
		return updateErr
	}

	return nil
}

func (s *service) CheckAllHealth(ctx context.Context) error {
	sites, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get sites for health check")
		return err
	}

	var lastErr error
	for _, site := range sites {
		if site.Status != entities.StatusActive {
			continue
		}

		if err = s.CheckHealth(ctx, site.ID); err != nil {
			lastErr = err
			s.logger.ErrorWithErr(err, "Health check failed for site")
		}
	}

	if lastErr != nil {
		return errors.New(errors.ErrCodeInternal, "Some health checks failed")
	}

	return nil
}

func (s *service) validateSite(site *entities.Site) error {
	if strings.TrimSpace(site.Name) == "" {
		return errors.Validation("Site name is required")
	}

	if strings.TrimSpace(site.URL) == "" {
		return errors.Validation("Site URL is required")
	}

	parsedURL, err := url.Parse(site.URL)
	if err != nil {
		return errors.Validation("Invalid URL format")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.Validation("URL must use http or https protocol")
	}

	if strings.TrimSpace(site.WPUsername) == "" {
		return errors.Validation("WordPress username is required")
	}

	if strings.TrimSpace(site.WPPassword) == "" {
		return errors.Validation("WordPress password is required")
	}

	return nil
}

func (s *service) performHealthCheck(ctx context.Context, site *entities.Site) (entities.HealthStatus, error) {
	return s.wpClient.CheckHealth(ctx, site)
}
