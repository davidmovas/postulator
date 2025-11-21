package commands

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type ExecutionProvider interface {
	Create(ctx context.Context, exec *entities.Execution) error
	Update(ctx context.Context, exec *entities.Execution) error
}
