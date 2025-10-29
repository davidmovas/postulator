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
	ScheduleManual   ScheduleType = "manual"
	ScheduleOnce     ScheduleType = "once"
	ScheduleInterval ScheduleType = "interval"
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

type Job struct {
	ID           int64
	Name         string
	SiteID       int64
	CategoryID   int64
	PromptID     int64
	AIProviderID int64
	AIModel      string

	RequiresValidation bool

	ScheduleType   ScheduleType
	IntervalValue  *int    // e.g., 3 (days), 2 (weeks), 1 (months)
	IntervalUnit   *string // days, weeks, months
	ScheduleHour   *int    // 0-23
	ScheduleMinute *int    // 0-59
	Weekdays       []int   // 1-7 (Mon=1..Sun=7)
	Monthdays      []int   // 1-31

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

	if err := s.RestoreState(ctx); err != nil {
		s.logger.Errorf("Failed to restore scheduler state: %v", err)
		return err
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
		go func(j *Job) {
			if err = s.executeAndReschedule(ctx, j); err != nil {
				s.logger.Errorf("Failed to execute job %d (%s): %v", j.ID, j.Name, err)
			}
		}(job)
	}
}

func (s *Scheduler) executeAndReschedule(ctx context.Context, job *Job) error {
	s.logger.Infof("Executing job %d (%s)", job.ID, job.Name)

	if err := s.executor.Execute(ctx, job); err != nil {
		s.logger.Errorf("Job %d execution failed: %v", job.ID, err)
	}

	now := time.Now()
	job.LastRunAt = &now

	if job.ScheduleType != ScheduleOnce && job.ScheduleType != ScheduleManual {
		nextRun := s.CalculateNextRun(job, now)
		job.NextRunAt = &nextRun
	} else if job.ScheduleType == ScheduleOnce {
		job.Status = StatusCompleted
	}

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
		if job.NextRunAt == nil || job.NextRunAt.Before(now) {
			if job.ScheduleType == ScheduleManual {
				continue
			}

			if job.NextRunAt != nil && job.NextRunAt.Before(now) {
				missedJobsCount++
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
	var nextRun time.Time

	switch job.ScheduleType {
	case ScheduleManual:
		return time.Time{}
	case ScheduleOnce:
		nextRun = s.calculateOnceNextRun(job, now)
	case ScheduleInterval:
		nextRun = s.calculateIntervalNextRun(job, now)
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
	hour := 0
	minute := 0
	if job.ScheduleHour != nil {
		hour = *job.ScheduleHour
	}
	if job.ScheduleMinute != nil {
		minute = *job.ScheduleMinute
	}
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !target.After(now) {
		target = target.Add(24 * time.Hour)
	}
	if job.LastRunAt != nil {
		return time.Time{}
	}
	return target
}

func (s *Scheduler) calculateIntervalNextRun(job *Job, now time.Time) time.Time {
	// Defaults
	interval := 1
	unit := "days"
	if job.IntervalValue != nil && *job.IntervalValue > 0 {
		interval = *job.IntervalValue
	}
	if job.IntervalUnit != nil && *job.IntervalUnit != "" {
		unit = *job.IntervalUnit
	}
	hour := 0
	minute := 0
	if job.ScheduleHour != nil {
		hour = *job.ScheduleHour
	}
	if job.ScheduleMinute != nil {
		minute = *job.ScheduleMinute
	}

	start := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if job.LastRunAt != nil && job.LastRunAt.After(start) {
		start = time.Date(job.LastRunAt.Year(), job.LastRunAt.Month(), job.LastRunAt.Day(), hour, minute, 0, 0, now.Location())
	}

	switch unit {
	case "days":
		if !start.After(now) {
			daysSince := int(now.Sub(start).Hours() / 24)
			nextDays := (daysSince/interval + 1) * interval
			return start.AddDate(0, 0, nextDays)
		}
		return start
	case "weeks":
		allowed := map[time.Weekday]bool{}
		if len(job.Weekdays) > 0 {
			for _, d := range job.Weekdays {
				if d >= 1 && d <= 7 {
					allowed[time.Weekday(d%7)] = true
				}
			}
		} else {
			for d := time.Weekday(0); d < 7; d++ {
				allowed[d] = true
			}
		}
		weekday := int(now.Weekday())
		mondayOffset := (weekday + 6) % 7
		weekStart := time.Date(now.Year(), now.Month(), now.Day()-mondayOffset, hour, minute, 0, 0, now.Location())

		if start.Before(now) {
			start = now
		}

		maxDays := interval*7 + 14
		candidate := time.Date(start.Year(), start.Month(), start.Day(), hour, minute, 0, 0, start.Location())
		for i := 0; i <= maxDays; i++ {
			wd := candidate.Weekday()
			// Check interval weeks cadence relative to weekStart
			weeksDiff := int(candidate.Sub(weekStart).Hours()) / (24 * 7)
			if weeksDiff%interval == 0 && allowed[wd] && candidate.After(now) {
				return candidate
			}
			candidate = candidate.Add(24 * time.Hour)
		}
		return now.Add(7 * 24 * time.Hour)
	case "months":
		days := job.Monthdays
		if len(days) == 0 {
			days = []int{1}
		}
		// sort days ascending
		for i := 0; i < len(days)-1; i++ {
			for j := i + 1; j < len(days); j++ {
				if days[j] < days[i] {
					days[i], days[j] = days[j], days[i]
				}
			}
		}
		// Start from current month, iterate months by interval to find next valid date
		baseMonth := time.Date(now.Year(), now.Month(), 1, hour, minute, 0, 0, now.Location())
		if job.LastRunAt != nil && job.LastRunAt.After(baseMonth) {
			baseMonth = time.Date(job.LastRunAt.Year(), job.LastRunAt.Month(), 1, hour, minute, 0, 0, now.Location())
		}
		for mOffset := 0; mOffset <= 24; mOffset += interval {
			monthBase := baseMonth.AddDate(0, mOffset, 0)
			for _, d := range days {
				// clamp day to last day of month
				lastDay := lastDayOfMonth(monthBase)
				day := d
				if day > lastDay {
					day = lastDay
				}
				cand := time.Date(monthBase.Year(), monthBase.Month(), day, hour, minute, 0, 0, monthBase.Location())
				if cand.After(now) {
					return cand
				}
			}
		}
		return now.AddDate(0, interval, 0)
	default:
		return now.Add(24 * time.Hour)
	}
}

func lastDayOfMonth(t time.Time) int {
	firstNext := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	last := firstNext.AddDate(0, 0, -1)
	return last.Day()
}
