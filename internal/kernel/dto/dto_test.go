package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func TestTimeMarshal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{name: "utc", in: time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC), want: `"2026-09-17T10:30:00Z"`},
		{name: "offset is normalised", in: time.Date(2026, 9, 17, 11, 30, 0, 0, time.FixedZone("CET", 3600)), want: `"2026-09-17T10:30:00Z"`},
		{name: "sub-second is truncated", in: time.Date(2026, 9, 17, 10, 30, 0, 500_000_000, time.UTC), want: `"2026-09-17T10:30:00Z"`},
		{name: "zero", in: time.Time{}, want: `null`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(dto.NewTime(tc.in))
			if err != nil {
				t.Fatalf("Marshal() error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestTimeUnmarshal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want time.Time
		fail bool
	}{
		{name: "utc", in: `"2026-09-17T10:30:00Z"`, want: time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)},
		{name: "offset", in: `"2026-09-17T11:30:00+01:00"`, want: time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC)},
		{name: "null", in: `null`, want: time.Time{}},
		{name: "empty string", in: `""`, want: time.Time{}},
		{name: "not a timestamp", in: `"yesterday"`, fail: true},
		{name: "not a string", in: `17`, fail: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var got dto.Time
			err := json.Unmarshal([]byte(tc.in), &got)
			if tc.fail {
				if !errors.IsCode(err, errors.Invalid) {
					t.Fatalf("Unmarshal() error = %v, want code %s", err, errors.Invalid)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal() error: %v", err)
			}
			if !got.Std().Equal(tc.want) {
				t.Fatalf("Std() = %v, want %v", got.Std(), tc.want)
			}
			if !got.Std().IsZero() && got.Std().Location() != time.UTC {
				t.Fatalf("Std() location = %v, want UTC", got.Std().Location())
			}
		})
	}
}

func TestTimeRoundTrip(t *testing.T) {
	t.Parallel()

	original := dto.NewTime(time.Date(2026, 9, 17, 10, 30, 0, 0, time.UTC))
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	var decoded dto.Time
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if !decoded.Std().Equal(original.Std()) {
		t.Fatalf("round trip = %v, want %v", decoded.Std(), original.Std())
	}
	if decoded.String() != "2026-09-17T10:30:00Z" {
		t.Fatalf("String() = %q", decoded.String())
	}
}

func TestListRequestNormalize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   dto.ListRequest
		want int
	}{
		{name: "zero falls back to the default", in: dto.ListRequest{}, want: dto.DefaultLimit},
		{name: "negative falls back to the default", in: dto.ListRequest{Limit: -1}, want: dto.DefaultLimit},
		{name: "within range is kept", in: dto.ListRequest{Limit: 10}, want: 10},
		{name: "above the maximum is clamped", in: dto.ListRequest{Limit: 10_000}, want: dto.MaxLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.in.Normalize().Limit; got != tc.want {
				t.Fatalf("Normalize().Limit = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestListRequestLimitsMatchPaging(t *testing.T) {
	t.Parallel()

	if dto.DefaultLimit != paging.DefaultLimit {
		t.Fatalf("dto.DefaultLimit = %d, paging.DefaultLimit = %d", dto.DefaultLimit, paging.DefaultLimit)
	}
	if dto.MaxLimit != paging.MaxLimit {
		t.Fatalf("dto.MaxLimit = %d, paging.MaxLimit = %d", dto.MaxLimit, paging.MaxLimit)
	}
}

func TestListRequestJSON(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(dto.ListRequest{Cursor: "c", Limit: 25, Sort: &dto.Sort{Field: "createdAt", Desc: true}})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	const want = `{"cursor":"c","limit":25,"sort":{"field":"createdAt","desc":true}}`
	if string(encoded) != want {
		t.Fatalf("Marshal() = %s, want %s", encoded, want)
	}

	var decoded dto.ListRequest
	if err = json.Unmarshal([]byte(want), &decoded); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if decoded.Sort == nil || decoded.Sort.Field != "createdAt" || !decoded.Sort.Desc {
		t.Fatalf("Sort = %+v", decoded.Sort)
	}
}

func TestListRequestOmitsAbsentSort(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(dto.ListRequest{})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	if string(encoded) != `{"limit":0}` {
		t.Fatalf("Marshal() = %s", encoded)
	}
}
