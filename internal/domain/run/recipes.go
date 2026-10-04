package run

import "github.com/davidmovas/postulator/internal/domain/template"

func (k Kind) Recipe() ([]template.StepSpec, bool) {
	switch k {
	case KindRelink:
		return RelinkRecipe(), true
	case KindRepair:
		return RepairRecipe(), true
	case KindSync:
		return SyncRecipe(), true
	case KindRevert:
		return RevertRecipe(), true
	case KindGenerate, KindAudit, KindImport, KindCustom:
		return nil, false
	default:
		return nil, false
	}
}

func steps(names ...string) []template.StepSpec {
	out := make([]template.StepSpec, 0, len(names))
	for _, name := range names {
		out = append(out, template.StepSpec{Name: name, Enabled: true})
	}
	return out
}

func GenerateRecipe() []template.StepSpec {
	return steps(
		string(StepResolveContext), string(StepGenerateBody), string(StepGenerateMeta),
		string(StepInsertLinks), string(StepRepairLinks), string(StepGenerateImages), string(StepValidate),
		string(StepJudge), string(StepPublish), string(StepRelinkNeighbors), string(StepSyncBack), string(StepReport),
	)
}

func RelinkRecipe() []template.StepSpec {
	return steps(string(StepResolveContext), string(StepRelinkPage), string(StepSyncBack), string(StepReport))
}

func RepairRecipe() []template.StepSpec {
	return steps(string(StepRepairHierarchy), string(StepSyncBack), string(StepReport))
}

func RevertRecipe() []template.StepSpec {
	return steps(string(StepRevert))
}

func SyncRecipe() []template.StepSpec {
	return steps(string(StepSyncSite))
}
