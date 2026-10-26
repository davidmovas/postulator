package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type fitRow struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func rowsUpTo(limit int) paging.List[fitRow] {
	rows := make([]fitRow, 0, limit)
	for range limit {
		rows = append(rows, fitRow{ID: "row", Text: strings.Repeat("x", 90)})
	}
	return paging.List[fitRow]{Items: rows, HasMore: true, Cursors: paging.Cursors{Next: paging.Cursor("issued")}}
}

func TestAPageIsAskedForAgainSmallerUntilItFitsTheCap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		cap      int
		limit    int
		calls    int
		lastSize int
	}{
		{name: "fits at once", cap: 16384, limit: 50, calls: 1, lastSize: 50},
		{name: "halved twice", cap: 2048, limit: 50, calls: 3, lastSize: 12},
		{name: "the default limit when none is asked", cap: 16384, limit: 0, calls: 1, lastSize: defaultPageLimit},
		{name: "one row is the floor", cap: 64, limit: 8, calls: 4, lastSize: 1},
		{name: "no cap asks once", cap: 0, limit: 500, calls: 1, lastSize: 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := Deps{}
			if tc.cap > 0 {
				deps.ResultBytes = func() int { return tc.cap }
			}
			asked := make([]int, 0)
			answer, err := fitted(deps, tc.limit, func(limit int) (paging.List[fitRow], error) {
				asked = append(asked, limit)
				return rowsUpTo(limit), nil
			})
			if err != nil {
				t.Fatalf("fitted: %v", err)
			}
			if len(asked) != tc.calls || len(answer.Items) != tc.lastSize {
				t.Fatalf("asked %v and answered %d rows, want %d calls ending at %d", asked, len(answer.Items),
					tc.calls, tc.lastSize)
			}
			if answer.Next != "issued" {
				t.Fatalf("the cursor is not the one the shorter page was issued with: %q", answer.Next)
			}
			encoded, err := json.Marshal(answer)
			if err != nil {
				t.Fatalf("encode the page: %v", err)
			}
			if tc.cap > 0 && len(answer.Items) > 1 && len(encoded) > tc.cap-tc.cap/fitHeadroomDivisor {
				t.Fatalf("the page is %d bytes against a cap of %d", len(encoded), tc.cap)
			}
		})
	}
}
