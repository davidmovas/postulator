import type {
    AgentModels,
    CatalogModels,
    ContentModels,
    DtoModels,
    GraphModels,
    ImportModels,
    LlmModels,
    PagesModels,
    ReportsModels,
    RunDomainModels,
    RunsModels,
    SchedulesModels,
    SchemaModels,
    SettingsModels,
    SitesModels,
    SyncModels,
    TemplateDomainModels,
    TemplatesModels,
    TransportModels,
} from "../lib/api.js";
import type { Wire } from "./wire.js";

export type ListFilter<T> = Omit<T, "cursor" | "limit" | "sort">;

export type SortSpec = DtoModels.Sort;

export type Site = Wire<SitesModels.Site>;
export type SiteDefaults = Wire<SitesModels.Defaults>;
export type SitePlugin = Wire<SitesModels.Plugin>;
export type SiteFilter = ListFilter<SitesModels.ListRequest>;
export type Reachability = Wire<SitesModels.Reachability>;

export type Page = Wire<PagesModels.Page>;
export type PageLink = Wire<PagesModels.PageLink>;
export type PageTreeNode = Wire<PagesModels.TreeNode>;
export type PageLinkInput = Wire<PagesModels.LinkInput>;
export type PageFilter = ListFilter<PagesModels.ListRequest>;

export type Entity = Wire<GraphModels.Entity>;
export type Edge = Wire<GraphModels.Edge>;
export type Anchor = Wire<GraphModels.Anchor>;
export type EntityFilter = ListFilter<GraphModels.ListEntitiesRequest>;
export type EdgeFilter = ListFilter<GraphModels.ListEdgesRequest>;

export type Template = Wire<TemplatesModels.Template>;
export type TemplateOverride = Wire<TemplatesModels.Override>;
export type LinkPolicy = Wire<TemplatesModels.LinkPolicy>;
export type TemplateFilter = ListFilter<TemplatesModels.ListTemplatesRequest>;
export type PolicyFilter = ListFilter<TemplatesModels.ListPoliciesRequest>;
export type TemplateSpec = Wire<TemplateDomainModels.TemplateSpec>;
export type TemplateSection = Wire<TemplateDomainModels.Section>;
export type LinkRules = Wire<TemplateDomainModels.LinkRules>;
export type MetaRules = Wire<TemplateDomainModels.MetaRules>;
export type KeywordRules = Wire<TemplateDomainModels.KeywordRules>;
export type StepSpec = Wire<TemplateDomainModels.StepSpec>;

export type Run = Wire<RunsModels.Run>;
export type RunItem = Wire<RunsModels.Item>;
export type RunArtifact = Wire<RunsModels.Artifact>;
export type RunEventRow = Wire<RunsModels.Event>;
export type RunFilter = ListFilter<RunsModels.ListRequest>;
export type RunItemFilter = ListFilter<RunsModels.ListItemsRequest>;
export type Budget = Wire<RunDomainModels.Budget>;
export type Estimate = Wire<RunDomainModels.Estimate>;
export type RunTotals = Wire<RunDomainModels.Stats>;

export type Conversation = Wire<AgentModels.Conversation>;
export type Message = Wire<AgentModels.Message>;
export type PendingAction = Wire<AgentModels.PendingAction>;
export type ConversationFilter = ListFilter<AgentModels.ListConversationsRequest>;
export type MessageFilter = ListFilter<AgentModels.ListMessagesRequest>;
export type PendingActionFilter = ListFilter<AgentModels.ListPendingActionsRequest>;

export type Schedule = Wire<SchedulesModels.Schedule>;
export type ScheduleFilter = ListFilter<SchedulesModels.ListRequest>;

export type CatalogModel = Wire<CatalogModels.Model>;
export type ModelRef = Wire<LlmModels.ModelRef>;
export type RoleProfile = Wire<CatalogModels.Profile>;
export type TokenUsage = Wire<CatalogModels.Usage>;
export type UsageSummary = Wire<CatalogModels.UsageSummaryResponse>;
export type ProviderKey = Wire<CatalogModels.ProviderKey>;

export type ImportMapping = Wire<ImportModels.Mapping>;
export type ImportOptions = Wire<ImportModels.Options>;
export type ImportFinding = Wire<ImportModels.Finding>;
export type ImportConflict = Wire<ImportModels.Conflict>;
export type ImportCounts = Wire<ImportModels.Counts>;
export type PreviewReport = Wire<ImportModels.PreviewReport>;
export type PreviewPage = Wire<ImportModels.PreviewPage>;
export type PreviewEntity = Wire<ImportModels.PreviewEntity>;
export type PreviewEdge = Wire<ImportModels.PreviewEdge>;
export type InspectResult = Wire<ImportModels.InspectResponse>;

export type SiteOverview = Wire<ReportsModels.SiteOverviewResponse>;
export type PageReport = Wire<ReportsModels.PageReportResponse>;
export type RunReport = Wire<ReportsModels.RunReportResponse>;
export type ItemReport = Wire<ReportsModels.ItemReport>;
export type EntityScore = Wire<ReportsModels.EntityScore>;
export type DepthBucket = Wire<ReportsModels.DepthBucket>;
export type EntityTotals = Wire<ReportsModels.EntityTotals>;
export type PageTotals = Wire<ReportsModels.PageTotals>;
export type EdgeTotals = Wire<ReportsModels.EdgeTotals>;
export type JudgeResult = Wire<ContentModels.JudgeResponse>;
export type JudgeReport = Wire<ContentModels.JudgeReport>;

export type PluginState = Wire<SyncModels.Plugin>;

export type SettingDescriptor = Wire<SettingsModels.Descriptor>;
export type SettingValue = Wire<TransportModels.GetSettingResponse>;
export type LockState = Wire<TransportModels.LockStateResponse>;
export type BuildInfo = Wire<TransportModels.BuildInfo>;
export type Tool = Wire<TransportModels.Tool>;
export type ToolSchema = Wire<SchemaModels.Schema>;
