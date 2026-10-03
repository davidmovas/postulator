package agent

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	rootLabel      = "the arguments"
	problemJoint   = "; "
	enumJoint      = ", "
	formatAsNumber = 'f'
)

type findings struct {
	problems []string
}

func (f *findings) add(path, problem string) {
	if path == "" {
		path = rootLabel
	}
	f.problems = append(f.problems, path+": "+problem)
}

func check(tool string, schema *llm.Schema, args json.RawMessage) error {
	if schema == nil {
		return nil
	}

	value, err := decoded(args)
	if err != nil {
		return errors.New(errors.Invalid, "the arguments of tool "+tool+" are not JSON; call it again with a JSON object").
			WithDetail("tool", tool)
	}
	if value == nil {
		value = map[string]any{}
	}

	var found findings
	found.inspect(schema, value, "")
	if len(found.problems) == 0 {
		return nil
	}
	return errors.New(errors.Invalid, "the arguments of tool "+tool+" do not fit its schema: "+
		strings.Join(found.problems, problemJoint)+"; correct them and call it again").
		WithDetail("tool", tool)
}

func (f *findings) inspect(schema *llm.Schema, value any, path string) {
	switch schema.Type {
	case llm.SchemaString:
		f.text(schema, value, path)
	case llm.SchemaNumber:
		number, ok := numberOf(value)
		if !ok {
			f.add(path, "expected a number")
			return
		}
		f.bounded(schema, number, path, "number")
	case llm.SchemaInteger:
		number, ok := integerOf(value)
		if !ok {
			f.add(path, "expected an integer")
			return
		}
		f.bounded(schema, number, path, "integer")
	case llm.SchemaBoolean:
		if _, ok := value.(bool); !ok {
			f.add(path, "expected true or false")
		}
	case llm.SchemaArray:
		f.list(schema, value, path)
	case llm.SchemaObject:
		f.object(schema, value, path)
	}
}

func (f *findings) text(schema *llm.Schema, value any, path string) {
	text, ok := value.(string)
	if !ok {
		f.add(path, "expected a string")
		return
	}
	if len(schema.Enum) > 0 && !slices.Contains(schema.Enum, text) {
		f.add(path, "value not in enum; it takes one of "+strings.Join(schema.Enum, enumJoint))
	}
}

func (f *findings) bounded(schema *llm.Schema, number float64, path, kind string) {
	if schema.Minimum != nil && number < *schema.Minimum {
		f.add(path, kind+" too small; the minimum is "+strconv.FormatFloat(*schema.Minimum, formatAsNumber, -1, 64))
	}
	if schema.Maximum != nil && number > *schema.Maximum {
		f.add(path, kind+" too large; the maximum is "+strconv.FormatFloat(*schema.Maximum, formatAsNumber, -1, 64))
	}
}

func (f *findings) list(schema *llm.Schema, value any, path string) {
	items, ok := value.([]any)
	if !ok {
		f.add(path, "expected a list")
		return
	}
	if schema.Items == nil {
		return
	}
	for i, item := range items {
		if item != nil {
			f.inspect(schema.Items, item, path+"["+strconv.Itoa(i)+"]")
		}
	}
}

func (f *findings) object(schema *llm.Schema, value any, path string) {
	object, ok := value.(map[string]any)
	if !ok {
		f.add(path, "expected an object")
		return
	}
	for _, name := range slices.Sorted(maps.Keys(schema.Properties)) {
		at := name
		if path != "" {
			at = path + "." + name
		}
		held, present := object[name]
		if !present || held == nil {
			if slices.Contains(schema.Required, name) {
				f.add(at, "required parameter missing")
			}
			continue
		}
		if property := schema.Properties[name]; property != nil {
			f.inspect(property, held, at)
		}
	}
}

func numberOf(value any) (float64, bool) {
	held, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	number, err := held.Float64()
	return number, err == nil
}

func integerOf(value any) (float64, bool) {
	held, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	if whole, err := held.Int64(); err == nil {
		return float64(whole), true
	}
	number, err := held.Float64()
	if err != nil || number != float64(int64(number)) {
		return 0, false
	}
	return number, true
}
