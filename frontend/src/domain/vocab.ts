export const siteStatuses = ["active", "paused", "error"] as const;
export type SiteStatus = (typeof siteStatuses)[number];

export const pageStatuses = ["planned", "exists", "published", "archived"] as const;
export type PageStatus = (typeof pageStatuses)[number];

export const pageWpTypes = ["page", "post", "product", "product_cat"] as const;
export type PageWpType = (typeof pageWpTypes)[number];

export const linkOrigins = ["generated", "observed"] as const;
export type LinkOrigin = (typeof linkOrigins)[number];

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

export const runKinds = ["generate", "relink", "audit", "sync", "import", "custom"] as const;
export type RunKind = (typeof runKinds)[number];

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
] as const;
export type ArtifactKind = (typeof artifactKinds)[number];

export const stepNames = [
    "resolve_context",
    "generate_body",
    "generate_meta",
    "generate_images",
    "insert_links",
    "validate",
    "judge",
    "publish",
    "sync_back",
    "relink_neighbors",
    "repair_links",
    "report",
    "sync_site",
] as const;
export type StepName = (typeof stepNames)[number];

export const templateScopes = ["global", "site"] as const;
export type TemplateScope = (typeof templateScopes)[number];

export const overrideScopes = ["site", "page"] as const;
export type OverrideScope = (typeof overrideScopes)[number];

export const anchorStrategies = ["prefer_user", "rotate"] as const;
export type AnchorStrategy = (typeof anchorStrategies)[number];

export const imageSources = ["ai", "wpmedia", "local"] as const;
export type ImageSource = (typeof imageSources)[number];

export const modelRoles = ["writer", "editor", "linker", "judge", "chat", "image"] as const;
export type ModelRole = (typeof modelRoles)[number];

export const conversationModes = ["confirm", "autonomous"] as const;
export type ConversationMode = (typeof conversationModes)[number];

export const messageRoles = ["user", "assistant", "tool"] as const;
export type MessageRole = (typeof messageRoles)[number];

export const pendingActionStatuses = ["pending", "approved", "rejected", "executed", "failed"] as const;
export type PendingActionStatus = (typeof pendingActionStatuses)[number];

export const toolRisks = ["read", "write", "dangerous"] as const;
export type ToolRisk = (typeof toolRisks)[number];

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

export const importActions = ["create", "update", "skip"] as const;
export type ImportAction = (typeof importActions)[number];

export const settingTypes = ["bool", "int", "string", "enum", "duration"] as const;
export type SettingType = (typeof settingTypes)[number];

export const settingGroups = [
    "agent",
    "images",
    "import",
    "llm",
    "runs",
    "schedules",
    "sync",
    "wp",
] as const;
export type SettingGroup = (typeof settingGroups)[number];

export function isOneOf<T extends string>(values: readonly T[], candidate: string): candidate is T {
    return (values as readonly string[]).includes(candidate);
}
