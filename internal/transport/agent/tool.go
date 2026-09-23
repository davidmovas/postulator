package agent

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"slices"
	"strings"

	"github.com/gollem-dev/gollem"

	applicationllm "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/log"
)

const resultKey = "result"

type gollemTool struct {
	tool    tools.Tool
	binding tools.Binding
}

func (t *gollemTool) Spec() gollem.ToolSpec {
	return gollem.ToolSpec{
		Name:        t.tool.Def.Name,
		Description: t.tool.Def.Description,
		Parameters:  parametersOf(t.tool.Def.Schema),
	}
}

func (t *gollemTool) Run(ctx context.Context, args map[string]any) (map[string]any, error) {
	if t.tool.Authorize != nil {
		if err := t.tool.Authorize(ctx, t.binding); err != nil {
			return nil, explain(err)
		}
	}

	encoded, err := json.Marshal(args)
	if err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "the arguments of tool "+t.tool.Def.Name+" are not readable")
	}

	out, err := t.tool.Run(ctx, t.binding, encoded)
	if err != nil {
		return nil, explain(err)
	}
	return objectOf(out)
}

func explain(err error) error {
	var known *errors.Error
	if !stderrors.As(err, &known) || len(known.Details) == 0 {
		return err
	}

	keys := make([]string, 0, len(known.Details))
	for key := range known.Details {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	named := make([]string, 0, len(keys))
	for _, key := range keys {
		value := known.Details[key]
		if log.IsSensitiveKey(key) {
			value = log.Mask
		}
		named = append(named, key+": "+fmt.Sprint(value))
	}
	return errors.New(known.Code, err.Error()+" ("+strings.Join(named, ", ")+")")
}

func objectOf(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "the tool answered something that cannot be encoded")
	}

	var object map[string]any
	if json.Unmarshal(encoded, &object) == nil && object != nil {
		return object, nil
	}

	var plain any
	if unmarshalErr := json.Unmarshal(encoded, &plain); unmarshalErr != nil {
		return nil, errors.Wrap(unmarshalErr, errors.Internal, "the tool answer could not be read back")
	}
	return map[string]any{resultKey: plain}, nil
}

func parametersOf(schema *applicationllm.Schema) map[string]*gollem.Parameter {
	if schema == nil || schema.Type != applicationllm.SchemaObject {
		return map[string]*gollem.Parameter{}
	}

	parameters := make(map[string]*gollem.Parameter, len(schema.Properties))
	for name, property := range schema.Properties {
		parameter := parameterOf(property)
		parameter.Required = slices.Contains(schema.Required, name)
		parameters[name] = parameter
	}
	return parameters
}

func parameterOf(schema *applicationllm.Schema) *gollem.Parameter {
	if schema == nil {
		return &gollem.Parameter{Type: gollem.TypeString}
	}

	parameter := &gollem.Parameter{
		Type:        gollem.ParameterType(schema.Type),
		Description: schema.Description,
		Enum:        schema.Enum,
		Minimum:     schema.Minimum,
		Maximum:     schema.Maximum,
	}
	if schema.Items != nil {
		parameter.Items = parameterOf(schema.Items)
	}
	if schema.Type == applicationllm.SchemaObject {
		parameter.Properties = make(map[string]*gollem.Parameter, len(schema.Properties))
		for name, property := range schema.Properties {
			nested := parameterOf(property)
			nested.Required = slices.Contains(schema.Required, name)
			parameter.Properties[name] = nested
		}
	}
	if schema.Type == applicationllm.SchemaArray && parameter.Items == nil {
		parameter.Items = &gollem.Parameter{Type: gollem.TypeString}
	}
	return parameter
}
