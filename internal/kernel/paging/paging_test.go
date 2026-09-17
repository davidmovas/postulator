package paging_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type row struct {
	createdAt time.Time
	id        string
	name      string
	score     int64
}

func rowKeyset(desc bool) paging.Keyset[row] {
	return paging.Keyset[row]{
		IDColumn: "id",
		Desc:     desc,
		ID:       func(r row) string { return r.id },
		Keys: []paging.SortKey[row]{
			paging.TimeKey[row]("createdAt", "created_at", func(r row) any { return r.createdAt }),
		},
	}
}

func mustCursor(t *testing.T, k paging.Keyset[row], r row) paging.Cursor {
	t.Helper()
	cursor, err := k.Encode(r)
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}
	return cursor
}

func TestRequestNormalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   int
		want int
	}{
		{name: "zero falls back to the default", in: 0, want: 50},
		{name: "negative falls back to the default", in: -7, want: 50},
		{name: "within range is kept", in: 25, want: 25},
		{name: "at the maximum is kept", in: 500, want: 500},
		{name: "above the maximum is clamped", in: 9999, want: 500},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := (paging.Request{Limit: tc.in}).Normalize().Limit; got != tc.want {
				t.Fatalf("Normalize().Limit = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)

	cases := []struct {
		name string
		keys []paging.SortKey[row]
		want []any
	}{
		{
			name: "text key",
			keys: []paging.SortKey[row]{paging.TextKey[row]("name", "name", func(r row) any { return r.name })},
			want: []any{"alpha"},
		},
		{
			name: "time key",
			keys: []paging.SortKey[row]{paging.TimeKey[row]("createdAt", "created_at", func(r row) any { return r.createdAt })},
			want: []any{"2026-09-17T10:30:00Z"},
		},
		{
			name: "int key",
			keys: []paging.SortKey[row]{paging.IntKey[row]("score", "score", func(r row) any { return r.score })},
			want: []any{int64(42)},
		},
		{
			name: "compound key",
			keys: []paging.SortKey[row]{
				paging.TimeKey[row]("createdAt", "created_at", func(r row) any { return r.createdAt }),
				paging.TextKey[row]("name", "name", func(r row) any { return r.name }),
			},
			want: []any{"2026-09-17T10:30:00Z", "alpha"},
		},
	}

	item := row{id: "9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1", name: "alpha", score: 42, createdAt: stamp}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			keyset := paging.Keyset[row]{IDColumn: "id", ID: func(r row) string { return r.id }, Keys: tc.keys}
			cursor, err := keyset.Encode(item)
			if err != nil {
				t.Fatalf("Encode() error: %v", err)
			}

			position, err := keyset.Position(cursor)
			if err != nil {
				t.Fatalf("Position() error: %v", err)
			}
			if position.ID != item.id {
				t.Fatalf("ID = %q, want %q", position.ID, item.id)
			}
			if position.Order != paging.Asc {
				t.Fatalf("Order = %q, want %q", position.Order, paging.Asc)
			}
			if len(position.Values) != len(tc.want) {
				t.Fatalf("Values = %v, want %v", position.Values, tc.want)
			}
			for i, want := range tc.want {
				if position.Values[i] != want {
					t.Fatalf("Values[%d] = %#v, want %#v", i, position.Values[i], want)
				}
			}
		})
	}
}

func TestCursorRejection(t *testing.T) {
	t.Parallel()

	keyset := rowKeyset(false)
	stamp := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)
	valid := mustCursor(t, keyset, row{id: "9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1", createdAt: stamp})

	cases := []struct {
		name   string
		keyset paging.Keyset[row]
		cursor paging.Cursor
	}{
		{name: "not base64", keyset: keyset, cursor: paging.Cursor("!!not base64!!")},
		{name: "not json", keyset: keyset, cursor: paging.Cursor("aGVsbG8gd29ybGQ")},
		{name: "order mismatch", keyset: rowKeyset(true), cursor: valid},
		{name: "field mismatch", keyset: paging.Keyset[row]{
			IDColumn: "id",
			ID:       func(r row) string { return r.id },
			Keys:     []paging.SortKey[row]{paging.TextKey[row]("name", "name", func(r row) any { return r.name })},
		}, cursor: valid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tc.keyset.Position(tc.cursor); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Position() error = %v, want code %s", err, errors.Invalid)
			}
		})
	}
}

