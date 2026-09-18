package steps

import "github.com/davidmovas/postulator/internal/kernel/settings"

const DefaultBatchSize = 100

var batchSizeSetting = settings.Int("sync.batchSize", DefaultBatchSize, settings.IntRange(10, 500))

func BatchSize(values *settings.Values) int {
	return batchSizeSetting.Get(values)
}
