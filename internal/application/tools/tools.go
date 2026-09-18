package tools

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Risk string

const (
	RiskRead      Risk = "read"
	RiskWrite     Risk = "write"
	RiskDangerous Risk = "dangerous"
)

func (r Risk) Valid() bool {
	switch r {
	case RiskRead, RiskWrite, RiskDangerous:
		return true
	default:
		return false
	}
}

func (r Risk) NeedsConfirmation() bool {
	return r == RiskWrite || r == RiskDangerous
}

type Def struct {
	Schema      *llm.Schema
	Name        string
	Description string
	Risk        Risk
}

type Binding struct {
	SiteID         string
	ConversationID string
	RunID          string
	Mode           agent.Mode
}

type Tool struct {
	Authorize func(ctx context.Context, b Binding) error
	Run       func(ctx context.Context, b Binding, args json.RawMessage) (any, error)
	Def       Def
}

type Confirmation struct {
	Status   string `json:"status"`
	ActionID string `json:"actionId"`
	Summary  string `json:"summary"`
}

const siteField = "siteId"

func NewTool[In, Out any](def Def, fn func(ctx context.Context, b Binding, in In) (Out, error)) Tool {
	schema, schemaErr := llm.SchemaFor[In]()
	def.Schema = schema

	return Tool{
		Def: def,
		Run: func(ctx context.Context, b Binding, args json.RawMessage) (any, error) {
			if schemaErr != nil {
				return nil, errors.Wrap(schemaErr, errors.Internal, "tool "+def.Name+" has no usable argument schema")
			}

			var in In
			if len(args) > 0 {
				if err := json.Unmarshal(args, &in); err != nil {
					return nil, errors.Wrap(err, errors.Invalid, "the arguments of tool "+def.Name+" are not readable")
				}
			}
			return fn(ctx, b, in)
		},
	}
}

func newSiteTool[In, Out any](def Def, fn func(ctx context.Context, b Binding, in In) (Out, error)) Tool {
	tool := NewTool(def, fn)
	tool.Authorize = requireSite
	tool.Def.Schema = withoutSite(tool.Def.Schema)
	return tool
}

func requireSite(_ context.Context, b Binding) error {
	if b.SiteID == "" {
		return errors.New(errors.Unauthorized, "this tool works inside one site and the conversation names none")
	}
	return nil
}

func withoutSite(schema *llm.Schema) *llm.Schema {
	if schema == nil || schema.Type != llm.SchemaObject {
		return schema
	}

	trimmed := *schema
	trimmed.Properties = make(map[string]*llm.Schema, len(schema.Properties))
	for name, property := range schema.Properties {
		if name == siteField {
			continue
		}
		trimmed.Properties[name] = property
	}
	trimmed.Required = slices.DeleteFunc(slices.Clone(schema.Required), func(name string) bool {
		return name == siteField
	})
	return &trimmed
}