func TestCursorArityMismatchIsInvalid(t *testing.T) {
	t.Parallel()

	compound := paging.Keyset[row]{
		IDColumn: "id",
		ID:       func(r row) string { return r.id },
		Keys: []paging.SortKey[row]{
			paging.TimeKey[row]("createdAt", "created_at", func(r row) any { return r.createdAt }),
			paging.TextKey[row]("name", "name", func(r row) any { return r.name }),
		},
	}
	cursor := mustCursor(t, compound, row{id: "a", createdAt: time.Now(), name: "n"})

	truncated := paging.Keyset[row]{
		IDColumn: "id",
		ID:       func(r row) string { return r.id },
		Keys: []paging.SortKey[row]{
			paging.TimeKey[row]("createdAt", "created_at", func(r row) any { return r.createdAt }),
		},
	}
	if _, err := truncated.Position(cursor); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Position() error = %v, want code %s", err, errors.Invalid)
	}
}

func TestKeysetApply(t *testing.T) {
	t.Parallel()

	stamp := time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)
	id := "9f0d0d22-6f4f-4c1a-9c07-5b6c1f6bd9a1"
	item := row{id: id, createdAt: stamp}

	ascending := rowKeyset(false)
	descending := rowKeyset(true)

	cases := []struct {
		name     string
		keyset   paging.Keyset[row]
		request  paging.Request
		wantSQL  string
		wantArgs []any
	}{
		{
			name:    "first page ascending",
			keyset:  ascending,
			request: paging.Request{Limit: 50},
			wantSQL: "SELECT * FROM pages ORDER BY created_at ASC, id ASC LIMIT 51",
		},
		{
			name:     "after ascending",
			keyset:   ascending,
			request:  paging.Request{After: mustCursor(t, ascending, item), Limit: 50},
			wantSQL:  "SELECT * FROM pages WHERE (created_at, id) > (?, ?) ORDER BY created_at ASC, id ASC LIMIT 51",
			wantArgs: []any{"2026-09-17T10:30:00Z", id},
		},
		{
			name:     "before ascending mirrors to descending",
			keyset:   ascending,
			request:  paging.Request{Before: mustCursor(t, ascending, item), Limit: 50},
			wantSQL:  "SELECT * FROM pages WHERE (created_at, id) < (?, ?) ORDER BY created_at DESC, id DESC LIMIT 51",
			wantArgs: []any{"2026-09-17T10:30:00Z", id},
		},
		{
			name:     "after descending",
			keyset:   descending,
			request:  paging.Request{After: mustCursor(t, descending, item), Limit: 50},
			wantSQL:  "SELECT * FROM pages WHERE (created_at, id) < (?, ?) ORDER BY created_at DESC, id DESC LIMIT 51",
			wantArgs: []any{"2026-09-17T10:30:00Z", id},
		},
		{
			name:     "before descending mirrors to ascending",
			keyset:   descending,
			request:  paging.Request{Before: mustCursor(t, descending, item), Limit: 50},
			wantSQL:  "SELECT * FROM pages WHERE (created_at, id) > (?, ?) ORDER BY created_at ASC, id ASC LIMIT 51",
			wantArgs: []any{"2026-09-17T10:30:00Z", id},
		},
		{
			name:    "limit is clamped before n+1",
			keyset:  ascending,
			request: paging.Request{Limit: 9999},
			wantSQL: "SELECT * FROM pages ORDER BY created_at ASC, id ASC LIMIT 501",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			builder, err := tc.keyset.Apply(squirrel.Select("*").From("pages"), tc.request)
			if err != nil {
				t.Fatalf("Apply() error: %v", err)
			}
			sql, args, err := builder.ToSql()
			if err != nil {
				t.Fatalf("ToSql() error: %v", err)
			}
			if sql != tc.wantSQL {
				t.Fatalf("sql = %q, want %q", sql, tc.wantSQL)
			}
			if len(args) != len(tc.wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, tc.wantArgs)
			}
			for i, want := range tc.wantArgs {
				if args[i] != want {
					t.Fatalf("args[%d] = %#v, want %#v", i, args[i], want)
				}
			}
		})
	}
}

