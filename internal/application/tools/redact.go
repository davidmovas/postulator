package tools

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/kernel/log"
)

func Redact(args json.RawMessage) json.RawMessage {
	if len(args) == 0 {
		return args
	}

	var decoded any
	if err := json.Unmarshal(args, &decoded); err != nil {
		return args
	}

	encoded, err := json.Marshal(scrub(decoded))
	if err != nil {
		return args
	}
	return encoded
}

func scrub(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			if log.IsSensitiveKey(key) {
				out[key] = log.Mask
				continue
			}
			out[key] = scrub(nested)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, nested := range typed {
			out[i] = scrub(nested)
		}
		return out
	default:
		return value
	}
}
