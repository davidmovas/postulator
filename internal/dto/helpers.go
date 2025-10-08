package dto

import "time"

func Success[T any](data T) Response[T] {
	return Response[T]{
		Success: true,
		Data:    data,
	}
}

func Failure[T any](code, message string, context map[string]any) Response[T] {
	return Response[T]{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Context: context,
		},
	}
}

func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}

func ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func ParseTimePtr(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