func TestKeysetApplyCompoundPredicate(t *testing.T) {
	t.Parallel()

	keyset := paging.Keyset[row]{
		IDColumn: "id",
		ID:       func(r row) string { return r.id },
		Keys: []paging.SortKey[row]{
			paging.IntKey[row]("score", "score", func(r row) any { return r.score }),
			paging.TextKey[row]("name", "name", func(r row) any { return r.name }),
		},
	}
	cursor := mustCursor(t, keyset, row{id: "x", score: 7, name: "alpha"})

	builder, err := keyset.Apply(squirrel.Select("id").From("pages"), paging.Request{After: cursor, Limit: 2})
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	sql, args, err := builder.ToSql()
	if err != nil {
		t.Fatalf("ToSql() error: %v", err)
	}

	const want = "SELECT id FROM pages WHERE (score, name, id) > (?, ?, ?) ORDER BY score ASC, name ASC, id ASC LIMIT 3"
	if sql != want {
		t.Fatalf("sql = %q, want %q", sql, want)
	}
	if len(args) != 3 || args[0] != int64(7) || args[1] != "alpha" || args[2] != "x" {
		t.Fatalf("args = %#v", args)
	}
}

func TestKeysetApplyRejectsBadRequests(t *testing.T) {
	t.Parallel()

	keyset := rowKeyset(false)
	cursor := mustCursor(t, keyset, row{id: "a", createdAt: time.Now()})

	cases := []struct {
		name    string
		keyset  paging.Keyset[row]
		request paging.Request
		code    errors.Code
	}{
		{name: "both cursors", keyset: keyset, request: paging.Request{After: cursor, Before: cursor}, code: errors.Invalid},
		{name: "bad cursor", keyset: keyset, request: paging.Request{After: paging.Cursor("!!")}, code: errors.Invalid},
		{name: "no keys", keyset: paging.Keyset[row]{IDColumn: "id", ID: func(r row) string { return r.id }}, code: errors.Internal},
		{name: "no id column", keyset: paging.Keyset[row]{ID: func(r row) string { return r.id }, Keys: keyset.Keys}, code: errors.Internal},
		{name: "no id accessor", keyset: paging.Keyset[row]{IDColumn: "id", Keys: keyset.Keys}, code: errors.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tc.keyset.Apply(squirrel.Select("*").From("pages"), tc.request); !errors.IsCode(err, tc.code) {
				t.Fatalf("Apply() error = %v, want code %s", err, tc.code)
			}
		})
	}
}

func TestEncodeRejectsUncoercibleValue(t *testing.T) {
	t.Parallel()

	keyset := paging.Keyset[row]{
		IDColumn: "id",
		ID:       func(r row) string { return r.id },
		Keys:     []paging.SortKey[row]{paging.IntKey[row]("score", "score", func(row) any { return struct{}{} })},
	}
	if _, err := keyset.Encode(row{id: "a"}); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("Encode() error = %v, want code %s", err, errors.Internal)
	}
}

