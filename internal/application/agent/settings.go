package agent

import (
	"time"

	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	DefaultLoopLimit          = 12
	DefaultHistoryBudgetChars = 48000
	DefaultMaxToolResultBytes = 16384

	MinHistoryToolResultBytes = 512
	MaxHistoryToolResultBytes = 65536

	historyToolResultShare = 4
)

func HistoryToolResultBytes(maxToolResult int) int {
	if maxToolResult <= 0 {
		maxToolResult = DefaultMaxToolResultBytes
	}
	return min(max(maxToolResult/historyToolResultShare, MinHistoryToolResultBytes), MaxHistoryToolResultBytes)
}

var (
	loopLimitSetting     = settings.Int("agent.loopLimit", DefaultLoopLimit, settings.IntRange(1, 64))
	historyBudgetSetting = settings.Int("agent.historyBudgetChars", DefaultHistoryBudgetChars, settings.IntRange(2000, 400000))
	toolResultSetting    = settings.Int("agent.maxToolResultBytes", DefaultMaxToolResultBytes, settings.IntRange(1024, 262144))
	turnTimeoutSetting   = settings.Duration("agent.turnTimeout", DefaultTurnTimeout, settings.DurationRange(time.Minute, 2*time.Hour))
)

func TurnTimeout(values *settings.Values) time.Duration {
	return turnTimeoutSetting.Get(values)
}

func LoopLimit(values *settings.Values) int {
	return loopLimitSetting.Get(values)
}

func HistoryBudgetChars(values *settings.Values) int {
	return historyBudgetSetting.Get(values)
}

func MaxToolResultBytes(values *settings.Values) int {
	return toolResultSetting.Get(values)
}
