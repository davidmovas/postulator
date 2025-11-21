package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/fault"
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipevents"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/pkg/ctx"
	"github.com/davidmovas/postulator/pkg/logger"
)

type Pipeline struct {
	commands      []Command
	eventBus      *events.EventBus
	errorHandler  fault.ErrorHandler
	logger        *logger.Logger
	retryStrategy RetryStrategy
}

type Builder struct {
	commands      []Command
	eventBus      *events.EventBus
	errorHandler  fault.ErrorHandler
	logger        *logger.Logger
	retryStrategy RetryStrategy
}

func NewPipelineBuilder() *Builder {
	return &Builder{
		commands:      make([]Command, 0),
		eventBus:      events.GetGlobalEventBus(),
		errorHandler:  &fault.DefaultErrorHandler{},
		retryStrategy: &ExponentialBackoffRetry{},
	}
}

func (b *Builder) WithEventBus(eventBus *events.EventBus) *Builder {
	b.eventBus = eventBus
	return b
}

func (b *Builder) WithErrorHandler(handler fault.ErrorHandler) *Builder {
	b.errorHandler = handler
	return b
}

func (b *Builder) WithLogger(logger *logger.Logger) *Builder {
	b.logger = logger
	return b
}

func (b *Builder) WithRetryStrategy(strategy RetryStrategy) *Builder {
	b.retryStrategy = strategy
	return b
}

func (b *Builder) AddCommand(cmd Command) *Builder {
	b.commands = append(b.commands, cmd)
	return b
}

func (b *Builder) AddCommands(cmds ...Command) *Builder {
	b.commands = append(b.commands, cmds...)
	return b
}

func (b *Builder) Build() *Pipeline {
	return &Pipeline{
		commands:      b.commands,
		eventBus:      b.eventBus,
		errorHandler:  b.errorHandler,
		logger:        b.logger,
		retryStrategy: b.retryStrategy,
	}
}

func (p *Pipeline) Execute(ctx context.Context, job *entities.Job) error {
	pctx := NewContext(job).
		WithContext(ctx).
		WithLogger(p.logger)

	p.publishEvent(events.Event{
		Type:      pipevents.EventPipelineStarted,
		Timestamp: time.Now(),
		Data: &pipevents.PipelineStartedEvent{
			JobID:   job.ID,
			JobName: job.Name,
		},
	})

	p.log("Pipeline started for job %d (%s)", job.ID, job.Name)

	for i, cmd := range p.commands {
		if ctx.Err() != nil {
			return p.handleCancellation(pctx, ctx.Err())
		}

		if !cmd.CanExecute(pctx) {
			p.log("Skipping command %s (condition not met)", cmd.Name())
			continue
		}

		if err := p.validateState(pctx, cmd); err != nil {
			return p.handleError(pctx, cmd, err)
		}

		if err := p.executeCommand(pctx, cmd, i); err != nil {
			return p.handleError(pctx, cmd, err)
		}

		if pctx.State.IsFinalState() {
			p.log("Pipeline reached final state: %s", pctx.State.CurrentState())
			break
		}
	}

	if pctx.State.CurrentState() == StateCompleted {
		p.publishCompletedEvent(pctx)
		p.log("Pipeline completed successfully for job %d", job.ID)
		return nil
	}

	if pctx.State.IsPausedState() {
		p.publishPausedEvent(pctx)
		p.log("Pipeline paused for job %d at state %s", job.ID, pctx.State.CurrentState())
		return nil
	}

	return fmt.Errorf("pipeline ended in unexpected state: %s", pctx.State.CurrentState())
}