func TestCut(t *testing.T) {
	t.Parallel()

	keyset := rowKeyset(false)
	base := time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)
	rows := make([]row, 4)
	for i := range rows {
		rows[i] = row{id: string(rune('a' + i)), createdAt: base.Add(time.Duration(i) * time.Hour)}
	}

	t.Run("full page has no cursors", func(t *testing.T) {
		t.Parallel()
		list, err := keyset.Cut(rows[:2], paging.Request{Limit: 2})
		if err != nil {
			t.Fatalf("Cut() error: %v", err)
		}
		if list.HasMore {
			t.Fatal("HasMore must be false when the page is not overfull")
		}
		if list.Next != "" || list.Prev != "" {
			t.Fatalf("cursors = %q/%q, want empty", list.Next, list.Prev)
		}
		if len(list.Items) != 2 {
			t.Fatalf("Items = %d, want 2", len(list.Items))
		}
	})

	t.Run("overfull page trims and sets next", func(t *testing.T) {
		t.Parallel()
		list, err := keyset.Cut(rows, paging.Request{Limit: 3})
		if err != nil {
			t.Fatalf("Cut() error: %v", err)
		}
		if !list.HasMore {
			t.Fatal("HasMore must be true when more rows were read than requested")
		}
		if len(list.Items) != 3 {
			t.Fatalf("Items = %d, want 3", len(list.Items))
		}
		if list.Next == "" {
			t.Fatal("Next must be set when there is another page")
		}
		if list.Prev != "" {
			t.Fatal("Prev must be empty on the first page")
		}

		position, err := keyset.Position(list.Next)
		if err != nil {
			t.Fatalf("Position() error: %v", err)
		}
		if position.ID != rows[2].id {
			t.Fatalf("Next points at %q, want %q", position.ID, rows[2].id)
		}
	})

	t.Run("after cursor sets prev", func(t *testing.T) {
		t.Parallel()
		list, err := keyset.Cut(rows[1:3], paging.Request{After: mustCursor(t, keyset, rows[0]), Limit: 2})
		if err != nil {
			t.Fatalf("Cut() error: %v", err)
		}
		if list.Prev == "" {
			t.Fatal("Prev must be set when the page was reached with a cursor")
		}
		position, err := keyset.Position(list.Prev)
		if err != nil {
			t.Fatalf("Position() error: %v", err)
		}
		if position.ID != rows[1].id {
			t.Fatalf("Prev points at %q, want %q", position.ID, rows[1].id)
		}
	})

	t.Run("before cursor reverses rows back into natural order", func(t *testing.T) {
		t.Parallel()
		descending := []row{rows[2], rows[1], rows[0]}
		list, err := keyset.Cut(descending, paging.Request{Before: mustCursor(t, keyset, rows[3]), Limit: 3})
		if err != nil {
			t.Fatalf("Cut() error: %v", err)
		}
		if list.HasMore {
			t.Fatal("HasMore must be false when the backward page is not overfull")
		}
		got := []string{list.Items[0].id, list.Items[1].id, list.Items[2].id}
		want := []string{rows[0].id, rows[1].id, rows[2].id}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Items = %v, want %v", got, want)
			}
		}
		if list.Next == "" {
			t.Fatal("Next must be set when paging backwards")
		}
		if list.Prev != "" {
			t.Fatal("Prev must be empty when no older rows remain")
		}
	})

	t.Run("overfull backward page keeps the rows nearest the cursor", func(t *testing.T) {
		t.Parallel()
		descending := []row{rows[3], rows[2], rows[1], rows[0]}
		list, err := keyset.Cut(descending, paging.Request{Before: mustCursor(t, keyset, rows[3]), Limit: 3})
		if err != nil {
			t.Fatalf("Cut() error: %v", err)
		}
		if !list.HasMore {
			t.Fatal("HasMore must be true when more rows were read than requested")
		}
		got := []string{list.Items[0].id, list.Items[1].id, list.Items[2].id}
		want := []string{rows[1].id, rows[2].id, rows[3].id}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Items = %v, want %v", got, want)
			}
		}
		if list.Next == "" || list.Prev == "" {
			t.Fatalf("cursors = %q/%q, want both set", list.Next, list.Prev)
		}
	})

	t.Run("empty page", func(t *testing.T) {
		t.Parallel()
		list, err := keyset.Cut(nil, paging.Request{Limit: 3})
		if err != nil {
			t.Fatalf("Cut() error: %v", err)
		}
		if list.HasMore || list.Next != "" || list.Prev != "" || len(list.Items) != 0 {
			t.Fatalf("empty page = %+v", list)
		}
	})
}

