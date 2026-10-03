package openaitest

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

const (
	minOutputTokens = 16
	maxCacheKey     = 64
	defaultEffort   = "medium"
	effortNone      = "none"
)

var (
	supportedEfforts = []string{"none", "low", "medium", "high", "xhigh", "max"}
	supportedTiers   = []string{"auto", "default", "flex", "fast", "priority"}
	samplingKeys     = []string{"temperature", "top_p"}
)

type check func(body map[string]any) (Reply, bool)

func refuse(body map[string]any) (Reply, bool) {
	for _, run := range []check{refuseModel, refuseEffort, refuseSampling, refuseTier, refuseCeiling, refuseCacheKey, refuseFormat, refuseTools, refuseInput} {
		if reply, refused := run(body); refused {
			return reply, true
		}
	}
	return Reply{}, false
}

func badRequest(param, code, message string) Reply {
	return Failure(http.StatusBadRequest, Fault{Type: invalidRequest, Param: param, Code: code, Message: message})
}

func textAt(object map[string]any, key string) (string, bool) {
	value, ok := object[key].(string)
	return value, ok
}

func objectAt(object map[string]any, key string) (map[string]any, bool) {
	value, ok := object[key].(map[string]any)
	return value, ok
}

func listAt(object map[string]any, key string) []any {
	if value, ok := object[key].([]any); ok {
		return value
	}
	return nil
}

func named(object map[string]any, key string) string {
	if value, ok := textAt(object, key); ok {
		return value
	}
	return ""
}

func refuseModel(body map[string]any) (Reply, bool) {
	if named(body, "model") != "" {
		return Reply{}, false
	}
	return badRequest("model", "missing_required_parameter", "Missing required parameter: 'model'."), true
}

func effortOf(body map[string]any) (string, bool) {
	reasoning, ok := objectAt(body, "reasoning")
	if !ok {
		return "", false
	}
	return textAt(reasoning, "effort")
}

func refuseEffort(body map[string]any) (Reply, bool) {
	effort, sent := effortOf(body)
	if !sent || slices.Contains(supportedEfforts, effort) {
		return Reply{}, false
	}
	return badRequest("reasoning.effort", "unsupported_value", "Unsupported value: '"+effort+"' is not supported with the '"+
		named(body, "model")+"' model. Supported values are: 'none', 'low', 'medium', 'high', 'xhigh', and 'max'."), true
}

func refuseSampling(body map[string]any) (Reply, bool) {
	effort, sent := effortOf(body)
	if !sent {
		effort = defaultEffort
	}
	if effort == effortNone {
		return Reply{}, false
	}
	for _, key := range samplingKeys {
		if value, present := body[key]; present && value != nil {
			return badRequest(key, "", "Unsupported parameter: '"+key+"' is not supported with this model."), true
		}
	}
	return Reply{}, false
}

func refuseTier(body map[string]any) (Reply, bool) {
	tier, sent := textAt(body, "service_tier")
	if !sent || slices.Contains(supportedTiers, tier) {
		return Reply{}, false
	}
	return badRequest("service_tier", "invalid_value", "Invalid value: '"+tier+
		"'. Supported values are: 'auto', 'default', 'fast', 'flex', and 'priority'."), true
}

func refuseCeiling(body map[string]any) (Reply, bool) {
	ceiling, sent := body["max_output_tokens"].(float64)
	if !sent || ceiling >= minOutputTokens {
		return Reply{}, false
	}
	return badRequest("max_output_tokens", "integer_below_min_value",
		"Invalid 'max_output_tokens': integer below minimum value. Expected a value >= 16, but got "+
			strconv.Itoa(int(ceiling))+" instead."), true
}

func refuseCacheKey(body map[string]any) (Reply, bool) {
	key, sent := textAt(body, "prompt_cache_key")
	if !sent || len(key) <= maxCacheKey {
		return Reply{}, false
	}
	return badRequest("prompt_cache_key", "string_above_max_length",
		"Invalid 'prompt_cache_key': string too long. Expected a string with maximum length 64, but got a string with length "+
			strconv.Itoa(len(key))+" instead."), true
}

