package paging

import (
	"encoding/json"
	"testing"
	"time"
)

type coercionRow struct{}

func TestSortKindCoercion(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("CET", 3600))

	cases := []struct {
		name string
		key  SortKey[coercionRow]
		in   any
		want any
		ok   bool
	}{
		{name: "text from string", key: TextKey[coercionRow]("n", "n", nil), in: "a", want: "a", ok: true},
		{name: "text from bytes", key: TextKey[coercionRow]("n", "n", nil), in: []byte("a"), want: "a", ok: true},
		{name: "text from number", key: TextKey[coercionRow]("n", "n", nil), in: 1, ok: false},
		{name: "int from int", key: IntKey[coercionRow]("s", "s", nil), in: 7, want: int64(7), ok: true},
		{name: "int from int32", key: IntKey[coercionRow]("s", "s", nil), in: int32(3), want: int64(3), ok: true},
		{name: "int from int64", key: IntKey[coercionRow]("s", "s", nil), in: int64(3), want: int64(3), ok: true},
		{name: "int from float", key: IntKey[coercionRow]("s", "s", nil), in: float64(7), want: int64(7), ok: true},
		{name: "int from json number", key: IntKey[coercionRow]("s", "s", nil), in: json.Number("7"), want: int64(7), ok: true},
		{name: "int from fractional json number", key: IntKey[coercionRow]("s", "s", nil), in: json.Number("3.5"), ok: false},
		{name: "int from string", key: IntKey[coercionRow]("s", "s", nil), in: "7", ok: false},
		{name: "int from bool", key: IntKey[coercionRow]("s", "s", nil), in: true, ok: false},
		{name: "time from time", key: TimeKey[coercionRow]("t", "t", nil), in: stamp, want: "2026-01-02T02:04:05Z", ok: true},
		{name: "time from rfc3339", key: TimeKey[coercionRow]("t", "t", nil), in: "2026-01-02T02:04:05Z", want: "2026-01-02T02:04:05Z", ok: true},
		{name: "time from nonsense", key: TimeKey[coercionRow]("t", "t", nil), in: "yesterday", ok: false},
		{name: "time from bytes", key: TimeKey[coercionRow]("t", "t", nil), in: []byte("2026-01-02T02:04:05Z"), ok: false},
		{name: "nil is never coercible", key: TextKey[coercionRow]("n", "n", nil), in: nil, ok: false},
		{name: "unknown kind", key: SortKey[coercionRow]{kind: sortKind(200), Field: "x", Column: "x"}, in: "x", ok: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := tc.key.literal(tc.in)
			if ok != tc.ok {
				t.Fatalf("literal() ok = %v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("literal() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
