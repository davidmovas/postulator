package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
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
	Approved       bool
}

type Tool struct {
	Authorize func(ctx context.Context, b Binding) error
	Check     func(args json.RawMessage) error
	Run       func(ctx context.Context, b Binding, args json.RawMessage) (any, error)
	Def       Def
}

type Confirmation struct {
	Status   string `json:"status"`
	ActionID string `json:"actionId"`
	Summary  string `json:"summary"`
}

const (
	siteField       = "siteId"
	sortField       = "sort"
	fieldField      = "field"
	sortByCreatedAt = "createdAt"
)

func NewTool[In, Out any](def Def, fn func(ctx context.Context, b Binding, in In) (Out, error)) Tool {
	schema, schemaErr := llm.SchemaFor[In]()
	def.Schema = schema

	read := func(args json.RawMessage) (In, error) {
		var in In
		if schemaErr != nil {
			return in, errors.Wrap(schemaErr, errors.Internal, "tool "+def.Name+" has no usable argument schema")
		}
		if len(args) == 0 {
			return in, nil
		}

		decoder := json.NewDecoder(bytes.NewReader(args))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&in); err != nil {
			return in, errors.New(errors.Invalid, "the arguments of tool "+def.Name+" are not what it takes").
				WithDetail("tool", def.Name).WithInternal(err)
		}
		return in, nil
	}

	return Tool{
		Def: def,
		Check: func(args json.RawMessage) error {
			_, err := read(args)
			return err
		},
		Run: func(ctx context.Context, b Binding, args json.RawMessage) (any, error) {
			in, err := read(args)
			if err != nil {
				return nil, err
			}
			return fn(ctx, b, in)
		},
	}
}

func checking[In any](tool Tool, validate func(in In) error) Tool {
	decode := tool.Check
	tool.Check = func(args json.RawMessage) error {
		if err := decode(args); err != nil {
			return err
		}

		var in In
		if len(args) == 0 {
			return validate(in)
		}
		if err := json.Unmarshal(args, &in); err != nil {
			return errors.Wrap(err, errors.Invalid, "the arguments of tool "+tool.Def.Name+" are not what it takes")
		}
		return validate(in)
	}
	return tool
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

func sortedBy(tool Tool, fields ...string) Tool {
	schema := tool.Def.Schema
	if schema == nil || schema.Properties == nil {
		return tool
	}
	ordering, ok := schema.Properties[sortField]
	if !ok || ordering.Properties == nil {
		return tool
	}
	key, ok := ordering.Properties[fieldField]
	if !ok {
		return tool
	}

	chosen := *key
	chosen.Enum = fields
	nested := *ordering
	nested.Properties = maps.Clone(ordering.Properties)
	nested.Properties[fieldField] = &chosen

	widened := *schema
	widened.Properties = maps.Clone(schema.Properties)
	widened.Properties[sortField] = &nested
	tool.Def.Schema = &widened
	return tool
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
