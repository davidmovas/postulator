import type {
    AgentModels,
    CatalogModels,
    ContentModels,
    ContextPackModels,
    DtoModels,
    FieldModels,
    FindingsModels,
    GraphModels,
    ImportModels,
    LlmModels,
    ObserveModels,
    PagesModels,
    PageTypeDomainModels,
    PageTypesModels,
    ReportsModels,
    RunDomainModels,
    RunsModels,
    SchedulesModels,
    SchemaModels,
    SettingsModels,
    SiteModelModels,
    SitesModels,
    SyncModels,
    TemplateDomainModels,
    TemplatesModels,
    ToolsModels,
    TransportModels,
} from "../lib/api.js";
import type { Wire } from "./wire.js";

export type ListFilter<T> = Omit<T, "cursor" | "limit" | "sort">;

export type SortSpec = DtoModels.Sort;
export type Keyword = Wire<DtoModels.Keyword>;

export type Site = Wire<SitesModels.Site>;
export type SiteDefaults = Wire<SitesModels.Defaults>;
export type SitePlugin = Wire<SitesModels.Plugin>;
export type SiteFilter = ListFilter<SitesModels.ListRequest>;
export type Reachability = Wire<SitesModels.Reachability>;
export type SiteSettings = Site["settings"];

export type Page = Wire<PagesModels.Page>;
export type PageLink = Wire<PagesModels.PageLink>;
export type PageTreeNode = Wire<PagesModels.TreeNode>;
export type PreviewLink = Wire<PagesModels.PreviewLinkResponse>;
export type PageFilter = ListFilter<PagesModels.ListRequest>;
export type PageCountRequest = PagesModels.CountRequest;
export type PageSelection = PagesModels.Selection;
export type PageFacets = PagesModels.PageFilter;
export type PageSubtree = PagesModels.Subtree;
export type SetTypesRequest = PagesModels.SetTypesRequest;
export type SetTypesResponse = Wire<PagesModels.SetTypesResponse>;
export type SetTemplatesRequest = PagesModels.SetTemplatesRequest;
export type SetTemplatesResponse = Wire<PagesModels.SetTemplatesResponse>;

export type PageModel = Wire<ObserveModels.ModelView>;
export type ModelIdentity = Wire<ObserveModels.IdentityView>;
export type ModelHierarchy = Wire<ObserveModels.HierarchyView>;
export type ModelTerm = Wire<ObserveModels.TermView>;
export type ModelRender = Wire<ObserveModels.RenderView>;
export type ModelRegion = Wire<ObserveModels.RegionView>;
export type ModelTrail = Wire<ObserveModels.TrailView>;
export type ModelCrumb = Wire<ObserveModels.CrumbView>;
export type ModelLink = Wire<ObserveModels.LinkView>;
export type ModelBlock = Wire<ObserveModels.BlockView>;
export type ModelField = Wire<ObserveModels.FieldView>;
export type ModelFinding = Wire<ObserveModels.FindingView>;
export type ModelSummary = Wire<ObserveModels.Summary>;

export type Finding = Wire<FindingsModels.Finding>;
export type FindingChoice = Wire<FindingsModels.Option>;
export type FindingFilter = ListFilter<FindingsModels.ListRequest>;
export type FindingCountsRequest = FindingsModels.CountsRequest;
export type FindingCounts = Wire<FindingsModels.CountsResponse>;
export type FindingCodeCount = Wire<FindingsModels.CodeCount>;
export type ExplainRequest = FindingsModels.ExplainRequest;
export type Explanation = Wire<FindingsModels.Explanation>;
export type ResolveFindingRequest = FindingsModels.ResolveRequest;
export type ResolveFindingResponse = Wire<FindingsModels.ResolveResponse>;
export type ResolveFindingsRequest = FindingsModels.ResolveManyRequest;
export type ResolveFindingsResponse = Wire<FindingsModels.ResolveManyResponse>;

export type SiteMapView = Wire<SiteModelModels.MapView>;
export type SiteMapNode = Wire<ContextPackModels.MapNode>;
export type SiteMapRef = Wire<ContextPackModels.Ref>;
export type ParentWhy = Wire<ContextPackModels.ParentWhy>;
export type SiteMapFilter = Omit<SiteModelModels.MapRequest, "cursor" | "limit">;

