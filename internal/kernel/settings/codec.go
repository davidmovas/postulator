package settings

import (
	"encoding/json"
	"fmt"
	"time"
)

func decodeBool(raw json.RawMessage) (bool, error) {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("value must be a boolean: %w", err)
	}
	return value, nil
}

func encodeBool(value bool) (json.RawMessage, error) {
	return json.Marshal(value)
}

func decodeInt(raw json.RawMessage) (int, error) {
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("value must be a whole number: %w", err)
	}
	return value, nil
}

func encodeInt(value int) (json.RawMessage, error) {
	return json.Marshal(value)
}

func decodeString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("value must be a string: %w", err)
	}
	return value, nil
}

func encodeString(value string) (json.RawMessage, error) {
	return json.Marshal(value)
}

func decodeDuration(raw json.RawMessage) (time.Duration, error) {
	text, err := decodeString(raw)
	if err != nil {
		return 0, fmt.Errorf("value must be a duration string such as \"30s\": %w", err)
	}
	parsed, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("value must be a duration string such as \"30s\": %w", err)
	}
	return parsed, nil
}

func encodeDuration(value time.Duration) (json.RawMessage, error) {
	return json.Marshal(value.String())
}
