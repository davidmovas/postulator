package job

import (
	"Postulator/pkg/di"
	"Postulator/pkg/errors"
	"Postulator/pkg/logger"
	"context"
	"math/rand"
	"strings"
	"time"
)

type ScheduleType string

const (
	ScheduleManual   ScheduleType = "manual"
	ScheduleOnce     ScheduleType = "once"
	ScheduleInterval ScheduleType = "interval"
	ScheduleDaily    ScheduleType = "daily"
)

const (
	SchedulerTickersInterval = 1 * time.Minute
)

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
	StatusError     Status = "error"
)

type IntervalUnit string

const (
	UnitHours  IntervalUnit = "hours"
	UnitDays   IntervalUnit = "days"
	UnitWeeks  IntervalUnit = "weeks"
	UnitMonths IntervalUnit = "months"
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

	// For ScheduleInterval
	IntervalValue *int
	IntervalUnit  *IntervalUnit

	// For ScheduleDaily
	ScheduleHour   *int
	ScheduleMinute *int
	Weekdays       []int // 1-7 (Mon=1..Sun=7)

	JitterEnabled bool
	JitterMinutes int

	Status    Status
	LastRunAt *time.Time
	NextRunAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (j *Job) Validate() error {
	switch j.ScheduleType {
	case ScheduleManual:
		return nil
	case ScheduleOnce:
		return nil
	case ScheduleInterval:
		if j.IntervalValue == nil || *j.IntervalValue <= 0 {
			return errors.Validation("IntervalValue must be positive for interval schedule")
		}
		if j.IntervalUnit == nil {
			return errors.Validation("IntervalUnit is required for interval schedule")
		}
		if *j.IntervalUnit == UnitHours && *j.IntervalValue > 24 {
			return errors.Validation("Interval cannot exceed 24 hours for hourly schedule")
		}
	case ScheduleDaily:
		if j.ScheduleHour == nil || *j.ScheduleHour < 0 || *j.ScheduleHour > 23 {
			return errors.Validation("ScheduleHour must be between 0 and 23 for daily schedule")
		}
		if j.ScheduleMinute == nil || *j.ScheduleMinute < 0 || *j.ScheduleMinute > 59 {
			return errors.Validation("ScheduleMinute must be between 0 and 59 for daily schedule")
		}
		if len(j.Weekdays) == 0 {
			return errors.Validation("At least one weekday must be specified for daily schedule")
		}
		for _, day := range j.Weekdays {
			if day < 1 || day > 7 {
				return errors.Validation("Weekdays must be between 1 and 7")
			}
		}
	default:
		return errors.Validation("Invalid schedule type")
	}

	if j.JitterEnabled && j.JitterMinutes < 0 {
		return errors.Validation("JitterMinutes cannot be negative")
	}

	return nil
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

	if err := s.RestoreState(ctx); err != nil {
		s.logger.Errorf("Failed to restore scheduler state: %v", err)
		return errors.Scheduler(err)
	}

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
	ticker := time.NewTicker(SchedulerTickersInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("Scheduler stopped")
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
		go func(j *Job) {
			if err = s.executeAndReschedule(ctx, j); err != nil {
				s.logger.Errorf("Failed to execute job %d (%s): %v", j.ID, j.Name, err)
			}
		}(job)
	}
}

func (s *Scheduler) executeAndReschedule(ctx context.Context, job *Job) error {
	s.logger.Infof("Executing job %d (%s)", job.ID, job.Name)

	err := s.executor.Execute(ctx, job)

	if IsNoTopicsError(err) {
		s.logger.Warnf("No available topics for job %d, pausing job", job.ID)

		job.Status = StatusPaused
		job.NextRunAt = nil

		if updateErr := s.jobRepo.Update(ctx, job); updateErr != nil {
			s.logger.Errorf("Failed to pause job %d: %v", job.ID, updateErr)
			return errors.Scheduler(updateErr)
		}

		s.logger.Infof("Job %d paused successfully due to no available topics", job.ID)
		return nil
	}

	if err != nil {
		s.logger.Errorf("Job %d execution failed: %v", job.ID, err)
	}

	now := time.Now()
	job.LastRunAt = &now

	if job.Status == StatusActive && job.ScheduleType != ScheduleManual {
		if job.ScheduleType == ScheduleOnce {
			job.Status = StatusCompleted
			job.NextRunAt = nil
		} else {
			nextRun := s.CalculateNextRun(job, now)
			job.NextRunAt = &nextRun
		}

		if err = s.jobRepo.Update(ctx, job); err != nil {
			return errors.Scheduler(err)
		}
	}

	if job.Status == StatusCompleted && job.ScheduleType == ScheduleOnce {
		err = s.jobRepo.Delete(ctx, job.ID)
		if err != nil {
			s.logger.Errorf("Failed to delete job %d: %v", job.ID, err)
			return errors.JobExecutionWithNote(job.ID, "Fail while deleting completed job", err)
		}
	}

	return nil
}

func (s *Scheduler) RestoreState(ctx context.Context) error {
	s.logger.Info("Restoring scheduler state")

	activeJobs, err := s.jobRepo.GetActive(ctx)
	if err != nil {
		return errors.Scheduler(err)
	}

	now := time.Now()
	missedJobsCount := 0

	for _, job := range activeJobs {
		if job.NextRunAt == nil || job.NextRunAt.Before(now) {
			if job.ScheduleType == ScheduleManual {
				continue
			}

			if job.NextRunAt != nil && job.NextRunAt.Before(now) {
				missedJobsCount++
				// Add random delay to avoid thundering herd
				delay := time.Duration(rand.Intn(300)) * time.Second
				nextRun := now.Add(delay)
				job.NextRunAt = &nextRun

				s.logger.Warnf("Job %d (%s) missed execution, rescheduling with %v delay",
					job.ID, job.Name, delay)
			} else {
				nextRun := s.CalculateNextRun(job, now)
				job.NextRunAt = &nextRun
			}

			if err = s.jobRepo.Update(ctx, job); err != nil {
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
	if job.Status != StatusActive {
		return time.Time{}
	}

	var nextRun time.Time

	switch job.ScheduleType {
	case ScheduleManual:
		return time.Time{}
	case ScheduleOnce:
		nextRun = s.calculateOnceNextRun(job, now)
	case ScheduleInterval:
		nextRun = s.calculateIntervalNextRun(job, now)
	case ScheduleDaily:
		nextRun = s.calculateDailyNextRun(job, now)
	default:
		s.logger.Errorf("Unknown schedule type: %s", job.ScheduleType)
		return now.Add(24 * time.Hour)
	}

	if job.JitterEnabled && job.JitterMinutes > 0 {
		jitter := rand.Intn(job.JitterMinutes*2+1) - job.JitterMinutes
		nextRun = nextRun.Add(time.Duration(jitter) * time.Minute)
	}

	return nextRun
}

func (s *Scheduler) calculateOnceNextRun(job *Job, now time.Time) time.Time {
	if job.ScheduleHour == nil || job.ScheduleMinute == nil {
		return time.Time{}
	}

	hour := *job.ScheduleHour
	minute := *job.ScheduleMinute

	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	if candidate.Before(now) {
		candidate = candidate.Add(24 * time.Hour)
	}

	return candidate
}

func (s *Scheduler) calculateIntervalNextRun(job *Job, now time.Time) time.Time {
	if job.IntervalValue == nil || job.IntervalUnit == nil {
		return now.Add(24 * time.Hour)
	}

	interval := *job.IntervalValue
	unit := *job.IntervalUnit

	baseTime := now
	if job.LastRunAt != nil {
		baseTime = *job.LastRunAt
	}

	var idealNextRun time.Time
	switch unit {
	case UnitHours:
		idealNextRun = baseTime.Add(time.Duration(interval) * time.Hour)
	case UnitDays:
		idealNextRun = baseTime.AddDate(0, 0, interval)
	case UnitWeeks:
		idealNextRun = baseTime.AddDate(0, 0, interval*7)
	case UnitMonths:
		idealNextRun = baseTime.AddDate(0, interval, 0)
	default:
		idealNextRun = baseTime.Add(24 * time.Hour)
	}

	return idealNextRun
}

func (s *Scheduler) calculateDailyNextRun(job *Job, now time.Time) time.Time {
	if job.ScheduleHour == nil || job.ScheduleMinute == nil || len(job.Weekdays) == 0 {
		return now.Add(24 * time.Hour)
	}

	hour := *job.ScheduleHour
	minute := *job.ScheduleMinute

	allowedDays := make(map[time.Weekday]bool)
	for _, day := range job.Weekdays {
		// Convert from 1-7 (Mon-Sun) to 0-6 (Sun-Sat)
		weekday := time.Weekday((day) % 7)
		allowedDays[weekday] = true
	}

	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	if candidate.Before(now) {
		candidate = candidate.Add(24 * time.Hour)
	}

	for i := 0; i < 8; i++ {
		if allowedDays[candidate.Weekday()] {
			return candidate
		}
		candidate = candidate.Add(24 * time.Hour)
	}

	return now.Add(24 * time.Hour)
}

func (s *Scheduler) ScheduleJob(ctx context.Context, job *Job) error {
	if err := job.Validate(); err != nil {
		return err
	}

	if job.ScheduleType != ScheduleManual {
		now := time.Now()
		nextRun := s.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	}

	job.Status = StatusActive

	if job.ID == 0 {
		return s.jobRepo.Create(ctx, job)
	}
	return s.jobRepo.Update(ctx, job)
}

func (s *Scheduler) TriggerJob(ctx context.Context, jobID int64) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Status != StatusActive {
		return errors.Validation("Job is not active")
	}

	go func() {
		if err = s.executeAndReschedule(ctx, job); err != nil {
			s.logger.Errorf("Failed to trigger job %d: %v", jobID, err)
		}
	}()

	return nil
}

func IsNoTopicsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no available topics for site")
}
