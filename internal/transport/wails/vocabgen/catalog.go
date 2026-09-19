package main

import (
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/content"
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
		{export: "pageStatuses", tsType: "PageStatus", pkg: "internal/domain/pagemap", typeName: "Status"},
		{export: "pageWpTypes", tsType: "PageWpType", pkg: "internal/domain/pagemap", typeName: "WPType"},
		{export: "linkOrigins", tsType: "LinkOrigin", pkg: "internal/domain/pagemap", typeName: "LinkOrigin"},
		{
			export: "cannibalizationReasons", tsType: "CannibalizationReason", pkg: "internal/domain/pagemap",
			typeName: "Reason",
		},
		{export: "entityKinds", tsType: "EntityKind", pkg: "internal/domain/graph", typeName: "Kind"},
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
		{export: "runKinds", tsType: "RunKind", pkg: "internal/domain/run", typeName: "Kind"},
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
			export: "retryBlockedReasons", tsType: "RetryBlockedReason", pkg: "internal/domain/run",
			typeName: "RetryBlockedReason",
		},
		{export: "templateScopes", tsType: "TemplateScope", pkg: "internal/domain/template", typeName: "Scope"},
		{export: "overrideScopes", tsType: "OverrideScope", pkg: "internal/domain/template", typeName: "OverrideScope"},
		{
			export: "anchorStrategies", tsType: "AnchorStrategy", pkg: "internal/domain/template",
			typeName: "AnchorStrategy",
		},
		{export: "imageSources", tsType: "ImageSource", pkg: "internal/domain/template", typeName: "ImageSource"},
		{export: "modelRoles", tsType: "ModelRole", pkg: "internal/domain/llm", typeName: "Role"},
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
		{export: "importFields", tsType: "ImportField", pkg: "internal/domain/importmap", typeName: "Field"},
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
		{export: "settingTypes", tsType: "SettingType", pkg: "internal/kernel/settings", typeName: "Type"},
		{export: "settingGroups", tsType: "SettingGroup", pkg: "internal/kernel/settings", typeName: "Group"},
		{export: "entitySortFields", tsType: "EntitySortField", pkg: "internal/domain/graph", typeName: "EntitySort"},
		{export: "pageSortFields", tsType: "PageSortField", pkg: "internal/domain/pagemap", typeName: "Sort"},
		{export: "runSortFields", tsType: "RunSortField", pkg: "internal/domain/run", typeName: "Sort"},
		{export: "siteSortFields", tsType: "SiteSortField", pkg: "internal/domain/site", typeName: "Sort"},
		{export: "templateSortFields", tsType: "TemplateSortField", pkg: "internal/domain/template", typeName: "Sort"},
	}
}
