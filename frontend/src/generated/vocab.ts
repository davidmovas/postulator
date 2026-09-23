export const siteStatuses = ["active", "paused", "error"] as const;
export type SiteStatus = (typeof siteStatuses)[number];

export const siteReaches = ["ok", "upgradeRequired", "unauthorized", "unreachable"] as const;
export type SiteReach = (typeof siteReaches)[number];

export const pageStatuses = ["planned", "exists", "published", "archived"] as const;
export type PageStatus = (typeof pageStatuses)[number];

export const pageWpTypes = ["page", "post", "product", "product_cat"] as const;
export type PageWpType = (typeof pageWpTypes)[number];

export const linkOrigins = ["generated", "observed"] as const;
export type LinkOrigin = (typeof linkOrigins)[number];

export const previewKinds = ["public", "preview"] as const;
export type PreviewKind = (typeof previewKinds)[number];

export const cannibalizationReasons = [
    "same_primary_keyword",
    "same_entity_canonical",
    "path_conflict",
] as const;
export type CannibalizationReason = (typeof cannibalizationReasons)[number];

export const entityKinds = ["hub", "product", "topic", "category", "custom"] as const;
export type EntityKind = (typeof entityKinds)[number];

export const edgeKinds = ["parent", "related"] as const;
export type EdgeKind = (typeof edgeKinds)[number];

export const edgeStatuses = ["approved", "proposed", "rejected"] as const;
export type EdgeStatus = (typeof edgeStatuses)[number];

export const sources = ["import", "user", "ai"] as const;
export type Source = (typeof sources)[number];

export const anchorSources = ["user", "ai"] as const;
export type AnchorSource = (typeof anchorSources)[number];

export const linkRelations = ["up", "down", "sibling"] as const;
export type LinkRelation = (typeof linkRelations)[number];

export const linkClasses = ["graph", "self", "external", "unknown_internal"] as const;
export type LinkClass = (typeof linkClasses)[number];

export const offGraphLinkClasses: readonly LinkClass[] = ["self", "external", "unknown_internal"];

export const linkBlockedReasons = ["no_canonical_page"] as const;
export type LinkBlockedReason = (typeof linkBlockedReasons)[number];

export const linkAuditSkipReasons = ["unmapped", "no_template"] as const;
export type LinkAuditSkipReason = (typeof linkAuditSkipReasons)[number];

export const runKinds = [
    "generate",
    "relink",
    "audit",
    "sync",
    "import",
    "repair",
    "revert",
    "custom",
] as const;
export type RunKind = (typeof runKinds)[number];

export const runKindsWithTheirOwnRecipe: readonly RunKind[] = ["relink", "sync", "repair", "revert"];

export const runStatuses = [
    "pending",
    "running",
    "waiting",
    "paused",
    "completed",
    "failed",
    "cancelled",
] as const;
export type RunStatus = (typeof runStatuses)[number];

export const activeRunStatuses: readonly RunStatus[] = ["pending", "running", "waiting", "paused"];

export const terminalRunStatuses: readonly RunStatus[] = ["completed", "failed", "cancelled"];

export const itemStatuses = runStatuses;
export type ItemStatus = RunStatus;

export const pauseReasons = ["budget_exceeded", "awaiting_confirmation", "needs_human", "user"] as const;
export type PauseReason = (typeof pauseReasons)[number];

export const publishModes = ["draft", "publish"] as const;
export type PublishMode = (typeof publishModes)[number];

export const actors = ["user", "agent", "schedule"] as const;
export type Actor = (typeof actors)[number];

export const artifactKinds = [
    "link_context",
    "draft",
    "body_html",
    "meta",
    "images",
    "validation_report",
    "judge_report",
    "publish_result",
    "relink_result",
    "sync_result",
    "final_report",
    "revert_result",
] as const;
export type ArtifactKind = (typeof artifactKinds)[number];

export const purgeableArtifactKinds: readonly ArtifactKind[] = ["draft", "body_html", "images"];

export const stepNames = [
    "resolve_context",
    "generate_body",
    "generate_meta",
    "insert_links",
    "repair_links",
    "generate_images",
    "validate",
    "judge",
    "publish",
    "relink_neighbors",
    "sync_back",
    "report",
] as const;
export type StepName = (typeof stepNames)[number];

