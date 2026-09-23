package wails

import (
	"context"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

type ToolCatalog interface {
	Names() []string
	Lookup(name string) (tools.Def, bool)
}

type ListToolsRequest struct{}

type Tool struct {
	Schema      *llm.Schema `json:"schema,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Risk        string      `json:"risk"`
}

type ListToolsResponse struct {
	Tools []Tool `json:"tools"`
}

type ToolsService struct {
	list middleware.Handler[ListToolsRequest, ListToolsResponse]
}

func NewToolsService(logger *zap.Logger, catalog Source[ToolCatalog]) *ToolsService {
	return &ToolsService{list: Wrap(logger, "tools.list", listTools(catalog))}
}

func listTools(catalog Source[ToolCatalog]) middleware.Handler[ListToolsRequest, ListToolsResponse] {
	return func(_ context.Context, _ ListToolsRequest) (ListToolsResponse, error) {
		live, err := catalog()
		if err != nil {
			return ListToolsResponse{}, err
		}

		names := live.Names()
		listed := make([]Tool, 0, len(names))
		for _, name := range names {
			def, known := live.Lookup(name)
			if !known {
				continue
			}
			listed = append(listed, Tool{
				Name:        def.Name,
				Description: def.Description,
				Risk:        string(def.Risk),
				Schema:      def.Schema,
			})
		}
		return ListToolsResponse{Tools: listed}, nil
	}
}

func (s *ToolsService) List(c context.Context, req ListToolsRequest) (ListToolsResponse, error) {
	return s.list(c, req)
}
