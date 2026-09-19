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
	StepPublish         StepName = "publish"
	StepRelinkNeighbors StepName = "relink_neighbors"
	StepSyncBack        StepName = "sync_back"
	StepReport          StepName = "report"
	StepSyncSite        StepName = "sync_site"
)

var stepNames = []StepName{
	StepResolveContext, StepGenerateBody, StepGenerateMeta, StepInsertLinks, StepRepairLinks,
	StepGenerateImages, StepValidate, StepJudge, StepPublish, StepRelinkNeighbors, StepSyncBack,
	StepReport, StepSyncSite,
}

func StepNames() []StepName {
	return slices.Clone(stepNames)
}