export const perKindStepNames = ["repair_hierarchy", "sync_site", "relink_page", "revert"] as const;
export type PerKindStepName = (typeof perKindStepNames)[number];

export const retryBlockedReasons = ["inputs_expired"] as const;
export type RetryBlockedReason = (typeof retryBlockedReasons)[number];

export const templateScopes = ["global", "site"] as const;
export type TemplateScope = (typeof templateScopes)[number];

export const overrideScopes = ["site", "page"] as const;
export type OverrideScope = (typeof overrideScopes)[number];

export const anchorStrategies = ["prefer_user", "rotate"] as const;
export type AnchorStrategy = (typeof anchorStrategies)[number];

export const imageSources = ["ai", "wpmedia", "local"] as const;
export type ImageSource = (typeof imageSources)[number];

export const modelRoles = ["writer", "editor", "linker", "judge", "chat", "image", "titler"] as const;
export type ModelRole = (typeof modelRoles)[number];

export const reasoningEfforts = ["none", "low", "medium", "high", "xhigh"] as const;
export type ReasoningEffort = (typeof reasoningEfforts)[number];

export const conversationModes = ["confirm", "autonomous"] as const;
export type ConversationMode = (typeof conversationModes)[number];

export const messageRoles = ["user", "assistant", "tool"] as const;
export type MessageRole = (typeof messageRoles)[number];

export const pendingActionStatuses = ["pending", "approved", "rejected", "executed", "failed"] as const;
export type PendingActionStatus = (typeof pendingActionStatuses)[number];

export const toolRisks = ["read", "write", "dangerous"] as const;
export type ToolRisk = (typeof toolRisks)[number];

export const toolRisksNeedingConfirmation: readonly ToolRisk[] = ["write", "dangerous"];

export const toolCallStatuses = ["ok", "denied", "error"] as const;
export type ToolCallStatus = (typeof toolCallStatuses)[number];

export const importFields = [
    "path",
    "title",
    "h1",
    "primary_keyword",
    "keywords",
    "anchors",
    "entity",
    "entity_kind",
    "parent_entity",
    "related",
    "page_kind",
    "meta_title",
    "meta_description",
    "wp_type",
] as const;
export type ImportField = (typeof importFields)[number];

export const importFindingCodes = [
    "bad_path",
    "no_target",
    "duplicate_path",
    "intermediate_path",
    "unknown_parent",
    "unknown_related",
    "self_edge",
    "cycle",
    "cannibalization",
    "unknown_entity_kind",
    "unknown_page_kind",
    "unknown_wp_type",
] as const;
export type ImportFindingCode = (typeof importFindingCodes)[number];

export const blockingImportFindingCodes: readonly ImportFindingCode[] = [
    "bad_path",
    "unknown_parent",
    "unknown_related",
    "self_edge",
    "cycle",
];

export const importActions = ["create", "update", "skip"] as const;
export type ImportAction = (typeof importActions)[number];

export const exportFormats = ["xlsx", "csv"] as const;
export type ExportFormat = (typeof exportFormats)[number];

export const settingTypes = ["bool", "int", "string", "enum", "duration"] as const;
export type SettingType = (typeof settingTypes)[number];

export const settingGroups = [
    "agent",
    "browser",
    "images",
    "import",
    "llm",
    "runs",
    "schedules",
    "sync",
    "wp",
] as const;
export type SettingGroup = (typeof settingGroups)[number];

export const entitySortFields = ["createdAt", "name"] as const;
export type EntitySortField = (typeof entitySortFields)[number];

export const pageSortFields = ["createdAt", "path"] as const;
export type PageSortField = (typeof pageSortFields)[number];

export const runSortFields = ["createdAt", "status"] as const;
export type RunSortField = (typeof runSortFields)[number];

export const siteSortFields = ["createdAt", "name"] as const;
export type SiteSortField = (typeof siteSortFields)[number];

export const templateSortFields = ["createdAt", "name"] as const;
export type TemplateSortField = (typeof templateSortFields)[number];

export function isOneOf<T extends string>(values: readonly T[], candidate: string): candidate is T {
    return (values as readonly string[]).includes(candidate);
}
