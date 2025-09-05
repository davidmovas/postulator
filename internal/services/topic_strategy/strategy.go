package topic_strategy

import (
	"Postulator/internal/repository"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"Postulator/internal/models"
)

// TopicSelectorInterface defines the interface for topic selection strategies
type TopicSelectorInterface interface {
	SelectTopic(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) (*models.TopicSelectionResult, error)
	GetStrategyName() string
	CanContinue(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) bool
}

// TopicStrategyService manages topic selection strategies
type TopicStrategyService struct {
	repo       *repository.Repository
	strategies map[string]TopicSelectorInterface
}

// NewTopicStrategyService creates a new topic strategy service
func NewTopicStrategyService(repo *repository.Repository) *TopicStrategyService {
	service := &TopicStrategyService{
		repo:       repo,
		strategies: make(map[string]TopicSelectorInterface),
	}

	// Register available strategies
	service.RegisterStrategy(&UniqueStrategy{repo: repo})
	service.RegisterStrategy(&RoundRobinStrategy{repo: repo})
	service.RegisterStrategy(&RandomStrategy{repo: repo})
	service.RegisterStrategy(&RandomAllStrategy{repo: repo})

	return service
}

// RegisterStrategy registers a new topic selection strategy
func (s *TopicStrategyService) RegisterStrategy(strategy TopicSelectorInterface) {
	s.strategies[strategy.GetStrategyName()] = strategy
}

// SelectTopicForSite selects a topic for article generation based on the site's strategy
func (s *TopicStrategyService) SelectTopicForSite(ctx context.Context, request *models.TopicSelectionRequest) (*models.TopicSelectionResult, error) {
	// Get the site to retrieve its strategy
	site, err := s.repo.GetSite(ctx, request.SiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get site: %w", err)
	}

	strategyName := site.Strategy
	if strategyName == "" {
		strategyName = string(models.StrategyUnique) // Default strategy
	}

	strategy, exists := s.strategies[strategyName]
	if !exists {
		return nil, fmt.Errorf("unknown strategy: %s", strategyName)
	}

	// Get available topics for this site and strategy
	availableTopics, err := s.repo.GetSiteTopicsForSelection(ctx, request.SiteID, strategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get available topics: %w", err)
	}

	if len(availableTopics) == 0 {
		return nil, fmt.Errorf("no topics available for site %d with strategy %s", request.SiteID, strategyName)
	}

	// Use the strategy to select a topic
	result, err := strategy.SelectTopic(ctx, request.SiteID, availableTopics)
	if err != nil {
		return nil, fmt.Errorf("failed to select topic: %w", err)
	}

	return result, nil
}

// GetTopicStatsForSite returns topic statistics for a site
func (s *TopicStrategyService) GetTopicStatsForSite(ctx context.Context, siteID int64) (*models.TopicStats, error) {
	return s.repo.GetTopicStats(ctx, siteID)
}

// CanContinueWithStrategy checks if more topics are available for the given strategy
func (s *TopicStrategyService) CanContinueWithStrategy(ctx context.Context, siteID int64, strategyName string) (bool, error) {
	strategy, exists := s.strategies[strategyName]
	if !exists {
		return false, fmt.Errorf("unknown strategy: %s", strategyName)
	}

	availableTopics, err := s.repo.GetSiteTopicsForSelection(ctx, siteID, strategyName)
	if err != nil {
		return false, err
	}

	return strategy.CanContinue(ctx, siteID, availableTopics), nil
}

// UniqueStrategy selects topics that haven't been used before
type UniqueStrategy struct {
	repo *repository.Repository
}

func (s *UniqueStrategy) GetStrategyName() string {
	return string(models.StrategyUnique)
}

