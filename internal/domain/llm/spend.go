package llm

import (
	"slices"
	"time"
)

type Purpose string

const (
	PurposeRun   Purpose = "run"
	PurposeChat  Purpose = "chat"
	PurposeTitle Purpose = "title"
	PurposeProbe Purpose = "probe"
	PurposeGraph Purpose = "graph"
	PurposeAudit Purpose = "audit"
	PurposeOther Purpose = "other"
)

const (
	StepChat                = "chat"
	StepTitle               = "title"
	StepProbe               = "test_provider"
	StepJudge               = "judge"
	StepProposeFromPages    = "propose_from_pages"
	StepProposeFromKeywords = "propose_from_keywords"
	StepProposeRelated      = "propose_related"
)

type PurposeRule struct {
	Purpose Purpose
	Steps   []string
}

func PurposeRules() []PurposeRule {
	return []PurposeRule{
		{Purpose: PurposeChat, Steps: []string{StepChat}},
		{Purpose: PurposeTitle, Steps: []string{StepTitle}},
		{Purpose: PurposeProbe, Steps: []string{StepProbe}},
		{Purpose: PurposeGraph, Steps: []string{StepProposeFromPages, StepProposeFromKeywords, StepProposeRelated}},
		{Purpose: PurposeAudit, Steps: []string{StepJudge}},
	}
}

func PurposeOf(runID, step string) Purpose {
	if runID != "" {
		return PurposeRun
	}
	for _, rule := range PurposeRules() {
		if slices.Contains(rule.Steps, step) {
			return rule.Purpose
		}
	}
	return PurposeOther
}

type SpendQuery struct {
	Since time.Time
	RunID string
}

type SpendSlice struct {
	Purpose     Purpose
	Provider    string
	Model       string
	Tier        ServiceTier
	Step        string
	Calls       int
	Failed      int
	Input       int
	CachedInput int
	CacheWrite  int
	Output      int
	Reasoning   int
	USD         float64
}