func TestSliceMarshalsNilAsArray(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   paging.Slice[string]
		want string
	}{
		{name: "nil", in: nil, want: "[]"},
		{name: "empty", in: paging.Slice[string]{}, want: "[]"},
		{name: "populated", in: paging.Slice[string]{"a", "b"}, want: `["a","b"]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestListMarshalsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(paging.List[string]{
		Cursors: paging.Cursors{Next: "n", Prev: "p"},
		HasMore: true,
	})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"nextCursor":"n","prevCursor":"p","items":[],"hasMore":true}`
	if string(encoded) != want {
		t.Fatalf("Marshal() = %s, want %s", encoded, want)
	}
}

func TestKeysetRejectsIncompleteSortKeys(t *testing.T) {
	t.Parallel()

	id := func(r row) string { return r.id }
	value := func(r row) any { return r.name }

	cases := []struct {
		name   string
		keyset paging.Keyset[row]
	}{
		{name: "no field", keyset: paging.Keyset[row]{IDColumn: "id", ID: id, Keys: []paging.SortKey[row]{{Kind: paging.Text, Column: "name", Value: value}}}},
		{name: "no column", keyset: paging.Keyset[row]{IDColumn: "id", ID: id, Keys: []paging.SortKey[row]{{Kind: paging.Text, Field: "name", Value: value}}}},
		{name: "no accessor", keyset: paging.Keyset[row]{IDColumn: "id", ID: id, Keys: []paging.SortKey[row]{paging.TextKey[row]("name", "name", nil)}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tc.keyset.Encode(row{}); !errors.IsCode(err, errors.Internal) {
				t.Fatalf("Encode() error = %v, want code %s", err, errors.Internal)
			}
			if _, err := tc.keyset.Position("x"); !errors.IsCode(err, errors.Internal) {
				t.Fatalf("Position() error = %v, want code %s", err, errors.Internal)
			}
			if _, err := tc.keyset.Cut(nil, paging.Request{}); !errors.IsCode(err, errors.Internal) {
				t.Fatalf("Cut() error = %v, want code %s", err, errors.Internal)
			}
		})
	}
}

func TestPositionRejectsEmptyID(t *testing.T) {
	t.Parallel()

	raw := base64.RawURLEncoding.EncodeToString([]byte(`{"o":"createdAt","d":"asc","v":["2026-09-17T10:30:00Z"],"i":""}`))
	if _, err := rowKeyset(false).Position(paging.Cursor(raw)); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Position() error = %v, want code %s", err, errors.Invalid)
	}
}

func TestPositionRejectsMistypedValue(t *testing.T) {
	t.Parallel()

	id := func(r row) string { return r.id }
	numeric := paging.Keyset[row]{IDColumn: "id", ID: id, Keys: []paging.SortKey[row]{paging.IntKey[row]("v", "v", func(r row) any { return r.score })}}
	boolean := paging.Keyset[row]{IDColumn: "id", ID: id, Keys: []paging.SortKey[row]{paging.BoolKey[row]("v", "v", func(row) any { return true })}}

	cursor, err := numeric.Encode(row{id: "a", score: 7})
	if err != nil {
		t.Fatalf("Encode() error: %v", err)
	}
	if _, err = boolean.Position(cursor); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Position() error = %v, want code %s", err, errors.Invalid)
	}
}

func TestCutPropagatesEncodeFailure(t *testing.T) {
	t.Parallel()

	broken := paging.Keyset[row]{
		IDColumn: "id",
		ID:       func(r row) string { return r.id },
		Keys:     []paging.SortKey[row]{paging.IntKey[row]("score", "score", func(row) any { return struct{}{} })},
	}
	working := rowKeyset(false)
	cursor := mustCursor(t, working, row{id: "a", createdAt: time.Now()})
	rows := []row{{id: "a"}, {id: "b"}}

	cases := []struct {
		name    string
		rows    []row
		request paging.Request
	}{
		{name: "forward next", rows: rows, request: paging.Request{Limit: 1}},
		{name: "forward prev", rows: rows[:1], request: paging.Request{After: cursor, Limit: 2}},
		{name: "backward next", rows: rows[:1], request: paging.Request{Before: cursor, Limit: 2}},
		{name: "backward prev", rows: rows, request: paging.Request{Before: cursor, Limit: 1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := broken.Cut(tc.rows, tc.request); !errors.IsCode(err, errors.Internal) {
				t.Fatalf("Cut() error = %v, want code %s", err, errors.Internal)
			}
		})
	}
}
