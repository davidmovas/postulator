package importer

import (
	"context"

	"Postulator/internal/domain/entities"
	"Postulator/internal/domain/site"
	"Postulator/internal/domain/topic"
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
)

var _ IImportService = (*ImportService)(nil)

type ImportService struct {
	topicService topic.IService
	siteService  site.IService
	logger       *logger.Logger
}

func NewImportService(c di.Container) (*ImportService, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	topicService, err := topic.NewService(c)
	if err != nil {
		return nil, err
	}

	siteService, err := site.NewService(c)
	if err != nil {
		return nil, err
	}

	return &ImportService{
		topicService: topicService,
		siteService:  siteService,
		logger:       l,
	}, nil
}

func (s *ImportService) ImportTopics(ctx context.Context, filePath string) (*ImportResult, error) {
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

	s.logger.Debugf("Parsed %d titles from file", len(titles))

	topics := make([]*entities.Topic, 0, len(titles))
	for _, title := range titles {
		topics = append(topics, &entities.Topic{Title: title})
	}

	batchResult, err := s.topicService.CreateTopicBatch(ctx, topics)
	if err != nil {
		s.logger.Errorf("Failed to create topics batch: %v", err)
		return nil, err
	}

	result := &ImportResult{
		TotalRead:    len(titles),
		TotalAdded:   batchResult.TotalAdded,
		TotalSkipped: batchResult.TotalSkipped,
		Added:        batchResult.Created,
		Skipped:      batchResult.Skipped,
		Errors:       make([]string, 0),
	}

	s.logger.Infof("Import completed: %d read, %d added, %d skipped", result.TotalRead, result.TotalAdded, result.TotalSkipped)

	return result, nil
}

func (s *ImportService) ImportAndAssignToSite(ctx context.Context, filePath string, siteID int64, categoryID int64, strategy entities.TopicStrategy) (*ImportResult, error) {
	s.logger.Infof("Starting topic import and assignment to site %d from file: %s", siteID, filePath)

	result, err := s.ImportTopics(ctx, filePath)
	if err != nil {
		return nil, err
	}

	if result.TotalAdded == 0 && result.TotalSkipped == 0 {
		s.logger.Infof("No topics to assign to site %d", siteID)
		return result, nil
	}

	allTitlesFromFile := append(result.Added, result.Skipped...)

	// Получаем все топики из базы для маппинга названий к ID
	allTopics, err := s.topicService.ListTopics(ctx)
	if err != nil {
		s.logger.Errorf("Failed to retrieve topics: %v", err)
		return nil, err
	}

	titleToID := make(map[string]int64)
	for _, t := range allTopics {
		titleToID[t.Title] = t.ID
	}

	if categoryID == 0 {
		var siteCategories []*entities.Category
		siteCategories, err = s.siteService.GetSiteCategories(ctx, siteID)
		if err != nil {
			s.logger.Errorf("Failed to get categories for site %d: %v", siteID, err)
			return nil, err
		}

		if len(siteCategories) == 0 {
			return nil, errors.Validation("site has no categories available for topic assignment")
		}

		categoryID = siteCategories[0].ID
		s.logger.Debugf("Using default category %d for site %d", categoryID, siteID)
	}

	assignmentErrors := make([]string, 0)
	successfulAssignments := 0

	for _, title := range allTitlesFromFile {
		topicID, ok := titleToID[title]
		if !ok {
			errMsg := "topic not found in database: " + title
			s.logger.Warnf("%s", errMsg)
			assignmentErrors = append(assignmentErrors, errMsg)
			continue
		}

		err = s.topicService.AssignToSite(ctx, siteID, topicID, categoryID, strategy)
		if err != nil {
			errMsg := "failed to assign topic '" + title + "' to site: " + err.Error()
			s.logger.Warnf("%s", errMsg)
			assignmentErrors = append(assignmentErrors, errMsg)
			continue
		}

		successfulAssignments++
	}

	result.Errors = assignmentErrors

	s.logger.Infof("Assignment completed: %d topics assigned to site %d, %d errors", successfulAssignments, siteID, len(assignmentErrors))

	return result, nil
}
