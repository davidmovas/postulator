package tuning

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type Policy struct {
	values *settings.Values
}

func NewPolicy(values *settings.Values) *Policy {
	return &Policy{values: values}
}

func (p *Policy) Effort(role llm.Role) llm.ReasoningEffort {
	setting, tuned := effortSettings[role]
	if !tuned {
		return ""
	}
	return llm.ReasoningEffort(setting.Get(p.values))
}

func (p *Policy) Tier(role llm.Role) llm.ServiceTier {
	setting, tuned := tierSettings[role]
	if !tuned {
		return ""
	}
	if setting.Get(p.values) == tierFlex {
		return llm.TierFlex
	}
	return llm.TierDefault
}

func (p *Policy) FlexPatience() time.Duration {
	return flexPatienceSetting.Get(p.values)
}
