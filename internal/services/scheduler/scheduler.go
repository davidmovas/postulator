package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"Postulator/internal/dto"
	"Postulator/internal/models"
	"Postulator/internal/repository"
	"Postulator/internal/services/pipeline"
	"github.com/go-co-op/gocron"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Service manages scheduled tasks for automated posting
type Service struct {
	scheduler   *gocron.Scheduler
	repos       *repository.Repository
	pipeline    *pipeline.Service
	jobRegistry map[int64]*gocron.Job // Maps schedule ID to job
	mutex       sync.RWMutex
	appContext  context.Context
	isRunning   bool
}

// Config holds scheduler service configuration
type Config struct {
	Location   *time.Location
	MaxJobs    int
	JobTimeout time.Duration
	RetryCount int
	RetryDelay time.Duration
}

// JobCallback represents a function called when a scheduled job executes
type JobCallback func(ctx context.Context, schedule *models.Schedule) error

// NewService creates a new scheduler service
func NewService(config Config, repos *repository.Repository, pipeline *pipeline.Service, appContext context.Context) *Service {
	if config.Location == nil {
		config.Location = time.UTC
	}
	if config.MaxJobs == 0 {
		config.MaxJobs = 100
	}
	if config.JobTimeout == 0 {
		config.JobTimeout = 10 * time.Minute
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Minute
	}

	scheduler := gocron.NewScheduler(config.Location)
	scheduler.SetMaxConcurrentJobs(config.MaxJobs, gocron.RescheduleMode)

	service := &Service{
		scheduler:   scheduler,
		repos:       repos,
		pipeline:    pipeline,
		jobRegistry: make(map[int64]*gocron.Job),
		appContext:  appContext,
		isRunning:   false,
	}

	// Start background job worker
	go service.startJobWorker()

	return service
}

// Start starts the scheduler
func (s *Service) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	// Load existing schedules from database
	if err := s.loadSchedules(); err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	s.scheduler.StartAsync()
	s.isRunning = true

	log.Println("Scheduler started successfully")
	s.emitEvent("scheduler:started", nil)

	return nil
}

// Stop stops the scheduler
func (s *Service) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	s.scheduler.Stop()
	s.isRunning = false

	log.Println("Scheduler stopped")
	s.emitEvent("scheduler:stopped", nil)

	return nil
}

// IsRunning returns whether the scheduler is running
func (s *Service) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isRunning
}

// AddSchedule adds a new schedule to the scheduler
func (s *Service) AddSchedule(schedule *models.Schedule, callback JobCallback) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Remove existing job if any
	if existingJob, exists := s.jobRegistry[schedule.ID]; exists {
		s.scheduler.RemoveByReference(existingJob)
		delete(s.jobRegistry, schedule.ID)
	}

	// Create new job
	job, err := s.scheduler.Cron(schedule.CronExpr).Do(s.executeScheduledJob, schedule.ID, callback)
	if err != nil {
		return fmt.Errorf("failed to create scheduled job: %w", err)
	}

	// Store job reference
	s.jobRegistry[schedule.ID] = job

	log.Printf("Added schedule %d with cron expression: %s", schedule.ID, schedule.CronExpr)
	s.emitEvent("schedule:added", map[string]interface{}{
		"schedule_id": schedule.ID,
		"cron_expr":   schedule.CronExpr,
	})

	return nil
}

// RemoveSchedule removes a schedule from the scheduler
func (s *Service) RemoveSchedule(scheduleID int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job, exists := s.jobRegistry[scheduleID]
	if !exists {
		return fmt.Errorf("schedule %d not found", scheduleID)
	}

	s.scheduler.RemoveByReference(job)
	delete(s.jobRegistry, scheduleID)

	log.Printf("Removed schedule %d", scheduleID)
	s.emitEvent("schedule:removed", map[string]interface{}{
		"schedule_id": scheduleID,
	})

	return nil
}

// UpdateSchedule updates an existing schedule
func (s *Service) UpdateSchedule(schedule *models.Schedule, callback JobCallback) error {
	// Remove and add again with new configuration
	if err := s.RemoveSchedule(schedule.ID); err != nil {
		// If removal fails, it might not exist, continue with adding
		log.Printf("Warning: failed to remove schedule %d for update: %v", schedule.ID, err)
	}

	return s.AddSchedule(schedule, callback)
}

// GetActiveJobs returns information about active jobs
func (s *Service) GetActiveJobs() map[int64]JobInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	jobs := make(map[int64]JobInfo)
	for scheduleID, job := range s.jobRegistry {
		jobs[scheduleID] = JobInfo{
			ScheduleID: scheduleID,
			NextRun:    job.NextRun(),
			LastRun:    job.LastRun(),
			RunCount:   uint64(job.RunCount()),
		}
	}

	return jobs
}