func (p *Pipeline) executeCommand(ctx *Context, cmd Command, cmdIndex int) error {
	cmdName := cmd.Name()
	startTime := time.Now()

	p.publishEvent(events.Event{
		Type:      pipevents.EventStepStarted,
		Timestamp: startTime,
		Data: &pipevents.StepStartedEvent{
			JobID:    ctx.Job.ID,
			StepName: cmdName,
			State:    string(ctx.State.CurrentState()),
		},
	})

	p.log("Executing command %d: %s", cmdIndex+1, cmdName)

	var lastErr error
	maxRetries := 0

	if cmd.IsRetryable() {
		maxRetries = cmd.MaxRetries()
		if maxRetries == 0 {
			maxRetries = 3
		}
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			var reason string
			if lastErr != nil {
				reason = lastErr.Error()
			}

			p.publishEvent(events.Event{
				Type:      pipevents.EventStepRetrying,
				Timestamp: time.Now(),
				Data: &pipevents.StepRetryingEvent{
					JobID:      ctx.Job.ID,
					StepName:   cmdName,
					Attempt:    attempt,
					MaxRetries: maxRetries,
					Reason:     reason,
				},
			})

			p.log("Retrying command %s (attempt %d/%d)", cmdName, attempt, maxRetries)

			if err := p.retryStrategy.Wait(ctx.Context(), attempt); err != nil {
				return err
			}
		}

		err := cmd.Execute(ctx)

		if err == nil {
			duration := time.Since(startTime)

			if err = ctx.State.Transition(cmd.NextState(), fmt.Sprintf("completed %s", cmdName)); err != nil {
				return fmt.Errorf("failed to transition state: %w", err)
			}

			p.publishEvent(events.NewEvent(
				pipevents.EventStepCompleted,
				&pipevents.StepCompletedEvent{
					JobID:    ctx.Job.ID,
					StepName: cmdName,
					Duration: duration,
					State:    string(ctx.State.CurrentState()),
				},
			))

			p.log("Command %s completed in %v", cmdName, duration)
			return nil
		}

		lastErr = err

		if !cmd.IsRetryable() || attempt >= maxRetries {
			break
		}

		var pErr *fault.PipelineError
		if errors.As(err, &pErr) {
			if !pErr.IsRetryable() {
				break
			}
		}
	}

	duration := time.Since(startTime)

	if lastErr != nil {
		p.publishStepFailedEvent(ctx, cmdName, lastErr, duration)
	}

	return lastErr
}

func (p *Pipeline) validateState(ctx *Context, cmd Command) error {
	required := cmd.RequiredState()
	current := ctx.State.CurrentState()

	if required != "" && current != required {
		return fmt.Errorf("invalid state for command %s: required=%s, current=%s",
			cmd.Name(), required, current)
	}

	return nil
}

func (p *Pipeline) handleError(ctx *Context, cmd Command, err error) error {
	if recoveredErr := cmd.OnError(ctx, err); recoveredErr == nil {
		p.log("Command %s recovered from error", cmd.Name())
		return nil
	} else if !errors.Is(recoveredErr, err) {
		err = recoveredErr
	}

	var pErr *fault.PipelineError
	var e *fault.PipelineError
	if errors.As(err, &e) {
		pErr = e
	} else {
		pErr = fault.NewFatalError(fault.ErrCodeUnknown, cmd.Name(), err.Error())
	}

	action := p.errorHandler.Handle(pErr)

	switch action {
	case fault.ActionRetry:
		return pErr

	case fault.ActionFail:
		_ = ctx.State.Transition(StateFailed, fmt.Sprintf("failed at %s: %s", cmd.Name(), err.Error()))
		p.publishFailedEvent(ctx, cmd.Name(), pErr)
		return pErr

	case fault.ActionPause:
		if pErr != nil {
			pauseState := StatePausedForValidation
			if pErr.Code == fault.ErrCodeNoTopics || pErr.Code == fault.ErrCodeNoCategories {
				pauseState = StatePausedNoResources
			}
			_ = ctx.State.Transition(pauseState, fmt.Sprintf("paused at %s: %s", cmd.Name(), err.Error()))
			p.publishPausedEvent(ctx)
			return nil
		}

	case fault.ActionRecover:
		p.log("Attempting to recover from error: %v", err)
		_ = ctx.State.Transition(StatePausedNoResources, fmt.Sprintf("recovering from %s", cmd.Name()))
		p.publishPausedEvent(ctx)
		return nil

	case fault.ActionContinue:
		p.log("Continuing despite error: %v", err)
		return nil

	default:
		return pErr
	}

	return pErr
}