export type WriterPack = Wire<SiteModelModels.PackView>;
export type WriterPackSection = Wire<SiteModelModels.Section>;
export type WriterPackDrop = Wire<SiteModelModels.Drop>;
export type WriterDirections = Pick<SiteModelModels.PackRequest, "brief" | "directive" | "runId">;

export type Entity = Wire<GraphModels.Entity>;
export type Edge = Wire<GraphModels.Edge>;
export type Anchor = Wire<GraphModels.Anchor>;
export type EntityPage = Wire<GraphModels.EntityPage>;
export type EntityFilter = ListFilter<GraphModels.ListEntitiesRequest>;
export type ProposedEntity = Wire<GraphModels.ProposedEntity>;
export type PreviewFromPagesRequest = GraphModels.PreviewFromPagesRequest;
export type PreviewFromPagesResponse = Wire<GraphModels.PreviewFromPagesResponse>;
export type ProposeFromKeywordsResponse = Wire<GraphModels.ProposeFromKeywordsResponse>;
export type ApplyProposalsResponse = Wire<GraphModels.ApplyProposalsResponse>;
export type EdgeFilter = ListFilter<GraphModels.ListEdgesRequest>;
export type CardProposal = Wire<GraphModels.CardProposal>;
export type ProposedCard = Wire<GraphModels.Card>;
export type CardFact = Wire<GraphModels.Fact>;
export type Adoption = Wire<GraphModels.AdoptionView>;
export type AdoptParentsRequest = GraphModels.AdoptParentsRequest;
export type AdoptParentsResponse = Wire<GraphModels.AdoptParentsResponse>;

export type Template = Wire<TemplatesModels.Template>;
export type TemplateOverride = Wire<TemplatesModels.Override>;
export type TemplateFilter = ListFilter<TemplatesModels.ListTemplatesRequest>;
export type TemplateSpec = Wire<TemplateDomainModels.TemplateSpec>;
export type TemplateSection = Wire<TemplateDomainModels.Section>;
export type LinkRules = Wire<FieldModels.LinkRules>;
export type LinkPolicy = Wire<FieldModels.LinkPolicy>;
export type PageType = Wire<PageTypesModels.Type>;
export type FieldScope = Wire<PageTypesModels.Scope>;
export type CatalogField = Wire<PageTypesModels.Field>;
export type PageTypeList = Wire<PageTypesModels.ListResponse>;
export type SetScopeRequest = Wire<PageTypesModels.SetScopeRequest>;
export type Rule = Wire<PageTypesModels.Rule>;
export type RuleConditions = PageTypeDomainModels.Conditions;
export type RuleURL = PageTypeDomainModels.URLMatch;
export type RuleList = Wire<PageTypesModels.RulesResponse>;
export type SaveRuleRequest = PageTypesModels.SaveRuleRequest;
export type ClassificationPreview = Wire<PageTypesModels.PreviewResponse>;
export type ClassificationGroup = Wire<PageTypesModels.PreviewGroup>;
export type ClassificationGroupKey = PageTypesModels.GroupKey;
export type ClassificationPage = Wire<PageTypesModels.PreviewPage>;
export type ClassificationPagesFilter = ListFilter<PageTypesModels.PreviewPagesRequest>;
export type ApplyClassificationResponse = Wire<PageTypesModels.ApplyResponse>;
export type TestRuleRequest = PageTypesModels.TestRuleRequest;
export type RuleTest = Wire<PageTypesModels.TestRuleResponse>;
export type PageFacts = Wire<PageTypesModels.FactsView>;
export type MetaRules = Wire<TemplateDomainModels.MetaRules>;
export type KeywordRules = Wire<TemplateDomainModels.KeywordRules>;
export type StepSpec = Wire<TemplateDomainModels.StepSpec>;

