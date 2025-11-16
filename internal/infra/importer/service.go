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
	createdTitles := make([]string, 0, len(batchResult.CreatedTopics))
	for _, topic := range batchResult.CreatedTopics {
		createdTopicIDs = append(createdTopicIDs, topic.ID)
		createdTitles = append(createdTitles, topic.Title)
	}

	existingTopics, err := s.topicService.GetByTitles(ctx, batchResult.SkippedTitles)
	if err != nil {
		s.logger.Errorf("Failed to get existing topics by titles: %v", err)
		return nil, err
	}
	titleToID := make(map[string]int64, len(existingTopics))
	existingIDs := make([]int64, 0, len(existingTopics))
	for _, t := range existingTopics {
		titleToID[t.Title] = t.ID
		existingIDs = append(existingIDs, t.ID)
	}

	candidateIDs := make([]int64, 0, len(createdTopicIDs)+len(existingIDs))
	candidateIDs = append(candidateIDs, createdTopicIDs...)
	candidateIDs = append(candidateIDs, existingIDs...)

	alreadyAssignedIDs, err := s.topicService.GetAssignedForSite(ctx, siteID, candidateIDs)
	if err != nil {
		s.logger.Errorf("Failed to get assigned topics for site %d: %v", siteID, err)
		return nil, err
	}
	assigned := make(map[int64]struct{}, len(alreadyAssignedIDs))
	for _, id := range alreadyAssignedIDs {
		assigned[id] = struct{}{}
	}

	toAssignSet := make(map[int64]struct{})
	for _, id := range candidateIDs {
		if _, ok := assigned[id]; !ok {
			toAssignSet[id] = struct{}{}
		}
	}

	assignedExistingTitles := make([]string, 0)
	trulySkippedTitles := make([]string, 0)
	for _, title := range batchResult.SkippedTitles {
		id, ok := titleToID[title]
		if !ok {
			trulySkippedTitles = append(trulySkippedTitles, title)
			continue
		}
		if _, already := assigned[id]; already {
			trulySkippedTitles = append(trulySkippedTitles, title)
			continue
		}
		assignedExistingTitles = append(assignedExistingTitles, title)
	}

	toAssign := make([]int64, 0, len(toAssignSet))
	for id := range toAssignSet {
		toAssign = append(toAssign, id)
	}

	var assignErr error
	if len(toAssign) > 0 {
		assignErr = s.topicService.AssignToSite(ctx, siteID, toAssign...)
		if assignErr != nil {
			s.logger.Errorf("Failed to assign topics to site %d: %v", siteID, assignErr)
		}
	}

	result := &ImportResult{
		TotalRead:    len(titles),
		TotalAdded:   batchResult.Created,
		TotalSkipped: len(trulySkippedTitles),
		Added:        make([]string, 0, len(createdTitles)+len(assignedExistingTitles)),
		Skipped:      trulySkippedTitles,
		Errors:       []string{},
	}

	result.Added = append(result.Added, createdTitles...)
	result.Added = append(result.Added, assignedExistingTitles...)

	if len(toAssign) > 0 && assignErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to assign %d topics to site: %v", len(toAssign), assignErr))
	}

	return result, nil
}
