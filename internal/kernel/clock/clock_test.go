package clock_test

import (
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/clock"
)

func TestSystemNowIsUTC(t *testing.T) {
	t.Parallel()

	before := time.Now().UTC()
	got := clock.System{}.Now()
	after := time.Now().UTC()

	if got.Location() != time.UTC {
		t.Fatalf("Now() location = %v, want UTC", got.Location())
	}
	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %v, want between %v and %v", got, before, after)
	}
}

func TestFake(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	fake := clock.NewFake(start)

	if got := fake.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}

	fake.Advance(90 * time.Minute)
	if got, want := fake.Now(), start.Add(90*time.Minute); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}

	reset := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	fake.Set(reset)
	if got := fake.Now(); !got.Equal(reset) {
		t.Fatalf("Now() = %v, want %v", got, reset)
	}
}

func TestFakeNormalisesToUTC(t *testing.T) {
	t.Parallel()

	local := time.Date(2026, 9, 17, 12, 0, 0, 0, time.FixedZone("CET", 3600))
	fake := clock.NewFake(local)

	if fake.Now().Location() != time.UTC {
		t.Fatalf("Now() location = %v, want UTC", fake.Now().Location())
	}
	if !fake.Now().Equal(local) {
		t.Fatalf("Now() = %v, want the same instant as %v", fake.Now(), local)
	}
}

func TestFakeIsConcurrencySafe(t *testing.T) {
	t.Parallel()

	fake := clock.NewFake(time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC))

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			fake.Advance(time.Second)
		}()
		go func() {
			defer wg.Done()
			_ = fake.Now()
		}()
	}
	wg.Wait()

	want := time.Date(2026, 9, 17, 12, 0, 32, 0, time.UTC)
	if got := fake.Now(); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
}

func TestFakeSatisfiesClock(t *testing.T) {
	t.Parallel()

	var c clock.Clock = clock.NewFake(time.Unix(0, 0))
	if c.Now().IsZero() {
		t.Fatal("Fake must satisfy Clock")
	}
	c = clock.System{}
	if c.Now().IsZero() {
		t.Fatal("System must satisfy Clock")
	}
}