export type Run = Wire<RunsModels.Run>;
export type RunItem = Wire<RunsModels.Item>;
export type AwaitedParent = Wire<RunsModels.AwaitedParent>;
export type AddedPage = Wire<RunsModels.AddedPage>;
export type RunArtifact = Wire<RunsModels.Artifact>;
export type RunEventRow = Wire<RunsModels.Event>;
export type RunFilter = ListFilter<RunsModels.ListRequest>;
export type RunItemFilter = ListFilter<RunsModels.ListItemsRequest>;
export type RunStartRequest = RunsModels.StartRequest;
export type RunBrief = RunDomainModels.Brief;
export type RunDirective = RunDomainModels.Directive;
export type RunMustLink = RunDomainModels.MustLink;
export type Budget = Wire<RunDomainModels.Budget>;
export type Estimate = Wire<RunDomainModels.Estimate>;
export type RunTotals = Wire<RunDomainModels.Stats>;

export type Conversation = Wire<AgentModels.Conversation>;
export type Message = Wire<AgentModels.Message>;
export type PendingAction = Wire<AgentModels.PendingAction>;
export type ConversationFilter = ListFilter<AgentModels.ListConversationsRequest>;
export type MessageFilter = ListFilter<AgentModels.ListMessagesRequest>;
export type PendingActionFilter = ListFilter<AgentModels.ListPendingActionsRequest>;
export type Plan = Wire<ToolsModels.PlanView>;
export type PlanStep = Wire<ToolsModels.PlanStepView>;
export type PlanStepEdit = Wire<ToolsModels.EditView>;
export type PlanStepChoice = AgentModels.StepChoice;

export type Schedule = Wire<SchedulesModels.Schedule>;
export type ScheduleFilter = ListFilter<SchedulesModels.ListRequest>;

export type CatalogModel = Wire<CatalogModels.Model>;
export type ModelRef = Wire<LlmModels.ModelRef>;
export type RoleProfile = Wire<CatalogModels.Profile>;
export type TokenUsage = Wire<CatalogModels.Usage>;
export type UsageSummary = Wire<CatalogModels.UsageSummaryResponse>;
export type ProviderKey = Wire<CatalogModels.ProviderKey>;
export type SpendReport = Wire<CatalogModels.SpendReportResponse>;
export type SpendTotals = Wire<CatalogModels.SpendTotals>;
export type SpendSlice = Wire<CatalogModels.SpendSlice>;
export type ModelCall = Wire<CatalogModels.Call>;
export type ModelCallFilter = ListFilter<CatalogModels.ListCallsRequest>;

export type ImportSheet = Wire<ImportModels.Sheet>;
export type ImportMapping = Wire<ImportModels.Mapping>;
export type ImportSheetMapping = Wire<ImportModels.SheetMapping>;
export type ImportOptions = Wire<ImportModels.Options>;
export type ImportFinding = Wire<ImportModels.Finding>;
export type ImportConflict = Wire<ImportModels.Conflict>;
export type ImportCounts = Wire<ImportModels.Counts>;
export type PreviewReport = Wire<ImportModels.PreviewReport>;
export type PreviewPage = Wire<ImportModels.PreviewPage>;
export type ImportCandidate = Wire<ImportModels.Candidate>;
export type ImportChoice = ImportModels.Choice;
export type PreviewEntity = Wire<ImportModels.PreviewEntity>;
export type PreviewGroup = Wire<ImportModels.PreviewGroup>;
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
export type LinkAudit = Wire<ReportsModels.LinkAuditResponse>;
export type LinkAuditPage = Wire<ReportsModels.LinkAuditPageResponse>;
export type PageAudit = Wire<ReportsModels.PageAudit>;
export type RequiredLink = Wire<ReportsModels.RequiredLink>;
export type ExtraLink = Wire<ReportsModels.ExtraLink>;
export type LinkTotals = Wire<ReportsModels.LinkTotals>;
export type LinkTypeCounts = Wire<ReportsModels.LinkTypeCounts>;
export type JudgeResult = Wire<ContentModels.JudgeResponse>;
export type JudgeReport = Wire<ContentModels.JudgeReport>;

export type PluginState = Wire<SyncModels.Plugin>;
export type SyncSiteRequest = SyncModels.SyncSiteRequest;

export type SettingDescriptor = Wire<SettingsModels.Descriptor>;
export type SettingValue = Wire<TransportModels.GetSettingResponse>;
export type LockState = Wire<TransportModels.LockStateResponse>;
export type BuildInfo = Wire<TransportModels.BuildInfo>;
export type Tool = Wire<TransportModels.Tool>;
export type ToolSchema = Wire<SchemaModels.Schema>;
