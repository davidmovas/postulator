package commands

import (
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
)

type BaseCommand struct {
	name          string
	requiredState pipeline.State
	nextState     pipeline.State
	retryable     bool
	maxRetries    int
}

func NewBaseCommand(name string, requiredState, nextState pipeline.State) *BaseCommand {
	return &BaseCommand{
		name:          name,
		requiredState: requiredState,
		nextState:     nextState,
		retryable:     false,
		maxRetries:    0,
	}
}

func (c *BaseCommand) WithRetry(maxRetries int) *BaseCommand {
	c.retryable = true
	c.maxRetries = maxRetries
	return c
}

func (c *BaseCommand) Name() string {
	return c.name
}

func (c *BaseCommand) CanExecute(ctx *pipeline.Context) bool {
	return true
}

func (c *BaseCommand) RequiredState() pipeline.State {
	return c.requiredState
}

func (c *BaseCommand) NextState() pipeline.State {
	return c.nextState
}

func (c *BaseCommand) OnError(ctx *pipeline.Context, err error) error {
	return err
}

func (c *BaseCommand) IsRetryable() bool {
	return c.retryable
}

func (c *BaseCommand) MaxRetries() int {
	return c.maxRetries
}

type ConditionalCommand struct {
	*BaseCommand
	condition func(*pipeline.Context) bool
}

func NewConditionalCommand(
	name string,
	requiredState,
	nextState pipeline.State,
	condition func(*pipeline.Context) bool,
) *ConditionalCommand {
	return &ConditionalCommand{
		BaseCommand: NewBaseCommand(name, requiredState, nextState),
		condition:   condition,
	}
}

func (c *ConditionalCommand) CanExecute(pctx *pipeline.Context) bool {
	if c.condition == nil {
		return true
	}
	return c.condition(pctx)
}

type CommandGroup struct {
	*BaseCommand
	commands []pipeline.Command
}

func NewCommandGroup(name string, requiredState, nextState pipeline.State) *CommandGroup {
	return &CommandGroup{
		BaseCommand: NewBaseCommand(name, requiredState, nextState),
		commands:    make([]pipeline.Command, 0),
	}
}

func (g *CommandGroup) Add(cmd pipeline.Command) *CommandGroup {
	g.commands = append(g.commands, cmd)
	return g
}

func (g *CommandGroup) Execute(ctx *pipeline.Context) error {
	for _, cmd := range g.commands {
		if !cmd.CanExecute(ctx) {
			continue
		}

		if err := cmd.Execute(ctx); err != nil {
			return err
		}
	}
	return nil
}
