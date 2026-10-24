//go:build !race

package scale_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
)

const (
	smallBlocks = 50
	largeBlocks = 125
)

type measured struct {
	sqlitetest.Tally
	took time.Duration
}

func (s *scaled) once(t *testing.T, op string, fn func() error) measured {
	t.Helper()

	started := time.Now()
	tally, err := s.counter.Measure(fn)
	took := time.Since(started)
	if err != nil {
		t.Fatalf("%s at %d blocks: %v", op, s.blocks, err)
	}
	for _, sql := range s.counter.Distinct() {
		s.traced[sql] = struct{}{}
	}
	t.Logf("%-28s %5d pages: %6d statements %8d rows %8s", op, s.pages(), tally.Statements, tally.Rows, took.Round(time.Millisecond))
	return measured{Tally: tally, took: took}
}

func (s *scaled) measure(t *testing.T, op string, fn func() error) measured {
	t.Helper()

	if err := fn(); err != nil {
		t.Fatalf("%s at %d blocks: %v", op, s.blocks, err)
	}
	return s.once(t, op, fn)
}

func (s *scaled) pages() int {
	return len(s.plan.specs) + 2*s.blocks
}

func flat(t *testing.T, op string, small, large measured, rows bool) {
	t.Helper()

	if small.Statements != large.Statements {
		t.Errorf("%s runs %d statements at %d pages and %d at %d", op, small.Statements, smallBlocks*blockPages, large.Statements, largeBlocks*blockPages)
	}
	if rows && small.Rows != large.Rows {
		t.Errorf("%s reads %d rows at %d pages and %d at %d", op, small.Rows, smallBlocks*blockPages, large.Rows, largeBlocks*blockPages)
	}
}
