package job

import (
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"context"
	"math/rand"
	"time"
)

type ScheduleType string

const (
	ScheduleManual  ScheduleType = "manual"
	ScheduleOnce    ScheduleType = "once"
	ScheduleDaily   ScheduleType = "daily"
	ScheduleWeekly  ScheduleType = "weekly"
	ScheduleMonthly ScheduleType = "monthly"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusError     Status = "error"
)

type Job struct {
	ID           int64
	Name         string
	SiteID       int64
	CategoryID   int64
	PromptID     int64
	AIProviderID int64
	AIModel      string

	RequiresValidation bool

	ScheduleType ScheduleType
	ScheduleTime *time.Time // time for daily (HH:MM:SS)
	ScheduleDay  *int       // day of week (1-7) for weekly or day of month (1-31) for monthly

	JitterEnabled bool
	JitterMinutes int

	Status    Status
	LastRunAt *time.Time
	NextRunAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ExecutionStatus string

const (
	ExecutionPending           ExecutionStatus = "pending"
	ExecutionGenerating        ExecutionStatus = "generating"
	ExecutionPendingValidation ExecutionStatus = "pending_validation"
	ExecutionValidated         ExecutionStatus = "validated"
	ExecutionPublishing        ExecutionStatus = "publishing"
	ExecutionPublished         ExecutionStatus = "published"
	ExecutionFailed            ExecutionStatus = "failed"
)

type Execution struct {
	ID      int64
	JobID   int64
	TopicID int64

	GeneratedTitle   *string
	GeneratedContent *string

	Status       ExecutionStatus
	ErrorMessage *string

	ArticleID *int64

	StartedAt   time.Time
	GeneratedAt *time.Time
	ValidatedAt *time.Time
	PublishedAt *time.Time
}

var _ IScheduler = (*Scheduler)(nil)

type Scheduler struct {
	jobRepo  IRepository
	executor IExecutor
	logger   *logger.Logger
	stopChan chan struct{}
	running  bool
}

func NewScheduler(c di.Container) (*Scheduler, error) {
	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	jobRepo, err := NewJobRepository(c)
	if err != nil {
		return nil, err
	}

	executor, err := NewExecutor(c)
	if err != nil {
		return nil, err
	}

	return &Scheduler{
		jobRepo:  jobRepo,
		executor: executor,
		logger:   l,
		stopChan: make(chan struct{}),
	}, nil
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting job scheduler")
	s.running = true

	// Restore state on startup
	if err := s.RestoreState(ctx); err != nil {
		s.logger.Errorf("Failed to restore scheduler state: %v", err)
		return err
	}

	// Start scheduler loop
	go s.run(ctx)

	return nil
}

func (s *Scheduler) Stop() error {
	s.logger.Info("Stopping job scheduler")
	s.running = false
	close(s.stopChan)
	return nil
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("IScheduler stopped")
			return
		case <-ticker.C:
			s.checkAndExecuteDueJobs(ctx)
		}
	}
}

func (s *Scheduler) checkAndExecuteDueJobs(ctx context.Context) {
	now := time.Now()
	dueJobs, err := s.jobRepo.GetDueJobs(ctx, now)
	if err != nil {
		s.logger.Errorf("Failed to get due jobs: %v", err)
		return
	}

	if len(dueJobs) == 0 {
		return
	}

	s.logger.Infof("Found %d due jobs to execute", len(dueJobs))

	for _, job := range dueJobs {
		// Execute job in separate goroutine to avoid blocking
		go func(j *Job) {
			if err = s.executeAndReschedule(ctx, j); err != nil {
				s.logger.Errorf("Failed to execute job %d (%s): %v", j.ID, j.Name, err)
			}
		}(job)
	}
}

