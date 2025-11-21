package commands

import (
	"github.com/davidmovas/postulator/internal/domain/jobs/execution/pipeline"
)

type ParallelCommand struct {
	*BaseCommand
	commands []Command
}

func NewParallelCommand(name string, requiredState, nextState pipeline.State) *ParallelCommand {
	return &ParallelCommand{
		BaseCommand: NewBaseCommand(name, requiredState, nextState),
		commands:    make([]Command, 0),
	}
}

func (p *ParallelCommand) Add(cmd Command) *ParallelCommand {
	p.commands = append(p.commands, cmd)
	return p
}

func (p *ParallelCommand) Execute(ctx *pipeline.Context) error {
	for _, cmd := range p.commands {
		if !cmd.CanExecute(ctx) {
			continue
		}

		if err := cmd.Execute(ctx); err != nil {
			return err
		}
	}
	return nil
}
