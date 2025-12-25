package stats

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*service)(nil)

type service struct {
	sitesService     sites.Service
	jobsRepo         jobs.Repository
	executionService ExecutionStatsReader
	repo             Repository
	logger           *logger.Logger
}

func NewService(
	sitesService sites.Service,
	jobsRepo jobs.Repository,
	executionService ExecutionStatsReader,
	repo Repository,
	logger *logger.Logger,
) Service {
	return &service{
		sitesService:     sitesService,
		jobsRepo:         jobsRepo,
		executionService: executionService,
		repo:             repo,
		logger: logger.
			WithScope("service").
			WithScope("stats"),
	}
}

func (s *service) GetSiteStatistics(ctx context.Context, siteID int64, from, to time.Time) ([]*entities.SiteStats, error) {
	if from.After(to) {
		return nil, errors.Validation("From date cannot be after to date")
	}

	stats, err := s.repo.GetSiteStats(ctx, siteID, from, to)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get site statistics")
		return nil, err
	}

	return stats, nil
}

func (s *service) GetTotalStatistics(ctx context.Context, siteID int64) (*entities.SiteStats, error) {
	stats, err := s.repo.GetTotalSiteStats(ctx, siteID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get total statistics")
		return nil, err
	}

	return stats, nil
}

func (s *service) GetDashboardSummary(ctx context.Context) (*entities.DashboardSummary, error) {
	summary := &entities.DashboardSummary{}

	if err := s.populateSitesInfo(ctx, summary); err != nil {
		return nil, err
	}

	if err := s.populateJobsInfo(ctx, summary); err != nil {
		return nil, err
	}

	if err := s.populateExecutionsInfo(ctx, summary); err != nil {
		return nil, err
	}

	return summary, nil
}

func (s *service) populateSitesInfo(ctx context.Context, summary *entities.DashboardSummary) error {
	allSites, err := s.sitesService.ListSites(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get sites for dashboard")
		return err
	}

	summary.TotalSites = len(allSites)
	summary.ActiveSites = s.countActiveSites(allSites)
	summary.UnhealthySites = s.countUnhealthySites(allSites)

	return nil
}

func (s *service) populateExecutionsInfo(ctx context.Context, summary *entities.DashboardSummary) error {
	pendingValidations, err := s.executionService.GetPendingValidations(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get pending validations for dashboard")
		return err
	}

	summary.PendingValidations = len(pendingValidations)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)

	allExecutions, _, err := s.executionService.ListExecutions(ctx, 0, 10000, 0)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get executions for dashboard")
		return err
	}

	summary.ExecutionsToday = s.countExecutionsToday(allExecutions, today, tomorrow)
	summary.FailedExecutionsToday = s.countFailedExecutionsToday(allExecutions, today, tomorrow)

	return nil
}

func (s *service) populateJobsInfo(ctx context.Context, summary *entities.DashboardSummary) error {
	allJobs, err := s.jobsRepo.GetAll(ctx)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get jobs for dashboard")
		return err
	}

	summary.TotalJobs = len(allJobs)
	summary.ActiveJobs = s.countActiveJobs(allJobs)
	summary.PausedJobs = s.countPausedJobs(allJobs)

	return nil
}

func (s *service) countActiveSites(sites []*entities.Site) int {
	count := 0
	for _, site := range sites {
		if site.Status == entities.StatusActive {
			count++
		}
	}
	return count
}

func (s *service) countUnhealthySites(sites []*entities.Site) int {
	count := 0
	for _, site := range sites {
		if site.HealthStatus == entities.HealthUnhealthy {
			count++
		}
	}
	return count
}

func (s *service) countActiveJobs(jobs []*entities.Job) int {
	count := 0
	for _, job := range jobs {
		if job.Status == entities.JobStatusActive {
			count++
		}
	}
	return count
}

func (s *service) countPausedJobs(jobs []*entities.Job) int {
	count := 0
	for _, job := range jobs {
		if job.Status == entities.JobStatusPaused {
			count++
		}
	}
	return count
}

func (s *service) countExecutionsToday(executions []*entities.Execution, today, tomorrow time.Time) int {
	count := 0
	for _, exec := range executions {
		if exec.StartedAt.After(today) && exec.StartedAt.Before(tomorrow) {
			count++
		}
	}
	return count
}

func (s *service) countFailedExecutionsToday(executions []*entities.Execution, today, tomorrow time.Time) int {
	count := 0
	for _, exec := range executions {
		if exec.StartedAt.After(today) && exec.StartedAt.Before(tomorrow) &&
			exec.Status == entities.ExecutionStatusFailed {
			count++
		}
	}
	return count
}

func (s *service) GetGlobalStatistics(ctx context.Context, from, to time.Time) ([]*entities.SiteStats, error) {
	if from.After(to) {
		return nil, errors.Validation("From date cannot be after to date")
	}

	stats, err := s.repo.GetGlobalStats(ctx, from, to)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get global statistics")
		return nil, err
	}

	return stats, nil
}