func (p *Pipeline) handleCancellation(pctx *Context, err error) error {
	p.log("Pipeline cancelled: %v", err)
	_ = pctx.State.Transition(StateFailed, "cancelled")
	return err
}

func (p *Pipeline) publishEvent(event events.Event) {
	if p.eventBus != nil {
		p.eventBus.Publish(ctx.FastCtx(), event)
	}
}

func (p *Pipeline) publishCompletedEvent(ctx *Context) {
	var articleID, executionID int64

	if ctx.Publication != nil && ctx.Publication.Article != nil {
		articleID = ctx.Publication.Article.ID
	}

	if ctx.Execution != nil && ctx.Execution.Execution != nil {
		executionID = ctx.Execution.Execution.ID
	}

	p.publishEvent(events.NewEvent(
		pipevents.EventPipelineCompleted,
		&pipevents.PipelineCompletedEvent{
			JobID:       ctx.Job.ID,
			JobName:     ctx.Job.Name,
			Duration:    ctx.Duration(),
			ArticleID:   articleID,
			ExecutionID: executionID,
		},
	))
}

func (p *Pipeline) publishFailedEvent(ctx *Context, stepName string, err *fault.PipelineError) {
	p.publishEvent(events.NewEvent(
		pipevents.EventPipelineFailed,
		&pipevents.PipelineFailedEvent{
			JobID:       ctx.Job.ID,
			JobName:     ctx.Job.Name,
			Duration:    ctx.Duration(),
			ErrorCode:   string(err.Code),
			ErrorMsg:    err.Message,
			FailedStep:  stepName,
			FailedState: string(ctx.State.CurrentState()),
		},
	))
}

func (p *Pipeline) publishPausedEvent(ctx *Context) {
	reason := "unknown"
	state := ctx.State.CurrentState()

	if state == StatePausedForValidation {
		reason = "requires validation"
	} else if state == StatePausedNoResources {
		reason = "no resources available"
	}

	p.publishEvent(events.NewEvent(
		pipevents.EventPipelinePaused,
		&pipevents.PipelinePausedEvent{
			JobID:    ctx.Job.ID,
			JobName:  ctx.Job.Name,
			Reason:   reason,
			PausedAt: string(state),
		},
	))
}

func (p *Pipeline) publishStepFailedEvent(pctx *Context, stepName string, err error, duration time.Duration) {
	errorCode := "unknown"
	errorMsg := err.Error()

	var pErr *fault.PipelineError
	if errors.As(err, &pErr) {
		errorCode = string(pErr.Code)
		errorMsg = pErr.Message
	}

	p.publishEvent(events.NewEvent(
		pipevents.EventStepFailed,
		&pipevents.StepFailedEvent{
			JobID:     pctx.Job.ID,
			StepName:  stepName,
			Duration:  duration,
			ErrorCode: errorCode,
			ErrorMsg:  errorMsg,
			State:     string(pctx.State.CurrentState()),
		},
	))
}

func (p *Pipeline) log(format string, args ...any) {
	if p.logger != nil {
		p.logger.Infof(format, args...)
	}
}

type RetryStrategy interface {
	Wait(ctx context.Context, attempt int) error
}

type ExponentialBackoffRetry struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

func (r *ExponentialBackoffRetry) Wait(ctx context.Context, attempt int) error {
	if r.BaseDelay == 0 {
		r.BaseDelay = 100 * time.Millisecond
	}
	if r.MaxDelay == 0 {
		r.MaxDelay = 10 * time.Second
	}

	delay := r.BaseDelay * time.Duration(1<<uint(attempt))
	if delay > r.MaxDelay {
		delay = r.MaxDelay
	}

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
