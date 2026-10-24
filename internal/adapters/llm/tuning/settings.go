package tuning

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	tierStandard = "standard"
	tierFlex     = "flex"

	defaultFlexPatience = 8 * time.Minute
	leastFlexPatience   = 30 * time.Second
	mostFlexPatience    = 14 * time.Minute
)

var (
	efforts = []string{
		string(llm.EffortNone), string(llm.EffortLow), string(llm.EffortMedium), string(llm.EffortHigh), string(llm.EffortXHigh),
	}
	tiers = []string{tierStandard, tierFlex}

	effortSettings = map[llm.Role]*settings.Setting[string]{
		llm.RoleWriter: settings.Enum("llm.effort.writer", string(llm.EffortMedium), efforts),
		llm.RoleEditor: settings.Enum("llm.effort.editor", string(llm.EffortLow), efforts),
		llm.RoleLinker: settings.Enum("llm.effort.linker", string(llm.EffortLow), efforts),
		llm.RoleJudge:  settings.Enum("llm.effort.judge", string(llm.EffortLow), efforts),
		llm.RoleTitler: settings.Enum("llm.effort.titler", string(llm.EffortNone), efforts),
	}

	tierSettings = map[llm.Role]*settings.Setting[string]{
		llm.RoleWriter: settings.Enum("llm.tier.writer", tierFlex, tiers),
		llm.RoleEditor: settings.Enum("llm.tier.editor", tierStandard, tiers),
		llm.RoleLinker: settings.Enum("llm.tier.linker", tierStandard, tiers),
		llm.RoleJudge:  settings.Enum("llm.tier.judge", tierStandard, tiers),
		llm.RoleTitler: settings.Enum("llm.tier.titler", tierStandard, tiers),
	}

	flexPatienceSetting = settings.Duration("llm.flexPatience", defaultFlexPatience,
		settings.DurationRange(leastFlexPatience, mostFlexPatience))

	inputBudgetSettings = map[llm.Role]*settings.Setting[int]{
		llm.RoleWriter: settings.Int("llm.inputBudget.writer", 0, settings.IntMin(0)),
		llm.RoleEditor: settings.Int("llm.inputBudget.editor", 0, settings.IntMin(0)),
		llm.RoleLinker: settings.Int("llm.inputBudget.linker", 0, settings.IntMin(0)),
		llm.RoleJudge:  settings.Int("llm.inputBudget.judge", 0, settings.IntMin(0)),
	}
)
