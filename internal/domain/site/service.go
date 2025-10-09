package site

import (
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/secret"
	"Postulator/internal/infra/wp"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
)

var _ IService = (*Service)(nil)

type Service struct {
	siteRepo     ISiteRepository
	categoryRepo ICategoryRepository
	wpClient     *wp.Client
	secrets      *secret.Manager
	logger       *logger.Logger
}

func NewService(c di.Container) (*Service, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	siteRepo, err := NewSiteRepository(c)
	if err != nil {
		return nil, err
	}

	categoryRepo, err := NewCategoryRepository(c)
	if err != nil {
		return nil, err
	}

	var wpClient *wp.Client
	if err = c.Resolve(&wpClient); err != nil {
		return nil, err
	}

	sec, err := secret.NewManager()
	if err != nil {
		return nil, err
	}

	return &Service{
		siteRepo:     siteRepo,
		categoryRepo: categoryRepo,
		wpClient:     wpClient,
		secrets:      sec,
		logger:       l,
	}, nil
}

func (s *Service) CreateSite(ctx context.Context, site *entities.Site) error {
	// Encrypt password if provided
	if site.WPPassword != "" {
		enc, err := s.secrets.Encrypt(site.WPPassword)
		if err != nil {
			return errors.Internal(err)
		}
		site.WPPassword = enc
	}
	return s.siteRepo.Create(ctx, site)
}

func (s *Service) GetSite(ctx context.Context, id int64) (*entities.Site, error) {
	return s.siteRepo.GetByID(ctx, id)
}

func (s *Service) ListSites(ctx context.Context) ([]*entities.Site, error) {
	return s.siteRepo.GetAll(ctx)
}

func (s *Service) UpdateSite(ctx context.Context, site *entities.Site) error {
	// If password is provided in update, encrypt it
	if site.WPPassword != "" {
		enc, err := s.secrets.Encrypt(site.WPPassword)
		if err != nil {
			return errors.Internal(err)
		}
		site.WPPassword = enc
	}
	return s.siteRepo.Update(ctx, site)
}

func (s *Service) DeleteSite(ctx context.Context, id int64) error {
	return s.siteRepo.Delete(ctx, id)
}

func (s *Service) GetSiteWithPassword(ctx context.Context, id int64) (*entities.Site, error) {
	siteEnt, err := s.siteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if siteEnt.WPPassword == "" {
		return siteEnt, nil
	}
	plain, err := s.secrets.Decrypt(siteEnt.WPPassword)
	if err != nil {
		return nil, errors.Internal(err)
	}
	copy := *siteEnt
	copy.WPPassword = plain
	return &copy, nil
}

func (s *Service) UpdateSitePassword(ctx context.Context, id int64, password string) error {
	if id == 0 {
		return errors.Validation("invalid site id")
	}
	enc, err := s.secrets.Encrypt(password)
	if err != nil {
		return errors.Internal(err)
	}
	siteEnt, err := s.siteRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	siteEnt.WPPassword = enc
	return s.siteRepo.Update(ctx, siteEnt)
}

func (s *Service) CheckHealth(ctx context.Context, siteID int64) error {
	site, err := s.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		return err
	}

	var healthStatus entities.HealthStatus
	if err = s.wpClient.CheckHealth(ctx, site); err == nil {
		healthStatus = entities.HealthStatusHealthy
	} else {
		healthStatus = entities.HealthStatusUnhealthy
	}

	if updateErr := s.siteRepo.UpdateHealthStatus(ctx, siteID, healthStatus); updateErr != nil {
		return updateErr
	}

	return err
}

func (s *Service) SyncCategories(ctx context.Context, siteID int64) error {
	site, err := s.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		return err
	}

	categories, err := s.wpClient.GetCategories(ctx, site)
	if err != nil {
		return err
	}

	for _, category := range categories {
		if err = s.categoryRepo.Create(ctx, category); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetSiteCategories(ctx context.Context, siteID int64) ([]*entities.Category, error) {
	return s.categoryRepo.GetBySiteID(ctx, siteID)
}
