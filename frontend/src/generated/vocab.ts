export const siteStatuses = ["active", "paused", "error"] as const;
export type SiteStatus = (typeof siteStatuses)[number];

export const siteReaches = ["ok", "upgradeRequired", "unauthorized", "unreachable"] as const;
export type SiteReach = (typeof siteReaches)[number];

export const siteCommerces = ["", "absent", "forbidden", "ready"] as const;
export type SiteCommerce = (typeof siteCommerces)[number];

export const urlForms = ["auto", "absolute", "relative"] as const;
export type UrlForm = (typeof urlForms)[number];

export const pageStatuses = ["planned", "exists", "published", "archived"] as const;
export type PageStatus = (typeof pageStatuses)[number];

export const pageWpTypes = ["page", "post", "product", "product_cat"] as const;
export type PageWpType = (typeof pageWpTypes)[number];

export const linkTypes = ["breadcrumb", "listing", "navigation", "contextual", "unknown"] as const;
export type LinkType = (typeof linkTypes)[number];

export const linkSources = ["raw", "render"] as const;
export type LinkSource = (typeof linkSources)[number];

export const parentReasons = [
    "jsonld_breadcrumb",
    "aria_breadcrumb",
    "wp_parent",
    "url",
    "workbook",
    "none",
] as const;
export type ParentReason = (typeof parentReasons)[number];

export const parentMethods = ["href", "redirect", "name", ""] as const;
export type ParentMethod = (typeof parentMethods)[number];

export const trailSources = ["jsonld", "nav"] as const;
export type TrailSource = (typeof trailSources)[number];

export const technicalReasons = [
    "posts_page",
    "privacy",
    "shop",
    "cart",
    "checkout",
    "account",
    "terms",
    "shortcode",
    "noindex",
    "workbook",
    "manual",
    "",
] as const;
export type TechnicalReason = (typeof technicalReasons)[number];

export const contentRootStates = ["found", "empty_raw", "dynamic_only", "not_found", "archive"] as const;
export type ContentRootState = (typeof contentRootStates)[number];

export const regionKinds = [
    "breadcrumb",
    "listing",
    "navigation",
    "content",
    "content_dynamic",
    "generated",
    "unknown",
] as const;
export type RegionKind = (typeof regionKinds)[number];

export const findingCodes = [
    "parent_unresolved",
    "parent_ambiguous",
    "parent_cycle",
    "parent_name_twin",
    "parent_differs",
    "plugin_outdated",
    "field_changed_on_site",
    "body_changed_on_site",
    "render_unreachable",
    "render_redirected",
    "builder_detected",
    "technical_detected",
    "page_title_differs",
    "page_h1_differs",
    "url_form_mixed",
    "entity_card_proposed",
    "row_ambiguous",
    "row_without_page",
    "page_without_entity",
    "type_by_default",
    "entity_parent_differs",
] as const;
export type FindingCode = (typeof findingCodes)[number];

export const findingSeverities = ["info", "warning", "error"] as const;
export type FindingSeverity = (typeof findingSeverities)[number];

export const findingStatuses = ["open", "resolved", "dismissed"] as const;
export type FindingStatus = (typeof findingStatuses)[number];

export const findingOptions = [
    "ignore",
    "accept",
    "accept_site",
    "accept_workbook",
    "map_to_page",
    "set_type",
    "rewrite_field",
    "relink_page",
    "rescan",
    "propose_entity",
    "adopt_tree",
    "keep_graph",
] as const;
export type FindingOption = (typeof findingOptions)[number];

export const findingBulkOptions: readonly FindingOption[] = [
    "ignore",
    "accept_site",
    "accept_workbook",
    "set_type",
    "propose_entity",
    "keep_graph",
];

export const findingOptionKinds = ["model", "run", "proposal"] as const;
export type FindingOptionKind = (typeof findingOptionKinds)[number];

export const matchRules = ["mapped", "url", "slug", "entity", "title", "h1", "sku"] as const;
export type MatchRule = (typeof matchRules)[number];

export const bodyStates = ["empty", "client", "ours", "mixed", "dynamic"] as const;
export type BodyState = (typeof bodyStates)[number];