func (s *UniqueStrategy) SelectTopic(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) (*models.TopicSelectionResult, error) {
	// Find topics that have never been used (UsageCount == 0)
	var unusedTopics []*models.SiteTopic
	for _, siteTopic := range availableTopics {
		if siteTopic.UsageCount == 0 {
			unusedTopics = append(unusedTopics, siteTopic)
		}
	}

	if len(unusedTopics) == 0 {
		return nil, fmt.Errorf("no unused topics available for unique strategy")
	}

	// Select the first unused topic (or could be random from unused)
	selectedSiteTopic := unusedTopics[0]

	// Get the topic details
	topic, err := s.repo.GetTopic(ctx, selectedSiteTopic.TopicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic details: %w", err)
	}

	// Update usage
	if err = s.repo.UpdateSiteTopicUsage(ctx, selectedSiteTopic.ID, s.GetStrategyName()); err != nil {
		return nil, fmt.Errorf("failed to update topic usage: %w", err)
	}

	result := &models.TopicSelectionResult{
		Topic:          topic,
		SiteTopic:      selectedSiteTopic,
		Strategy:       s.GetStrategyName(),
		CanContinue:    len(unusedTopics) > 1, // More than the one we just selected
		RemainingCount: len(unusedTopics) - 1,
	}

	return result, nil
}

func (s *UniqueStrategy) CanContinue(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) bool {
	for _, siteTopic := range availableTopics {
		if siteTopic.UsageCount == 0 {
			return true
		}
	}
	return false
}

// RoundRobinStrategy cycles through topics in order
type RoundRobinStrategy struct {
	repo *repository.Repository
}

func (s *RoundRobinStrategy) GetStrategyName() string {
	return string(models.StrategyRoundRobin)
}

func (s *RoundRobinStrategy) SelectTopic(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) (*models.TopicSelectionResult, error) {
	if len(availableTopics) == 0 {
		return nil, fmt.Errorf("no topics available for round-robin strategy")
	}

	var selectedSiteTopic *models.SiteTopic

	// First, try to find topics that haven't been used in round-robin yet (position 0)
	for _, siteTopic := range availableTopics {
		if siteTopic.RoundRobinPos == 0 {
			if selectedSiteTopic == nil {
				selectedSiteTopic = siteTopic
			} else {
				// Among unused topics, prefer the one that was used longest ago (or never used)
				if siteTopic.LastUsedAt == nil {
					// If current topic has never been used, prefer it
					if selectedSiteTopic.LastUsedAt != nil {
						selectedSiteTopic = siteTopic
					}
				} else if selectedSiteTopic.LastUsedAt == nil {
					// Keep the current selection (never used is preferred)
				} else if siteTopic.LastUsedAt.Before(*selectedSiteTopic.LastUsedAt) {
					selectedSiteTopic = siteTopic
				}
			}
		}
	}

	// If all topics have been used, find the next one in round-robin order
	if selectedSiteTopic == nil {
		// Find the topic with the lowest non-zero position (next in cycle)
		minPos := int(^uint(0) >> 1) // Max int
		for _, siteTopic := range availableTopics {
			if siteTopic.RoundRobinPos > 0 && siteTopic.RoundRobinPos < minPos {
				minPos = siteTopic.RoundRobinPos
				selectedSiteTopic = siteTopic
			}
		}

		// If still no selection (shouldn't happen), fall back to first available
		if selectedSiteTopic == nil {
			selectedSiteTopic = availableTopics[0]
		}
	}

	if selectedSiteTopic == nil {
		return nil, fmt.Errorf("could not select topic for round-robin strategy")
	}

	// Get the topic details
	topic, err := s.repo.GetTopic(ctx, selectedSiteTopic.TopicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic details: %w", err)
	}

	// Update usage and round-robin position
	if err = s.repo.UpdateSiteTopicUsage(ctx, selectedSiteTopic.ID, s.GetStrategyName()); err != nil {
		return nil, fmt.Errorf("failed to update topic usage: %w", err)
	}

	result := &models.TopicSelectionResult{
		Topic:          topic,
		SiteTopic:      selectedSiteTopic,
		Strategy:       s.GetStrategyName(),
		CanContinue:    true, // Round-robin can always continue
		RemainingCount: len(availableTopics),
	}

	return result, nil
}

