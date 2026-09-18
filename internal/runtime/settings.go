package runtime

import (
	"time"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	DefaultWorkers       = 2
	DefaultPerSite       = 1
	DefaultSweepInterval = 5 * time.Second
	DefaultRetentionDays = 30
	DefaultStepTimeout   = 5 * time.Minute
	DefaultLeaseDuration = 10 * time.Minute
	DefaultRunDeadline   = 6 * time.Hour
	DefaultRetryBackoff  = time.Second
	maxRetryBackoff      = 5 * time.Minute
	sweepBatch           = 100
	dispatchBatch        = 100
	queueCapacity        = 256
)

var (
	workersSetting       = settings.Int("runs.workers", DefaultWorkers, settings.IntRange(1, 16))
	perSiteSetting       = settings.Int("runs.perSite", DefaultPerSite, settings.IntRange(1, 8))
	sweepSetting         = settings.Duration("runs.sweepInterval", DefaultSweepInterval, settings.DurationRange(time.Second, 5*time.Minute))
	retentionSetting     = settings.Int("runs.artifactRetentionDays", DefaultRetentionDays, settings.IntRange(1, 365))
	stepTimeoutSetting   = settings.Duration("runs.stepTimeout", DefaultStepTimeout, settings.DurationRange(10*time.Second, time.Hour))
	leaseDurationSetting = settings.Duration("runs.leaseDuration", DefaultLeaseDuration, settings.DurationRange(30*time.Second, 2*time.Hour))
)

type Config struct {
	Workers       int
	PerSite       int
	SweepInterval time.Duration
	RetentionDays int
	StepTimeout   time.Duration
	LeaseDuration time.Duration
	RunDeadline   time.Duration
}

func Settings(values *settings.Values) Config {
	return Config{
		Workers:       workersSetting.Get(values),
		PerSite:       perSiteSetting.Get(values),
		SweepInterval: sweepSetting.Get(values),
		RetentionDays: retentionSetting.Get(values),
		StepTimeout:   stepTimeoutSetting.Get(values),
		LeaseDuration: leaseDurationSetting.Get(values),
		RunDeadline:   DefaultRunDeadline,
	}
}

func (c Config) normalized() Config {
	if c.Workers <= 0 {
		c.Workers = DefaultWorkers
	}
	if c.PerSite <= 0 {
		c.PerSite = DefaultPerSite
	}
	if c.SweepInterval <= 0 {
		c.SweepInterval = DefaultSweepInterval
	}
	if c.RetentionDays <= 0 {
		c.RetentionDays = DefaultRetentionDays
	}
	if c.StepTimeout <= 0 {
		c.StepTimeout = DefaultStepTimeout
	}
	if c.LeaseDuration <= 0 {
		c.LeaseDuration = DefaultLeaseDuration
	}
	if c.RunDeadline <= 0 {
		c.RunDeadline = DefaultRunDeadline
	}
	return c
}