func (s *Scheduler) executeAndReschedule(ctx context.Context, job *Job) error {
	s.logger.Infof("Executing job %d (%s)", job.ID, job.Name)

	// Execute the job
	if err := s.executor.Execute(ctx, job); err != nil {
		s.logger.Errorf("Job %d execution failed: %v", job.ID, err)
		// Don't return error - still need to reschedule
	}

	// Update last run time
	now := time.Now()
	job.LastRunAt = &now

	// Calculate and set next run time
	if job.ScheduleType != ScheduleOnce && job.ScheduleType != ScheduleManual {
		nextRun := s.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	} else if job.ScheduleType == ScheduleOnce {
		// Mark as completed after one-time execution
		job.Status = StatusCompleted
	}

	// Update job in database
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) RestoreState(ctx context.Context) error {
	s.logger.Info("Restoring scheduler state")

	activeJobs, err := s.jobRepo.GetActive(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	missedJobsCount := 0

	for _, job := range activeJobs {
		// If job has no next run time or it's in the past, recalculate
		if job.NextRunAt == nil || job.NextRunAt.Before(now) {
			if job.ScheduleType == ScheduleManual {
				continue // Manual jobs don't need rescheduling
			}

			// Check if this is a missed execution
			if job.NextRunAt != nil && job.NextRunAt.Before(now) {
				missedJobsCount++
				// Add random delay to spread out missed executions
				delay := time.Duration(rand.Intn(300)) * time.Second
				nextRun := now.Add(delay)
				job.NextRunAt = &nextRun

				s.logger.Warnf("Job %d (%s) missed execution, rescheduling with %v delay",
					job.ID, job.Name, delay)
			} else {
				// Calculate next run normally
				nextRun := s.CalculateNextRun(job, now)
				job.NextRunAt = &nextRun
			}

			if err := s.jobRepo.Update(ctx, job); err != nil {
				s.logger.Errorf("Failed to update job %d during state restore: %v", job.ID, err)
				continue
			}
		}
	}

	s.logger.Infof("State restored: %d active jobs, %d missed executions rescheduled",
		len(activeJobs), missedJobsCount)

	return nil
}

func (s *Scheduler) CalculateNextRun(job *Job, now time.Time) time.Time {
	var nextRun time.Time

	switch job.ScheduleType {
	case ScheduleManual:
		// Manual jobs have no automatic next run
		return time.Time{}

	case ScheduleOnce:
		// One-time jobs: run immediately if not yet executed
		if job.LastRunAt == nil {
			nextRun = now
		} else {
			return time.Time{}
		}

	case ScheduleDaily:
		nextRun = s.calculateDailyNextRun(job, now)

	case ScheduleWeekly:
		nextRun = s.calculateWeeklyNextRun(job, now)

	case ScheduleMonthly:
		nextRun = s.calculateMonthlyNextRun(job, now)

	default:
		s.logger.Errorf("Unknown schedule type: %s", job.ScheduleType)
		return now.Add(24 * time.Hour) // Default to 24 hours
	}

	// Apply jitter if enabled
	if job.JitterEnabled && job.JitterMinutes > 0 {
		jitter := rand.Intn(job.JitterMinutes*2+1) - job.JitterMinutes
		nextRun = nextRun.Add(time.Duration(jitter) * time.Minute)
	}

	return nextRun
}

func (s *Scheduler) calculateDailyNextRun(job *Job, now time.Time) time.Time {
	var targetTime time.Time

	if job.ScheduleTime != nil {
		// Use specified time
		hour, min, sec := job.ScheduleTime.Clock()
		targetTime = time.Date(now.Year(), now.Month(), now.Day(), hour, min, sec, 0, now.Location())
	} else {
		// Default to current time
		targetTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	}

	// If target time today has passed, schedule for tomorrow
	if targetTime.Before(now) || targetTime.Equal(now) {
		targetTime = targetTime.Add(24 * time.Hour)
	}

	return targetTime
}

func (s *Scheduler) calculateWeeklyNextRun(job *Job, now time.Time) time.Time {
	targetWeekday := time.Monday // Default to Monday
	if job.ScheduleDay != nil && *job.ScheduleDay >= 1 && *job.ScheduleDay <= 7 {
		targetWeekday = time.Weekday(*job.ScheduleDay % 7)
	}

	var targetTime time.Time
	if job.ScheduleTime != nil {
		hour, min, sec := job.ScheduleTime.Clock()
		targetTime = time.Date(now.Year(), now.Month(), now.Day(), hour, min, sec, 0, now.Location())
	} else {
		targetTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	}

	// Find next occurrence of target weekday
	daysUntilTarget := int(targetWeekday - now.Weekday())
	if daysUntilTarget <= 0 {
		daysUntilTarget += 7
	}

	// If it's today but time has passed, move to next week
	if daysUntilTarget == 0 && targetTime.Before(now) {
		daysUntilTarget = 7
	}

	targetTime = targetTime.Add(time.Duration(daysUntilTarget) * 24 * time.Hour)

	return targetTime
}

func (s *Scheduler) calculateMonthlyNextRun(job *Job, now time.Time) time.Time {
	targetDay := 1 // Default to 1st of month
	if job.ScheduleDay != nil && *job.ScheduleDay >= 1 && *job.ScheduleDay <= 31 {
		targetDay = *job.ScheduleDay
	}

	var targetTime time.Time
	if job.ScheduleTime != nil {
		hour, min, sec := job.ScheduleTime.Clock()
		targetTime = time.Date(now.Year(), now.Month(), targetDay, hour, min, sec, 0, now.Location())
	} else {
		targetTime = time.Date(now.Year(), now.Month(), targetDay, now.Hour(), now.Minute(), 0, 0, now.Location())
	}

	// If target day this month has passed, move to next month
	if targetTime.Before(now) || targetTime.Equal(now) {
		targetTime = targetTime.AddDate(0, 1, 0)
	}

	// Handle months with fewer days (e.g., February 31 -> February 28/29)
	for targetTime.Day() != targetDay && targetDay > 28 {
		targetTime = targetTime.AddDate(0, 0, -1)
	}

	return targetTime
}