export const proposalClashes = ["", "no_name", "name_in_batch", "name_carried"] as const;
export type ProposalClash = (typeof proposalClashes)[number];

export const proposalSources = ["model", "site"] as const;
export type ProposalSource = (typeof proposalSources)[number];

export const blockKinds = [
    "paragraph",
    "heading",
    "list",
    "quote",
    "html",
    "shortcode",
    "wp",
    "foreign",
] as const;
export type BlockKind = (typeof blockKinds)[number];

export const bodyKinds = ["classic", "blocks"] as const;
export type BodyKind = (typeof bodyKinds)[number];

export const blockOwners = ["ours", "client", "site"] as const;
export type BlockOwner = (typeof blockOwners)[number];

export const blockChanges = ["kept", "added", "removed"] as const;
export type BlockChange = (typeof blockChanges)[number];

export const previewKinds = ["public", "preview"] as const;
export type PreviewKind = (typeof previewKinds)[number];

export const cannibalizationReasons = [
    "same_primary_keyword",
    "same_entity_canonical",
    "path_conflict",
] as const;
export type CannibalizationReason = (typeof cannibalizationReasons)[number];

export const pageTypePresets = [
    "landing",
    "category",
    "guide",
    "info",
    "product",
    "store_category",
    "catalog",
    "technical",
    "hub",
    "entity",
] as const;
export type PageTypePreset = (typeof pageTypePresets)[number];

export const pageTypeRoles = ["hub", "entity", "product", "archive", "catalog", "technical"] as const;
export type PageTypeRole = (typeof pageTypeRoles)[number];

export const ruleKinds = ["rule", "fallback"] as const;
export type RuleKind = (typeof ruleKinds)[number];

export const urlModes = ["prefix", "segment", "glob"] as const;
export type URLMode = (typeof urlModes)[number];

export const schemaTypes = [
    "Article",
    "BlogPosting",
    "HowTo",
    "FAQPage",
    "AboutPage",
    "ContactPage",
    "CollectionPage",
    "ItemList",
] as const;
export type SchemaType = (typeof schemaTypes)[number];

export const ruleConditions = [
    "postType",
    "url",
    "slug",
    "ancestor",
    "level",
    "listing",
    "schema",
    "words",
    "h2",
    "technical",
    "frontPage",
    "shop",
] as const;
export type RuleCondition = (typeof ruleConditions)[number];

export const ruleVerdicts = ["met", "unmet", "waiting", "blocked"] as const;
export type RuleVerdict = (typeof ruleVerdicts)[number];

export const pageTypeSources = ["", "auto", "manual"] as const;
export type PageTypeSource = (typeof pageTypeSources)[number];

export const pageTypeReasons = [
    "",
    "product",
    "store_category",
    "shop",
    "technical",
    "listing",
    "default",
    "kind",
    "user",
    "ai",
    "import",
    "rule",
    "unmatched",
] as const;
export type PageTypeReason = (typeof pageTypeReasons)[number];

export const fieldKeys = [
    "title",
    "h1",
    "excerpt",
    "body",
    "seo_title",
    "seo_description",
    "seo_canonical",
    "product_description",
    "product_short_description",
    "variation_description",
    "featured_image",
    "image_alt",
    "image_title",
    "upsells",
    "cross_sells",
    "meta",
    "slug",
    "status",
    "parent",
    "price",
    "regular_price",
    "sale_price",
    "sku",
    "stock_status",
    "stock_quantity",
    "manage_stock",
    "attributes",
    "variation",
    "terms",
] as const;
export type FieldKey = (typeof fieldKeys)[number];

export const scopableFieldKeys: readonly FieldKey[] = [
    "title",
    "excerpt",
    "body",
    "seo_title",
    "seo_description",
    "seo_canonical",
    "product_description",
    "product_short_description",
    "variation_description",
    "image_alt",
    "image_title",
    "upsells",
    "cross_sells",
];

export const packFieldKeys: readonly FieldKey[] = [
    "title",
    "excerpt",
    "body",
    "seo_title",
    "seo_description",
    "product_description",
    "product_short_description",
    "variation_description",
    "image_alt",
    "image_title",
];

