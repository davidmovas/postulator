package run

import "slices"

type StepName string

const (
	StepResolveContext  StepName = "resolve_context"
	StepGenerateBody    StepName = "generate_body"
	StepGenerateMeta    StepName = "generate_meta"
	StepInsertLinks     StepName = "insert_links"
	StepRepairLinks     StepName = "repair_links"
	StepGenerateImages  StepName = "generate_images"
	StepValidate        StepName = "validate"
	StepJudge           StepName = "judge"
	StepRepairHierarchy StepName = "repair_hierarchy"
	StepPublish         StepName = "publish"
	StepRelinkNeighbors StepName = "relink_neighbors"
	StepSyncBack        StepName = "sync_back"
	StepReport          StepName = "report"
	StepSyncSite        StepName = "sync_site"
)

const (
	RevertStep     = "revert"
	RelinkPageStep = "relink_page"
)

var stepNames = []StepName{
	StepResolveContext, StepGenerateBody, StepGenerateMeta, StepInsertLinks, StepRepairLinks,
	StepGenerateImages, StepValidate, StepJudge, StepRepairHierarchy, StepPublish, StepRelinkNeighbors, StepSyncBack,
	StepReport, StepSyncSite,
}

var perKindSteps = []string{
	string(StepRepairHierarchy), string(StepSyncSite), RelinkPageStep, RevertStep,
}

func StepNames() []StepName {
	return slices.Clone(stepNames)
}

func PerKindStep(name string) bool {
	return slices.Contains(perKindSteps, name)
}

func TemplateStepNames() []StepName {
	out := make([]StepName, 0, len(stepNames))
	for _, name := range stepNames {
		if !PerKindStep(string(name)) {
			out = append(out, name)
		}
	}
	return out
}
