package linking

const (
	DefaultLanguage = "English"

	// Concurrency calculation constants for RPM-based limiting
	RPMMultiplier  = 8   // RPM * 0.8 / 10 = RPM * 8 / 100
	RPMDivisor     = 100
	MinConcurrency = 1
	MaxConcurrency = 10

	// Batch processing
	MaxNodesPerBatch = 15
)
