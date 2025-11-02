package importer

import (
	"context"
	"fmt"

	"github.com/davidmovas/postulator/internal/domain/categories"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/domain/topics"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	topicService      topics.Service
	siteService       sites.Service
	categoriesService categories.Service
	logger            *logger.Logger
}

func NewImportService(
	topicService topics.Service,
	siteService sites.Service,
	categoriesService categories.Service,
	logger *logger.Logger,
) (Service, error) {
	return &service{
		topicService:      topicService,
		siteService:       siteService,
		categoriesService: categoriesService,
		logger:            logger.WithScope("importer"),
	}, nil
}

func (s *service) ImportTopics(ctx context.Context, filePath string) (*ImportResult, error) {
	s.logger.Infof("Starting topic import from file: %s", filePath)

	parser, err := GetParser(filePath)
	if err != nil {
		return nil, err
	}

	titles, err := parser.Parse(filePath)
	if err != nil {
		s.logger.Errorf("Failed to parse file %s: %v", filePath, err)
		return nil, err
	}

	tops := make([]*entities.Topic, 0, len(titles))
	for _, title := range titles {
		tops = append(tops, &entities.Topic{Title: title})
	}

	batchResult, err := s.topicService.CreateTopics(ctx, tops...)
	if err != nil {
		s.logger.Errorf("Failed to create topics batch: %v", err)
		return nil, err
	}

	result := &ImportResult{
		TotalRead:    len(titles),
		TotalAdded:   batchResult.Created,
		TotalSkipped: batchResult.Skipped,
	}

	s.logger.Infof("Import completed: %d read, %d added, %d skipped", result.TotalRead, result.TotalAdded, result.TotalSkipped)

	return result, nil
}

func (s *service) ImportAndAssignToSite(ctx context.Context, filePath string, siteID int64) (*ImportResult, error) {
	s.logger.Infof("Starting topic import and assignment to site %d from file: %s", siteID, filePath)

	parser, err := GetParser(filePath)
	if err != nil {
		return nil, err
	}

	titles, err := parser.Parse(filePath)
	if err != nil {
		s.logger.Errorf("Failed to parse file %s: %v", filePath, err)
		return nil, err
	}

	tops := make([]*entities.Topic, 0, len(titles))
	for _, title := range titles {
		tops = append(tops, &entities.Topic{Title: title})
	}

	batchResult, err := s.topicService.CreateTopics(ctx, tops...)
	if err != nil {
		s.logger.Errorf("Failed to create topics batch: %v", err)
		return nil, err
	}

	createdTopicIDs := make([]int64, 0, len(batchResult.CreatedTopics))
	for _, topic := range batchResult.CreatedTopics {
		createdTopicIDs = append(createdTopicIDs, topic.ID)
	}

	var errs []string
	if len(createdTopicIDs) > 0 {
		err = s.topicService.AssignToSite(ctx, siteID, createdTopicIDs...)
		if err != nil {
			s.logger.Errorf("Failed to assign topics to site %d: %v", siteID, err)
		}
	}

	result := &ImportResult{
		TotalRead:    len(titles),
		TotalAdded:   batchResult.Created,
		TotalSkipped: batchResult.Skipped,
		Added:        make([]string, 0, len(batchResult.CreatedTopics)),
		Skipped:      batchResult.SkippedTitles,
		Errors:       errs,
	}

	for _, topic := range batchResult.CreatedTopics {
		result.Added = append(result.Added, topic.Title)
	}

	if len(createdTopicIDs) > 0 && err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to assign %d topics to site: %v", len(createdTopicIDs), err))
	}

	s.logger.Infof("Import completed: %d read, %d added, %d skipped", result.TotalRead, result.TotalAdded, result.TotalSkipped)
	s.logger.Infof("Assignment completed: %d topics assigned to site %d", len(createdTopicIDs), siteID)

	return result, nil
}