export const exploreViews = ["tree", "links", "gaps", "pack"] as const;
export type ExploreView = (typeof exploreViews)[number];

export const siteGaps = ["planned", "empty", "unmapped", "orphan", "missing_links", "findings"] as const;
export type SiteGap = (typeof siteGaps)[number];

export const fieldProviders = ["", "plugin", "store"] as const;
export type FieldProvider = (typeof fieldProviders)[number];

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

export const linkBlockedReasons = ["no_canonical_page", "no_page"] as const;
export type LinkBlockedReason = (typeof linkBlockedReasons)[number];

export const linkAuditSkipReasons = ["unmapped"] as const;
export type LinkAuditSkipReason = (typeof linkAuditSkipReasons)[number];

export const linkStates = [
    "placed",
    "target_unpublished",
    "missing",
    "awaiting_target",
    "awaiting_page",
    "blocked",
] as const;
export type LinkState = (typeof linkStates)[number];

export const runKinds = [
    "generate",
    "relink",
    "audit",
    "sync",
    "import",
    "repair",
    "revert",
    "edit",
    "custom",
] as const;
export type RunKind = (typeof runKinds)[number];

export const runKindsWithTheirOwnRecipe: readonly RunKind[] = ["relink", "sync", "revert", "edit"];

export const startableRunKinds: readonly RunKind[] = [
    "generate",
    "relink",
    "audit",
    "sync",
    "import",
    "custom",
];

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

export const pauseReasons = [
    "budget_exceeded",
    "awaiting_confirmation",
    "needs_human",
    "user",
    "awaiting_parent",
    "awaiting_review",
] as const;
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
    "fields",
    "images",
    "validation_report",
    "judge_report",
    "publish_result",
    "relink_result",
    "sync_result",
    "final_report",
    "revert_result",
    "edit_plan",
] as const;
export type ArtifactKind = (typeof artifactKinds)[number];

export const purgeableArtifactKinds: readonly ArtifactKind[] = ["draft", "body_html", "images"];

export const stepNames = [
    "resolve_context",
    "generate_body",
    "generate_meta",
    "generate_fields",
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

export const perKindStepNames = ["sync_site", "relink_page", "revert", "edit_page"] as const;
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

export const serviceTiers = ["default", "flex"] as const;
export type ServiceTier = (typeof serviceTiers)[number];

export const spendPurposes = ["run", "chat", "title", "probe", "graph", "audit", "other"] as const;
export type SpendPurpose = (typeof spendPurposes)[number];

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
    "page_type",
    "parent_entity",
    "related",
    "meta_title",
    "meta_description",
    "wp_type",
    "own_entity",
    "sku",
] as const;
export type ImportField = (typeof importFields)[number];

export const importColumnUses = ["field", "level", "trail", "note", "indent", "ignored"] as const;
export type ImportColumnUse = (typeof importColumnUses)[number];

export const importRowTypes = ["pages", "products", "type"] as const;
export type ImportRowType = (typeof importRowTypes)[number];

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
    "unknown_page_type",
    "page_type_kept",
    "page_type_column_dropped",
    "unknown_wp_type",
    "root_page_skipped",
    "bad_volume",
    "unknown_own_entity",
    "technical_parent",
    "group_without_page",
    "ambiguous_parent",
    "ambiguous_entity",
    "row_ambiguous",
    "row_without_page",
    "parent_differs",
    "product_row_left",
    "wp_type_kept",
    "intermediate_level",
    "scope_clash",
] as const;
export type ImportFindingCode = (typeof importFindingCodes)[number];

export const blockingImportFindingCodes: readonly ImportFindingCode[] = [
    "bad_path",
    "unknown_parent",
    "unknown_related",
    "self_edge",
    "cycle",
    "ambiguous_parent",
    "ambiguous_entity",
    "scope_clash",
];

export const importActions = ["create", "update", "skip", "ambiguous"] as const;
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

export const pageSortFields = ["createdAt", "path", "title"] as const;
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