// JobInfo contains information about a scheduled job
type JobInfo struct {
	ScheduleID int64     `json:"schedule_id"`
	NextRun    time.Time `json:"next_run"`
	LastRun    time.Time `json:"last_run"`
	RunCount   uint64    `json:"run_count"`
}

// loadSchedules loads all active schedules from the database
func (s *Service) loadSchedules() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	schedules, err := s.repos.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active schedules: %w", err)
	}

	for _, schedule := range schedules {
		// Create default callback that triggers article generation and posting
		callback := s.createDefaultCallback()

		if err := s.AddSchedule(schedule, callback); err != nil {
			log.Printf("Failed to add schedule %d: %v", schedule.ID, err)
			continue
		}
	}

	log.Printf("Loaded %d active schedules", len(schedules))
	return nil
}

// createDefaultCallback creates the default callback for scheduled jobs
func (s *Service) createDefaultCallback() JobCallback {
	return func(ctx context.Context, schedule *models.Schedule) error {
		log.Printf("Executing scheduled job for site %d", schedule.SiteID)

		// Emit event to notify about job execution
		s.emitEvent("job:started", map[string]interface{}{
			"schedule_id": schedule.ID,
			"site_id":     schedule.SiteID,
		})

		// Update last run time
		if err := s.repos.UpdateLastRun(ctx, schedule.ID); err != nil {
			log.Printf("Failed to update last run time for schedule %d: %v", schedule.ID, err)
		}

		// Calculate next run time
		nextRun := s.calculateNextRun(schedule.CronExpr)
		if nextRun > 0 {
			if err := s.repos.UpdateNextRun(ctx, schedule.ID, nextRun); err != nil {
				log.Printf("Failed to update next run time for schedule %d: %v", schedule.ID, err)
			}
		}

		// Create publishing job using pipeline service
		publishReq := dto.GeneratePublishRequest{
			SiteID: schedule.SiteID,
		}

		_, err := s.pipeline.CreatePublishJob(ctx, publishReq)
		if err != nil {
			log.Printf("Failed to create publish job for schedule %d: %v", schedule.ID, err)
			s.emitEvent("job:failed", map[string]interface{}{
				"schedule_id": schedule.ID,
				"site_id":     schedule.SiteID,
				"error":       err.Error(),
			})
			return err
		}

		s.emitEvent("job:trigger_posting", map[string]interface{}{
			"schedule_id": schedule.ID,
			"site_id":     schedule.SiteID,
		})

		s.emitEvent("job:completed", map[string]interface{}{
			"schedule_id": schedule.ID,
			"site_id":     schedule.SiteID,
		})

		return nil
	}
}

// executeScheduledJob is the wrapper function that executes scheduled jobs
func (s *Service) executeScheduledJob(scheduleID int64, callback JobCallback) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Get schedule from database
	schedule, err := s.repos.GetByID(ctx, scheduleID)
	if err != nil {
		log.Printf("Failed to get schedule %d: %v", scheduleID, err)
		return
	}

	if schedule == nil {
		log.Printf("Schedule %d not found", scheduleID)
		return
	}

	// Check if schedule is still active
	if !schedule.IsActive {
		log.Printf("Schedule %d is not active, removing from scheduler", scheduleID)
		s.RemoveSchedule(scheduleID)
		return
	}

	// Execute callback
	if err := callback(ctx, schedule); err != nil {
		log.Printf("Failed to execute scheduled job for schedule %d: %v", scheduleID, err)
		s.emitEvent("job:failed", map[string]interface{}{
			"schedule_id": scheduleID,
			"error":       err.Error(),
		})
	}
}

// calculateNextRun calculates the next run time for a cron expression
func (s *Service) calculateNextRun(cronExpr string) int64 {
	// Use gocron's scheduler to parse and calculate next run
	tempScheduler := gocron.NewScheduler(time.UTC)
	job, err := tempScheduler.Cron(cronExpr).Do(func() {})
	if err != nil {
		return 0
	}

	return job.NextRun().Unix()
}

// ValidateCronExpression validates a cron expression
func (s *Service) ValidateCronExpression(cronExpr string) error {
	tempScheduler := gocron.NewScheduler(time.UTC)
	_, err := tempScheduler.Cron(cronExpr).Do(func() {})
	return err
}

// GetNextRunTime returns the next run time for a cron expression
func (s *Service) GetNextRunTime(cronExpr string) (time.Time, error) {
	tempScheduler := gocron.NewScheduler(time.UTC)
	job, err := tempScheduler.Cron(cronExpr).Do(func() {})
	if err != nil {
		return time.Time{}, err
	}

	return job.NextRun(), nil
}

// RefreshSchedules reloads all schedules from the database
func (s *Service) RefreshSchedules() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Clear existing jobs
	s.scheduler.Clear()
	s.jobRegistry = make(map[int64]*gocron.Job)

	// Reload schedules
	if err := s.loadSchedules(); err != nil {
		return fmt.Errorf("failed to refresh schedules: %w", err)
	}

	log.Println("Schedules refreshed successfully")
	s.emitEvent("schedules:refreshed", nil)

	return nil
}