func refuseFormat(body map[string]any) (Reply, bool) {
	text, ok := objectAt(body, "text")
	if !ok {
		return Reply{}, false
	}
	format, ok := objectAt(text, "format")
	if !ok || format["type"] != "json_schema" || format["strict"] != true {
		return Reply{}, false
	}
	problem := strictProblem(format["schema"])
	if problem == "" {
		return Reply{}, false
	}
	return badRequest("text.format.schema", "invalid_json_schema",
		"Invalid schema for response_format '"+named(format, "name")+"': "+problem), true
}

func refuseTools(body map[string]any) (Reply, bool) {
	for i, each := range listAt(body, "tools") {
		tool, isObject := each.(map[string]any)
		if !isObject || tool["type"] != "function" || tool["strict"] != true {
			continue
		}
		if problem := strictProblem(tool["parameters"]); problem != "" {
			return badRequest("tools["+strconv.Itoa(i)+"].parameters", "invalid_function_parameters",
				"Invalid schema for function '"+named(tool, "name")+"': "+problem), true
		}
	}
	return Reply{}, false
}

func strictProblem(schema any) string {
	root, ok := schema.(map[string]any)
	if !ok || root["type"] != "object" {
		found := "None"
		if ok {
			if kind, isText := textAt(root, "type"); isText {
				found = kind
			}
		}
		return "schema must be a JSON Schema of 'type: \"object\"', got 'type: \"" + found + "\"'."
	}
	return strictNode(root, nil)
}

func strictNode(node map[string]any, path []string) string {
	for i, branch := range listAt(node, "anyOf") {
		if nested, isObject := branch.(map[string]any); isObject {
			if problem := strictNode(nested, append(slices.Clone(path), "anyOf", strconv.Itoa(i))); problem != "" {
				return problem
			}
		}
	}
	if items, ok := objectAt(node, "items"); ok {
		if problem := strictNode(items, append(slices.Clone(path), "items")); problem != "" {
			return problem
		}
	}
	if !isObjectType(node["type"]) {
		return ""
	}
	return strictObject(node, path)
}

func strictObject(node map[string]any, path []string) string {
	where := "In context=(" + contextOf(path) + "), "
	if node["additionalProperties"] != false {
		return where + "'additionalProperties' is required to be supplied and to be false."
	}

	properties, _ := objectAt(node, "properties")
	required := listAt(node, "required")
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		if !slices.Contains(required, any(name)) {
			return where + "'required' is required to be supplied and to be an array including every key in properties. Missing '" + name + "'."
		}
	}
	for _, name := range names {
		if nested, ok := objectAt(properties, name); ok {
			if problem := strictNode(nested, append(slices.Clone(path), "properties", name)); problem != "" {
				return problem
			}
		}
	}
	return ""
}

func isObjectType(kind any) bool {
	switch typed := kind.(type) {
	case string:
		return typed == "object"
	case []any:
		return slices.Contains(typed, any("object"))
	default:
		return false
	}
}

func contextOf(path []string) string {
	quoted := make([]string, 0, len(path))
	for _, step := range path {
		quoted = append(quoted, "'"+step+"'")
	}
	return strings.Join(quoted, ", ")
}

func refuseInput(body map[string]any) (Reply, bool) {
	var calls []string
	answered := map[string]bool{}
	for _, each := range listAt(body, "input") {
		item, isObject := each.(map[string]any)
		if !isObject {
			continue
		}
		callID := named(item, "call_id")
		switch item["type"] {
		case "reasoning":
			if _, carried := textAt(item, "encrypted_content"); !carried {
				return Failure(http.StatusNotFound, Fault{
					Type: invalidRequest, Param: "input",
					Message: "Item with id '" + named(item, "id") + "' not found. Items are not persisted when `store` is set to false. " +
						"Try again with `store` set to true, or remove this item from your input.",
				}), true
			}
		case "function_call":
			calls = append(calls, callID)
		case "function_call_output":
			if !slices.Contains(calls, callID) {
				return badRequest("input", "", "No tool call found for function call output with call_id "+callID+"."), true
			}
			answered[callID] = true
		}
	}

	for _, callID := range calls {
		if !answered[callID] {
			return badRequest("input", "", "No tool output found for function call "+callID+"."), true
		}
	}
	return Reply{}, false
}
