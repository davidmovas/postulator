package imports

import "github.com/davidmovas/postulator/internal/kernel/settings"

const DefaultMaxRows = 20000

var maxRowsSetting = settings.Int("import.maxRows", DefaultMaxRows, settings.IntRange(100, 200000))

func MaxRows(values *settings.Values) int {
	return maxRowsSetting.Get(values)
}
