package categories

import (
	"context"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	wp          wp.Client
	siteService sites.Service
	repo        Repository
	statsRepo   StatisticsRepository
	logger      *logger.Logger
}

func NewService(
	wp wp.Client,
	siteService sites.Service,
	repo Repository,
	statsRepo StatisticsRepository,
	logger *logger.Logger,
) Service {
	return &service{
		wp:          wp,
		siteService: siteService,
		repo:        repo,
		statsRepo:   statsRepo,
		logger:      logger.WithScope("service").WithScope("categories"),
	}
}

func (s *service) CreateCategory(ctx context.Context, category *entities.Category) error {
	if err := s.validateCategory(category); err != nil {
		return err
	}

	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	if err := s.repo.Create(ctx, category); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create category")
		return err
	}

	s.logger.Infof("Category created successfull %d", category.ID)
	return nil
}

func (s *service) GetCategory(ctx context.Context, id int64) (*entities.Category, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get category")
		return nil, err
	}

	return category, nil
}

func (s *service) ListSiteCategories(ctx context.Context, siteID int64) ([]*entities.Category, error) {
	categories, err := s.repo.GetBySiteID(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to list site categories")
		return nil, err
	}

	return categories, nil
}

func (s *service) UpdateCategory(ctx context.Context, category *entities.Category) error {
	if err := s.validateCategory(category); err != nil {
		return err
	}

	category.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, category); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update category")
		return err
	}

	s.logger.Info("Category updated successfully")
	return nil
}

func (s *service) DeleteCategory(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete category")
		return err
	}

	s.logger.Info("Category deleted successfully")
	return nil
}

func (s *service) SyncFromWordPress(ctx context.Context, siteID int64) error {
	site, err := s.siteService.GetSiteWithPassword(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site")
		return err
	}

	localCategories, err := s.repo.GetBySiteID(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get local categories")
		return err
	}

	remoteCategories, err := s.wp.GetCategories(ctx, site)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get categories from WordPress")
		return err
	}

	localMap := make(map[int]*entities.Category)
	for _, cat := range localCategories {
		localMap[cat.WPCategoryID] = cat
	}

	remoteMap := make(map[int]*entities.Category)
	for _, cat := range remoteCategories {
		remoteMap[cat.WPCategoryID] = cat
	}

	// Категории для создания (есть в удаленке, но нет в локальной)
	var categoriesToCreate []*entities.Category
	for wpID, remoteCat := range remoteMap {
		if _, exists := localMap[wpID]; !exists {
			remoteCat.SiteID = siteID
			categoriesToCreate = append(categoriesToCreate, remoteCat)
		}
	}

	// Категории для удаления (есть в локальной, но нет в удаленке)
	var categoriesToDelete []int64
	for wpID, localCat := range localMap {
		if _, exists := remoteMap[wpID]; !exists {
			categoriesToDelete = append(categoriesToDelete, localCat.ID)
		}
	}

	if len(categoriesToCreate) > 0 {
		if err = s.repo.BulkUpsert(ctx, siteID, categoriesToCreate); err != nil {
			s.logger.ErrorWithErr(err, "Failed to bulk upsert categories")
			return err
		}
		s.logger.Infof("Categories created during sync: %d", len(categoriesToCreate))
	}

	if len(categoriesToDelete) > 0 {
		for _, catID := range categoriesToDelete {
			if err = s.repo.Delete(ctx, catID); err != nil {
				s.logger.ErrorWithErr(err, "Failed to delete category during sync")
				return err
			}
		}
		s.logger.Infof("Categories deleted during sync: %d", len(categoriesToDelete))
	}

	s.logger.Info("Categories sync completed successfully")
	return nil
}

func (s *service) CreateInWordPress(ctx context.Context, category *entities.Category) error {
	site, err := s.siteService.GetSiteWithPassword(ctx, category.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site")
		return err
	}

	if err = s.wp.CreateCategory(ctx, site, category); err != nil {
		s.logger.ErrorWithErr(err, "Failed to create category in WordPress")
		return err
	}

	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	if err = s.repo.Create(ctx, category); err != nil {
		s.logger.ErrorWithErr(err, "Failed to save created category locally")
		return err
	}

	s.logger.Info("Category created in WordPress successfully")
	return nil
}

func (s *service) UpdateInWordPress(ctx context.Context, category *entities.Category) error {
	site, err := s.siteService.GetSiteWithPassword(ctx, category.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site")
		return err
	}

	if err = s.wp.UpdateCategory(ctx, site, category); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update category in WordPress")
		return err
	}

	category.UpdatedAt = time.Now()

	if err = s.repo.Update(ctx, category); err != nil {
		s.logger.ErrorWithErr(err, "Failed to update category locally")
		return err
	}

	s.logger.Info("Category updated in WordPress successfully")
	return nil
}

func (s *service) DeleteInWordPress(ctx context.Context, categoryID int64) error {
	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get category")
		return err
	}

	site, err := s.siteService.GetSiteWithPassword(ctx, category.SiteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site")
		return err
	}

	if err = s.wp.DeleteCategory(ctx, site, category.WPCategoryID); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete category in WordPress")
		return err
	}

	if err = s.repo.Delete(ctx, categoryID); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete category locally")
		return err
	}

	s.logger.Info("Category deleted from WordPress successfully")
	return nil
}

func (s *service) IncrementUsage(ctx context.Context, siteID, categoryID int64, date time.Time, articlesPublished, totalWords int) error {
	return s.statsRepo.Increment(ctx, siteID, categoryID, date, articlesPublished, totalWords)
}

func (s *service) GetStatistics(ctx context.Context, categoryID int64, from, to time.Time) ([]*entities.Statistics, error) {
	if from.After(to) {
		return nil, errors.Validation("From date cannot be after to date")
	}

	stats, err := s.statsRepo.GetByCategory(ctx, categoryID, from, to)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get category statistics")
		return nil, err
	}

	return stats, nil
}

func (s *service) GetSiteStatistics(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.Statistics, error) {
	if from.After(to) {
		return nil, errors.Validation("From date cannot be after to date")
	}

	stats, err := s.statsRepo.GetBySite(ctx, siteID, from, to)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site statistics")
		return nil, err
	}

	return stats, nil
}

func (s *service) validateCategory(category *entities.Category) error {
	if category.SiteID <= 0 {
		return errors.Validation("Site ID is required")
	}

	if strings.TrimSpace(category.Name) == "" {
		return errors.Validation("Category name is required")
	}

	return nil
}
