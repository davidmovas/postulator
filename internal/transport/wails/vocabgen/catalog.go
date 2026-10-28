package main

import (
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
)

type derivation struct {
	keep   func(value string) bool
	export string
}

type vocabulary struct {
	export      string
	tsType      string
	pkg         string
	typeName    string
	aliasExport string
	aliasType   string
	derived     []derivation
}

func catalog() []vocabulary {
	return []vocabulary{
		{export: "siteStatuses", tsType: "SiteStatus", pkg: "internal/domain/site", typeName: "Status"},
		{export: "siteReaches", tsType: "SiteReach", pkg: "internal/domain/site", typeName: "Reach"},
		{export: "siteCommerces", tsType: "SiteCommerce", pkg: "internal/domain/site", typeName: "Commerce"},
		{export: "urlForms", tsType: "UrlForm", pkg: "internal/domain/site", typeName: "URLForm"},
		{export: "pageStatuses", tsType: "PageStatus", pkg: "internal/domain/pagemap", typeName: "Status"},
		{export: "pageWpTypes", tsType: "PageWpType", pkg: "internal/domain/pagemap", typeName: "WPType"},
		{export: "linkTypes", tsType: "LinkType", pkg: "internal/domain/pagemap", typeName: "LinkType"},
		{export: "linkSources", tsType: "LinkSource", pkg: "internal/domain/pagemap", typeName: "LinkSource"},
		{export: "parentReasons", tsType: "ParentReason", pkg: "internal/domain/pagemap", typeName: "ParentReason"},
		{export: "parentMethods", tsType: "ParentMethod", pkg: "internal/domain/pagemap", typeName: "ParentMethod"},
		{export: "trailSources", tsType: "TrailSource", pkg: "internal/domain/pagemap", typeName: "TrailSource"},
		{export: "technicalReasons", tsType: "TechnicalReason", pkg: "internal/domain/pagemap", typeName: "TechnicalReason"},
		{export: "contentRootStates", tsType: "ContentRootState", pkg: "internal/domain/render", typeName: "RootState"},
		{export: "regionKinds", tsType: "RegionKind", pkg: "internal/domain/render", typeName: "RegionKind"},
		{export: "findingCodes", tsType: "FindingCode", pkg: "internal/domain/finding", typeName: "Code"},
		{export: "findingSeverities", tsType: "FindingSeverity", pkg: "internal/domain/finding", typeName: "Severity"},
		{export: "findingStatuses", tsType: "FindingStatus", pkg: "internal/domain/finding", typeName: "Status"},
		{
			export: "findingOptions", tsType: "FindingOption", pkg: "internal/domain/finding", typeName: "Option",
			derived: []derivation{
				{export: "findingBulkOptions", keep: func(value string) bool { return finding.Option(value).Bulk() }},
			},
		},
		{export: "findingOptionKinds", tsType: "FindingOptionKind", pkg: "internal/domain/finding", typeName: "OptionKind"},
		{export: "matchRules", tsType: "MatchRule", pkg: "internal/domain/pagemap", typeName: "MatchRule"},
		{export: "bodyStates", tsType: "BodyState", pkg: "internal/domain/contextpack", typeName: "BodyState"},
		{export: "proposalClashes", tsType: "ProposalClash", pkg: "internal/domain/contextpack", typeName: "ProposalClash"},
		{export: "parentWhyReasons", tsType: "ParentWhyReason", pkg: "internal/domain/contextpack", typeName: "WhyReason"},
		{export: "adoptionSkips", tsType: "AdoptionSkip", pkg: "internal/domain/contextpack", typeName: "AdoptionSkip"},
		{export: "proposalSources", tsType: "ProposalSource", pkg: "internal/application/graph", typeName: "ProposalSource"},
		{export: "blockKinds", tsType: "BlockKind", pkg: "internal/domain/block", typeName: "BlockKind"},
		{export: "bodyKinds", tsType: "BodyKind", pkg: "internal/domain/block", typeName: "Kind"},
		{export: "blockOwners", tsType: "BlockOwner", pkg: "internal/domain/block", typeName: "Owner"},
		{export: "blockChanges", tsType: "BlockChange", pkg: "internal/domain/block", typeName: "ChangeKind"},
		{export: "previewKinds", tsType: "PreviewKind", pkg: "internal/application/pages", typeName: "PreviewKind"},
		{
			export: "cannibalizationReasons", tsType: "CannibalizationReason", pkg: "internal/domain/pagemap",
			typeName: "Reason",
		},
		{
			export: "pageTypePresets", tsType: "PageTypePreset", pkg: "internal/domain/pagetype", typeName: "Preset",
			derived: []derivation{
				{export: "creatablePageTypePresets", keep: func(value string) bool { return !pagetype.Preset(value).Legacy() }},
			},
		},
		{export: "pageTypeRoles", tsType: "PageTypeRole", pkg: "internal/domain/pagetype", typeName: "Role"},
		{export: "syncStages", tsType: "SyncStage", pkg: "internal/application/observe", typeName: "Stage"},
		{export: "ruleKinds", tsType: "RuleKind", pkg: "internal/domain/pagetype", typeName: "RuleKind"},
		{export: "urlModes", tsType: "URLMode", pkg: "internal/domain/pagetype", typeName: "URLMode"},
		{export: "schemaTypes", tsType: "SchemaType", pkg: "internal/domain/pagetype", typeName: "SchemaType"},
		{export: "ruleConditions", tsType: "RuleCondition", pkg: "internal/domain/pagetype", typeName: "ConditionKind"},
		{export: "ruleVerdicts", tsType: "RuleVerdict", pkg: "internal/domain/pagetype", typeName: "Verdict"},
		{export: "pageTypeSources", tsType: "PageTypeSource", pkg: "internal/domain/pagemap", typeName: "TypeSource"},
		{export: "pageTypeReasons", tsType: "PageTypeReason", pkg: "internal/domain/pagemap", typeName: "TypeReason"},
		{
			export: "fieldKeys", tsType: "FieldKey", pkg: "internal/domain/field", typeName: "Key",
			derived: []derivation{
				{export: "scopableFieldKeys", keep: func(value string) bool { return field.CheckScope(field.Key(value)) == nil }},
				{export: "packFieldKeys", keep: packable},
			},
		},
		{export: "treeShapes", tsType: "TreeShape", pkg: "internal/application/sitemodel", typeName: "TreeShape"},
		{export: "linksShapes", tsType: "LinksShape", pkg: "internal/application/sitemodel", typeName: "LinksShape"},
		{export: "siteGaps", tsType: "SiteGap", pkg: "internal/domain/contextpack", typeName: "Gap"},
		{export: "fieldProviders", tsType: "FieldProvider", pkg: "internal/domain/field", typeName: "Provider"},
		{export: "edgeKinds", tsType: "EdgeKind", pkg: "internal/domain/graph", typeName: "EdgeKind"},
		{export: "edgeStatuses", tsType: "EdgeStatus", pkg: "internal/domain/graph", typeName: "EdgeStatus"},
		{export: "sources", tsType: "Source", pkg: "internal/domain/graph", typeName: "Source"},
		{export: "anchorSources", tsType: "AnchorSource", pkg: "internal/domain/graph", typeName: "AnchorSource"},
		{export: "linkRelations", tsType: "LinkRelation", pkg: "internal/domain/content", typeName: "Relation"},
		{
			export: "linkClasses", tsType: "LinkClass", pkg: "internal/domain/content", typeName: "LinkClass",
			derived: []derivation{
				{export: "offGraphLinkClasses", keep: func(value string) bool { return content.LinkClass(value).OffGraph() }},
			},
		},
		{
			export: "linkBlockedReasons", tsType: "LinkBlockedReason", pkg: "internal/domain/content",
			typeName: "BlockedReason",
		},
		{
			export: "linkAuditSkipReasons", tsType: "LinkAuditSkipReason", pkg: "internal/application/reports",
			typeName: "SkipReason",
		},
		{export: "linkStates", tsType: "LinkState", pkg: "internal/application/reports", typeName: "LinkState"},
		{
			export: "runKinds", tsType: "RunKind", pkg: "internal/domain/run", typeName: "Kind",
			derived: []derivation{
				{
					export: "runKindsWithTheirOwnRecipe",
					keep: func(value string) bool {
						_, owns := run.Kind(value).Recipe()
						return owns
					},
				},
				{export: "startableRunKinds", keep: func(value string) bool { return run.Kind(value).Startable() }},
			},
		},
		{
			export: "runStatuses", tsType: "RunStatus", pkg: "internal/domain/run", typeName: "Status",
			derived: []derivation{
				{export: "activeRunStatuses", keep: func(value string) bool { return run.Status(value).Active() }},
				{export: "terminalRunStatuses", keep: func(value string) bool { return run.Status(value).Terminal() }},
			},
		},
		{export: "itemStatuses", tsType: "ItemStatus", aliasExport: "runStatuses", aliasType: "RunStatus"},
		{export: "pauseReasons", tsType: "PauseReason", pkg: "internal/domain/run", typeName: "PauseReason"},
		{export: "publishModes", tsType: "PublishMode", pkg: "internal/domain/run", typeName: "PublishMode"},
		{export: "actors", tsType: "Actor", pkg: "internal/kernel/ctx", typeName: "Actor"},
		{
			export: "artifactKinds", tsType: "ArtifactKind", pkg: "internal/domain/run", typeName: "ArtifactKind",
			derived: []derivation{
				{
					export: "purgeableArtifactKinds",
					keep:   func(value string) bool { return run.ArtifactKind(value).Purgeable() },
				},
			},
		},
		{export: "stepNames", tsType: "StepName", pkg: "internal/domain/run", typeName: "StepName"},
		{
			export: "perKindStepNames", tsType: "PerKindStepName", pkg: "internal/domain/run",
			typeName: "PerKindStepName",
		},
		{
			export: "retryBlockedReasons", tsType: "RetryBlockedReason", pkg: "internal/domain/run",
			typeName: "RetryBlockedReason",
		},
		{export: "templateScopes", tsType: "TemplateScope", pkg: "internal/domain/template", typeName: "Scope"},
		{export: "overrideScopes", tsType: "OverrideScope", pkg: "internal/domain/template", typeName: "OverrideScope"},
		{
			export: "anchorStrategies", tsType: "AnchorStrategy", pkg: "internal/domain/field",
			typeName: "AnchorStrategy",
		},
		{export: "imageSources", tsType: "ImageSource", pkg: "internal/domain/template", typeName: "ImageSource"},
		{export: "modelRoles", tsType: "ModelRole", pkg: "internal/domain/llm", typeName: "Role"},
		{
			export: "reasoningEfforts", tsType: "ReasoningEffort", pkg: "internal/domain/llm",
			typeName: "ReasoningEffort",
		},
		{export: "serviceTiers", tsType: "ServiceTier", pkg: "internal/domain/llm", typeName: "ServiceTier"},
		{export: "spendPurposes", tsType: "SpendPurpose", pkg: "internal/domain/llm", typeName: "Purpose"},
		{export: "conversationModes", tsType: "ConversationMode", pkg: "internal/domain/agent", typeName: "Mode"},
		{export: "messageRoles", tsType: "MessageRole", pkg: "internal/domain/agent", typeName: "Role"},
		{
			export: "pendingActionStatuses", tsType: "PendingActionStatus", pkg: "internal/domain/agent",
			typeName: "ActionStatus",
		},
		{
			export: "toolRisks", tsType: "ToolRisk", pkg: "internal/application/tools", typeName: "Risk",
			derived: []derivation{
				{
					export: "toolRisksNeedingConfirmation",
					keep:   func(value string) bool { return tools.Risk(value).NeedsConfirmation() },
				},
			},
		},
		{export: "toolCallStatuses", tsType: "ToolCallStatus", pkg: "internal/domain/agent", typeName: "CallStatus"},
		{export: "planStatuses", tsType: "PlanStatus", pkg: "internal/domain/agent", typeName: "PlanStatus"},
		{export: "planStepStatuses", tsType: "PlanStepStatus", pkg: "internal/domain/agent", typeName: "StepStatus"},
		{export: "importFields", tsType: "ImportField", pkg: "internal/domain/importmap", typeName: "Field"},
		{export: "importColumnUses", tsType: "ImportColumnUse", pkg: "internal/domain/importmap", typeName: "Use"},
		{export: "importRowTypes", tsType: "ImportRowType", pkg: "internal/domain/importmap", typeName: "RowType"},
		{
			export: "importFindingCodes", tsType: "ImportFindingCode", pkg: "internal/application/imports",
			typeName: "FindingCode",
			derived: []derivation{
				{
					export: "blockingImportFindingCodes",
					keep:   func(value string) bool { return imports.FindingCode(value).Blocking() },
				},
			},
		},
		{export: "importActions", tsType: "ImportAction", pkg: "internal/application/imports", typeName: "Action"},
		{export: "exportFormats", tsType: "ExportFormat", pkg: "internal/domain/importmap", typeName: "Format"},
		{export: "settingTypes", tsType: "SettingType", pkg: "internal/kernel/settings", typeName: "Type"},
		{export: "settingGroups", tsType: "SettingGroup", pkg: "internal/kernel/settings", typeName: "Group"},
		{export: "entitySortFields", tsType: "EntitySortField", pkg: "internal/domain/graph", typeName: "EntitySort"},
		{export: "pageSortFields", tsType: "PageSortField", pkg: "internal/domain/pagemap", typeName: "Sort"},
		{export: "runSortFields", tsType: "RunSortField", pkg: "internal/domain/run", typeName: "Sort"},
		{export: "siteSortFields", tsType: "SiteSortField", pkg: "internal/domain/site", typeName: "Sort"},
		{export: "templateSortFields", tsType: "TemplateSortField", pkg: "internal/domain/template", typeName: "Sort"},
	}
}

func packable(value string) bool {
	spec, known := field.Lookup(field.Key(value))
	return known && spec.Text && spec.AIWritable() && field.CheckScope(field.Key(value)) == nil
}
