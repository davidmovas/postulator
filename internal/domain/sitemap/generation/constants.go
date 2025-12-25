package generation

import "time"

const (
	DefaultTaskTimeout    = 4 * time.Hour
	DefaultMaxConcurrency = 3
	DraftSlugPrefix       = "draft-%d"
)