func (s *RoundRobinStrategy) CanContinue(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) bool {
	return len(availableTopics) > 0 // Round-robin can always continue if there are topics
}

// RandomStrategy selects a random topic from available ones
type RandomStrategy struct {
	repo *repository.Repository
}

func (s *RandomStrategy) GetStrategyName() string {
	return string(models.StrategyRandom)
}

func (s *RandomStrategy) SelectTopic(ctx context.Context, _ int64, availableTopics []*models.SiteTopic) (*models.TopicSelectionResult, error) {
	if len(availableTopics) == 0 {
		return nil, fmt.Errorf("no topics available for random strategy")
	}

	// Select a random topic
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(availableTopics))))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random number: %w", err)
	}

	selectedSiteTopic := availableTopics[randomIndex.Int64()]

	// Get the topic details
	topic, err := s.repo.GetTopic(ctx, selectedSiteTopic.TopicID)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic details: %w", err)
	}

	// Update usage
	if err = s.repo.UpdateSiteTopicUsage(ctx, selectedSiteTopic.ID, s.GetStrategyName()); err != nil {
		return nil, fmt.Errorf("failed to update topic usage: %w", err)
	}

	result := &models.TopicSelectionResult{
		Topic:          topic,
		SiteTopic:      selectedSiteTopic,
		Strategy:       s.GetStrategyName(),
		CanContinue:    true, // Random can always continue
		RemainingCount: len(availableTopics),
	}

	return result, nil
}

func (s *RandomStrategy) CanContinue(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) bool {
	return len(availableTopics) > 0 // Random can always continue if there are topics
}

// RandomAllStrategy selects a random topic from all available topics in the system (not just site-specific)
type RandomAllStrategy struct {
	repo *repository.Repository
}

func (s *RandomAllStrategy) GetStrategyName() string {
	return string(models.StrategyRandomAll)
}

func (s *RandomAllStrategy) SelectTopic(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) (*models.TopicSelectionResult, error) {
	// Get all active topics from the system, not just site-specific ones
	allActiveTopics, err := s.repo.GetActiveTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active topics: %w", err)
	}

	if len(allActiveTopics) == 0 {
		return nil, fmt.Errorf("no active topics available for random_all strategy")
	}

	// Select a random topic from all active topics
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(allActiveTopics))))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random number: %w", err)
	}

	selectedTopic := allActiveTopics[randomIndex.Int64()]

	// Check if this topic is already associated with the site, if not create a SiteTopic entry
	siteTopic, err := s.repo.GetSiteTopic(ctx, siteID, selectedTopic.ID)
	if err != nil {
		// Topic is not associated with this site, create a temporary SiteTopic for tracking
		siteTopic = &models.SiteTopic{
			SiteID:   siteID,
			TopicID:  selectedTopic.ID,
			IsActive: true,
			Priority: 1,
		}
		// We don't actually save this to the database since it's a random_all selection
		// The user might not want to permanently associate this topic with the site
	}

	// Record usage in topic_usage table without updating SiteTopic
	if err = s.repo.RecordTopicUsage(ctx, siteID, selectedTopic.ID, 0, s.GetStrategyName()); err != nil {
		// Log error but don't fail the selection
		// Article ID is 0 since we don't have it at selection time
	}

	result := &models.TopicSelectionResult{
		Topic:          selectedTopic,
		SiteTopic:      siteTopic,
		Strategy:       s.GetStrategyName(),
		CanContinue:    true, // Random_all can always continue if there are active topics
		RemainingCount: len(allActiveTopics),
	}

	return result, nil
}

func (s *RandomAllStrategy) CanContinue(ctx context.Context, siteID int64, availableTopics []*models.SiteTopic) bool {
	// For random_all, we need to check if there are any active topics in the system
	allActiveTopics, err := s.repo.GetActiveTopics(ctx)
	if err != nil {
		return false
	}
	return len(allActiveTopics) > 0
}