// GetScheduleInfo returns detailed information about a specific schedule
func (s *Service) GetScheduleInfo(scheduleID int64) (*JobInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	job, exists := s.jobRegistry[scheduleID]
	if !exists {
		return nil, fmt.Errorf("schedule %d not found", scheduleID)
	}

	return &JobInfo{
		ScheduleID: scheduleID,
		NextRun:    job.NextRun(),
		LastRun:    job.LastRun(),
		RunCount:   uint64(job.RunCount()),
	}, nil
}

// RunScheduleNow manually triggers a schedule to run immediately
func (s *Service) RunScheduleNow(scheduleID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	schedule, err := s.repos.GetByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to get schedule: %w", err)
	}

	if schedule == nil {
		return fmt.Errorf("schedule %d not found", scheduleID)
	}

	callback := s.createDefaultCallback()
	go func() {
		if err := callback(ctx, schedule); err != nil {
			log.Printf("Failed to execute manual job for schedule %d: %v", scheduleID, err)
		}
	}()

	log.Printf("Manually triggered schedule %d", scheduleID)
	s.emitEvent("schedule:manual_run", map[string]interface{}{
		"schedule_id": scheduleID,
	})

	return nil
}

// emitEvent emits an event to the frontend
// startJobWorker processes pending PostingJobs in background
func (s *Service) startJobWorker() {
	ticker := time.NewTicker(30 * time.Second) // Check for jobs every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processPendingJobs()
		case <-s.appContext.Done():
			log.Println("Job worker shutting down...")
			return
		}
	}
}

// processPendingJobs processes all pending jobs in the queue
func (s *Service) processPendingJobs() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get pending jobs from database
	pendingJobs, err := s.repos.GetJobsByStatus(ctx, "pending", 10, 0)
	if err != nil {
		log.Printf("Failed to get pending jobs: %v", err)
		return
	}

	for _, job := range pendingJobs.Data {
		// Process job in separate goroutine to avoid blocking
		go s.processJob(job)
	}
}

// processJob processes a single PostingJob
func (s *Service) processJob(job *models.PostingJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Update job status to running
	job.Status = "running"
	job.Progress = 10
	job.StartedAt = time.Now()
	_, err := s.repos.UpdateJob(ctx, job)
	if err != nil {
		log.Printf("Failed to update job %d status to running: %v", job.ID, err)
		return
	}

	// Create GeneratePublishRequest from job
	publishReq := dto.GeneratePublishRequest{
		SiteID: job.SiteID,
	}

	// Update progress
	job.Progress = 50
	s.repos.UpdateJob(ctx, job)

	// Call pipeline to generate and publish article
	article, err := s.pipeline.GenerateAndPublish(ctx, publishReq)
	if err != nil {
		// Mark job as failed
		job.Status = "failed"
		job.Progress = 0
		job.ErrorMsg = err.Error()
		job.CompletedAt = time.Now()
		s.repos.UpdateJob(ctx, job)
		log.Printf("Job %d failed: %v", job.ID, err)
		return
	}

	// Update job with article ID and mark as completed
	job.Status = "completed"
	job.Progress = 100
	job.ArticleID = article.ID
	job.CompletedAt = time.Now()
	job.ErrorMsg = ""

	_, err = s.repos.UpdateJob(ctx, job)
	if err != nil {
		log.Printf("Failed to update job %d completion status: %v", job.ID, err)
		return
	}

	log.Printf("Job %d completed successfully, created article %d", job.ID, article.ID)
}

func (s *Service) emitEvent(eventName string, data interface{}) {
	if s.appContext != nil {
		runtime.EventsEmit(s.appContext, eventName, data)
	}
}

// GetStats returns scheduler statistics
func (s *Service) GetStats() SchedulerStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := SchedulerStats{
		IsRunning:   s.isRunning,
		TotalJobs:   len(s.jobRegistry),
		ActiveJobs:  0,
		NextRunTime: time.Time{},
	}

	var nextRun time.Time
	for _, job := range s.jobRegistry {
		if nextRun.IsZero() || (!job.NextRun().IsZero() && job.NextRun().Before(nextRun)) {
			nextRun = job.NextRun()
		}
		if !job.NextRun().IsZero() {
			stats.ActiveJobs++
		}
	}

	stats.NextRunTime = nextRun
	return stats
}

// SchedulerStats contains scheduler statistics
type SchedulerStats struct {
	IsRunning   bool      `json:"is_running"`
	TotalJobs   int       `json:"total_jobs"`
	ActiveJobs  int       `json:"active_jobs"`
	NextRunTime time.Time `json:"next_run_time"`
}
