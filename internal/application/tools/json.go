package tools

import (
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func decodeJSONArgument(raw string, into any, what string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New(errors.Invalid, "the "+what+" must be a JSON object").WithDetail("field", what)
	}
	if err := json.Unmarshal([]byte(trimmed), into); err != nil {
		return errors.Wrap(err, errors.Invalid, "the "+what+" is not readable as JSON")
	}
	return nil
}
