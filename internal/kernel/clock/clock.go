package clock

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time {
	return time.Now().UTC()
}

type Fake struct {
	now time.Time
	mu  sync.RWMutex
}

func NewFake(now time.Time) *Fake {
	return &Fake{now: now.UTC()}
}

func (f *Fake) Now() time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.now
}

func (f *Fake) Set(now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = now.UTC()
}

func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}
