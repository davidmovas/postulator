package wails

import (
	"encoding/json"
	stderrors "errors"
	"maps"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const fallbackBody = `{"code":"INTERNAL","message":"unexpected internal error"}`

type Retry struct {
	AfterMs int64 `json:"afterMs"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Retry   *Retry         `json:"retry,omitempty"`
}

func describe(err error) Error {
	code, message := errors.Describe(err)
	described := Error{Code: string(code), Message: message}

	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil {
		return described
	}

	if kernel.Code != errors.Internal && len(kernel.Details) > 0 {
		described.Details = maps.Clone(kernel.Details)
	}
	if kernel.Retry != nil {
		described.Retry = &Retry{AfterMs: kernel.Retry.After.Milliseconds()}
	}
	return described
}

func Convert(err error) error {
	if err == nil {
		return nil
	}

	described := describe(err)
	converted := errors.New(errors.Code(described.Code), described.Message)
	for key, value := range described.Details {
		converted = converted.WithDetail(key, value)
	}
	if described.Retry != nil {
		converted = converted.WithRetry(time.Duration(described.Retry.AfterMs) * time.Millisecond)
	}
	return converted
}

func MarshalError(err error) []byte {
	encoded, marshalErr := json.Marshal(describe(err))
	if marshalErr != nil {
		return []byte(fallbackBody)
	}
	return encoded
}
