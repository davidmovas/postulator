package log

import (
	"slices"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const Mask = "***"

var redacted = []string{"password", "apikey", "token", "authorization"}

func IsSensitiveKey(key string) bool {
	return slices.Contains(redacted, strings.ToLower(key))
}

func redact(fields []zapcore.Field) []zapcore.Field {
	var out []zapcore.Field
	for i, field := range fields {
		if !IsSensitiveKey(field.Key) {
			continue
		}
		if out == nil {
			out = slices.Clone(fields)
		}
		out[i] = zap.String(field.Key, Mask)
	}
	if out == nil {
		return fields
	}
	return out
}

type redacting struct {
	zapcore.Core
}

func (c redacting) With(fields []zapcore.Field) zapcore.Core {
	return redacting{Core: c.Core.With(redact(fields))}
}

func (c redacting) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c redacting) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(entry, redact(fields))
}
