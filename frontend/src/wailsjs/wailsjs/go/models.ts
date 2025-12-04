export namespace dto {
	
	export class AIUsageByOperation {
	    operationType: string;
	    totalTokens: number;
	    totalCostUsd: number;
	    requestCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AIUsageByOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationType = source["operationType"];
	        this.totalTokens = source["totalTokens"];
	        this.totalCostUsd = source["totalCostUsd"];
	        this.requestCount = source["requestCount"];
	    }
	}
	export class AIUsageByPeriod {
	    period: string;
	    totalTokens: number;
	    totalCostUsd: number;
	    requestCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AIUsageByPeriod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.period = source["period"];
	        this.totalTokens = source["totalTokens"];
	        this.totalCostUsd = source["totalCostUsd"];
	        this.requestCount = source["requestCount"];
	    }
	}
	export class AIUsageByProvider {
	    providerName: string;
	    modelName: string;
	    totalTokens: number;
	    totalCostUsd: number;
	    requestCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AIUsageByProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerName = source["providerName"];
	        this.modelName = source["modelName"];
	        this.totalTokens = source["totalTokens"];
	        this.totalCostUsd = source["totalCostUsd"];
	        this.requestCount = source["requestCount"];
	    }
	}
	export class AIUsageBySite {
	    siteId: number;
	    siteName: string;
	    totalTokens: number;
	    totalCostUsd: number;
	    requestCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AIUsageBySite(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.siteName = source["siteName"];
	        this.totalTokens = source["totalTokens"];
	        this.totalCostUsd = source["totalCostUsd"];
	        this.requestCount = source["requestCount"];
	    }
	}
	export class AIUsageLog {
	    id: number;
	    siteId: number;
	    operationType: string;
	    providerName: string;
	    modelName: string;
	    inputTokens: number;
	    outputTokens: number;
	    totalTokens: number;
	    costUsd: number;
	    durationMs: number;
	    success: boolean;
	    errorMessage?: string;
	    metadata?: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AIUsageLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.operationType = source["operationType"];
	        this.providerName = source["providerName"];
	        this.modelName = source["modelName"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.totalTokens = source["totalTokens"];
	        this.costUsd = source["costUsd"];
	        this.durationMs = source["durationMs"];
	        this.success = source["success"];
	        this.errorMessage = source["errorMessage"];
	        this.metadata = source["metadata"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class AIUsageLogsResult {
	    items: AIUsageLog[];
	    total: number;
	    limit: number;
	    offset: number;
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AIUsageLogsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], AIUsageLog);
	        this.total = source["total"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	        this.hasMore = source["hasMore"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AIUsageSummary {
	    totalRequests: number;
	    totalTokens: number;
	    totalInputTokens: number;
	    totalOutputTokens: number;
	    totalCostUsd: number;
	    successCount: number;
	    errorCount: number;
	
	    static createFrom(source: any = {}) {
	        return new AIUsageSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRequests = source["totalRequests"];
	        this.totalTokens = source["totalTokens"];
	        this.totalInputTokens = source["totalInputTokens"];
	        this.totalOutputTokens = source["totalOutputTokens"];
	        this.totalCostUsd = source["totalCostUsd"];
	        this.successCount = source["successCount"];
	        this.errorCount = source["errorCount"];
	    }
	}
	export class AppVersion {
	    version: string;
	    commit: string;
	    buildDate: string;
	
	    static createFrom(source: any = {}) {
	        return new AppVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.buildDate = source["buildDate"];
	    }
	}
	export class Article {
	    id: number;
	    siteId: number;
	    jobId?: number;
	    topicId?: number;
	    title: string;
	    originalTitle: string;
	    content: string;
	    excerpt?: string;
	    wpPostId: number;
	    wpPostUrl: string;
	    wpCategoryIds: number[];
	    wpTagIds: number[];
	    status: string;
	    wordCount?: number;
	    source: string;
	    isEdited: boolean;
	    createdAt: string;
	    publishedAt?: string;
	    updatedAt: string;
	    lastSyncedAt?: string;
	    slug?: string;
	    featuredMediaId?: number;
	    featuredMediaUrl?: string;
	    metaDescription?: string;
	    author?: number;
	
	    static createFrom(source: any = {}) {
	        return new Article(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.jobId = source["jobId"];
	        this.topicId = source["topicId"];
	        this.title = source["title"];
	        this.originalTitle = source["originalTitle"];
	        this.content = source["content"];
	        this.excerpt = source["excerpt"];
	        this.wpPostId = source["wpPostId"];
	        this.wpPostUrl = source["wpPostUrl"];
	        this.wpCategoryIds = source["wpCategoryIds"];
	        this.wpTagIds = source["wpTagIds"];
	        this.status = source["status"];
	        this.wordCount = source["wordCount"];
	        this.source = source["source"];
	        this.isEdited = source["isEdited"];
	        this.createdAt = source["createdAt"];
	        this.publishedAt = source["publishedAt"];
	        this.updatedAt = source["updatedAt"];
	        this.lastSyncedAt = source["lastSyncedAt"];
	        this.slug = source["slug"];
	        this.featuredMediaId = source["featuredMediaId"];
	        this.featuredMediaUrl = source["featuredMediaUrl"];
	        this.metaDescription = source["metaDescription"];
	        this.author = source["author"];
	    }
	}
	export class ArticleListFilter {
	    siteId: number;
	    status?: string;
	    source?: string;
	    categoryId?: number;
	    search?: string;
	    sortBy: string;
	    sortOrder: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new ArticleListFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.categoryId = source["categoryId"];
	        this.search = source["search"];
	        this.sortBy = source["sortBy"];
	        this.sortOrder = source["sortOrder"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class ArticleListResult {
	    articles: Article[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new ArticleListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.articles = this.convertValues(source["articles"], Article);
	        this.total = source["total"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Site {
	    id: number;
	    name: string;
	    url: string;
	    wpUsername: string;
	    wpPassword: string;
	    status: string;
	    lastHealthCheck: string;
	    autoHealthCheck: boolean;
	    healthStatus: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Site(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.wpUsername = source["wpUsername"];
	        this.wpPassword = source["wpPassword"];
	        this.status = source["status"];
	        this.lastHealthCheck = source["lastHealthCheck"];
	        this.autoHealthCheck = source["autoHealthCheck"];
	        this.healthStatus = source["healthStatus"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class AutoCheckResult {
	    unhealthy: Site[];
	    recovered: Site[];
	
	    static createFrom(source: any = {}) {
	        return new AutoCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unhealthy = this.convertValues(source["unhealthy"], Site);
	        this.recovered = this.convertValues(source["recovered"], Site);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Topic {
	    id: number;
	    title: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Topic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class BatchResult {
	    created: number;
	    skipped: number;
	    skippedTitles: string[];
	    createdTopics: Topic[];
	
	    static createFrom(source: any = {}) {
	        return new BatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = source["created"];
	        this.skipped = source["skipped"];
	        this.skippedTitles = source["skippedTitles"];
	        this.createdTopics = this.convertValues(source["createdTopics"], Topic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Category {
	    id: number;
	    siteId: number;
	    wpCategoryId: number;
	    name: string;
	    slug?: string;
	    description?: string;
	    count: number;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.wpCategoryId = source["wpCategoryId"];
	        this.name = source["name"];
	        this.slug = source["slug"];
	        this.description = source["description"];
	        this.count = source["count"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CreateNodeRequest {
	    sitemapId: number;
	    parentId?: number;
	    title: string;
	    slug: string;
	    description?: string;
	    position: number;
	    source: string;
	    keywords?: string[];
	
	    static createFrom(source: any = {}) {
	        return new CreateNodeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapId = source["sitemapId"];
	        this.parentId = source["parentId"];
	        this.title = source["title"];
	        this.slug = source["slug"];
	        this.description = source["description"];
	        this.position = source["position"];
	        this.source = source["source"];
	        this.keywords = source["keywords"];
	    }
	}
	export class CreateSitemapRequest {
	    siteId: number;
	    name: string;
	    description?: string;
	    source: string;
	    siteUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateSitemapRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.source = source["source"];
	        this.siteUrl = source["siteUrl"];
	    }
	}
	export class DashboardSettings {
	    autoRefreshEnabled: boolean;
	    autoRefreshInterval: number;
	    minRefreshInterval: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoRefreshEnabled = source["autoRefreshEnabled"];
	        this.autoRefreshInterval = source["autoRefreshInterval"];
	        this.minRefreshInterval = source["minRefreshInterval"];
	    }
	}
	export class DashboardSummary {
	    totalSites: number;
	    activeSites: number;
	    unhealthySites: number;
	    totalJobs: number;
	    activeJobs: number;
	    pausedJobs: number;
	    pendingValidations: number;
	    executionsToday: number;
	    failedExecutionsToday: number;
	
	    static createFrom(source: any = {}) {
	        return new DashboardSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalSites = source["totalSites"];
	        this.activeSites = source["activeSites"];
	        this.unhealthySites = source["unhealthySites"];
	        this.totalJobs = source["totalJobs"];
	        this.activeJobs = source["activeJobs"];
	        this.pausedJobs = source["pausedJobs"];
	        this.pendingValidations = source["pendingValidations"];
	        this.executionsToday = source["executionsToday"];
	        this.failedExecutionsToday = source["failedExecutionsToday"];
	    }
	}
	export class DistributeKeywordsRequest {
	    sitemapId: number;
	    keywords: string[];
	    strategy: string;
	
	    static createFrom(source: any = {}) {
	        return new DistributeKeywordsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapId = source["sitemapId"];
	        this.keywords = source["keywords"];
	        this.strategy = source["strategy"];
	    }
	}
	export class DuplicateSitemapRequest {
	    id: number;
	    newName: string;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateSitemapRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.newName = source["newName"];
	    }
	}
	export class Error {
	    code: string;
	    message: string;
	    userMessage?: string;
	    context?: Record<string, any>;
	    isUserFacing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Error(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.userMessage = source["userMessage"];
	        this.context = source["context"];
	        this.isUserFacing = source["isUserFacing"];
	    }
	}
	export class FileFilter {
	    displayName: string;
	    pattern: string;
	
	    static createFrom(source: any = {}) {
	        return new FileFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.pattern = source["pattern"];
	    }
	}
	export class GenerateContentInput {
	    siteId: number;
	    providerId: number;
	    promptId: number;
	    topicId?: number;
	    customTopicTitle: string;
	    placeholderValues: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new GenerateContentInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.providerId = source["providerId"];
	        this.promptId = source["promptId"];
	        this.topicId = source["topicId"];
	        this.customTopicTitle = source["customTopicTitle"];
	        this.placeholderValues = source["placeholderValues"];
	    }
	}
	export class GenerateContentResult {
	    title: string;
	    content: string;
	    excerpt: string;
	    metaDescription: string;
	    topicId?: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateContentResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.content = source["content"];
	        this.excerpt = source["excerpt"];
	        this.metaDescription = source["metaDescription"];
	        this.topicId = source["topicId"];
	    }
	}
	export class TitleInput {
	    title: string;
	    keywords?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TitleInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.keywords = source["keywords"];
	    }
	}
	export class GenerateSitemapStructureRequest {
	    sitemapId?: number;
	    siteId?: number;
	    name?: string;
	    promptId: number;
	    placeholders?: Record<string, string>;
	    titles: TitleInput[];
	    parentNodeIds?: number[];
	    maxDepth?: number;
	    includeExistingTree?: boolean;
	    providerId: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateSitemapStructureRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapId = source["sitemapId"];
	        this.siteId = source["siteId"];
	        this.name = source["name"];
	        this.promptId = source["promptId"];
	        this.placeholders = source["placeholders"];
	        this.titles = this.convertValues(source["titles"], TitleInput);
	        this.parentNodeIds = source["parentNodeIds"];
	        this.maxDepth = source["maxDepth"];
	        this.includeExistingTree = source["includeExistingTree"];
	        this.providerId = source["providerId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GenerateSitemapStructureResponse {
	    sitemapId: number;
	    nodesCreated: number;
	    durationMs: number;
	
	    static createFrom(source: any = {}) {
	        return new GenerateSitemapStructureResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapId = source["sitemapId"];
	        this.nodesCreated = source["nodesCreated"];
	        this.durationMs = source["durationMs"];
	    }
	}
	export class HealthCheckHistory {
	    id: number;
	    siteId: number;
	    checkedAt: string;
	    status: string;
	    responseTimeMs: number;
	    statusCode: number;
	    errorMessage: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthCheckHistory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.checkedAt = source["checkedAt"];
	        this.status = source["status"];
	        this.responseTimeMs = source["responseTimeMs"];
	        this.statusCode = source["statusCode"];
	        this.errorMessage = source["errorMessage"];
	    }
	}
	export class HealthCheckSettings {
	    enabled: boolean;
	    interval_minutes: number;
	    min_interval_minutes: number;
	    notify_when_hidden: boolean;
	    notify_always: boolean;
	    notify_with_sound: boolean;
	    notify_on_recover: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HealthCheckSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.interval_minutes = source["interval_minutes"];
	        this.min_interval_minutes = source["min_interval_minutes"];
	        this.notify_when_hidden = source["notify_when_hidden"];
	        this.notify_always = source["notify_always"];
	        this.notify_with_sound = source["notify_with_sound"];
	        this.notify_on_recover = source["notify_on_recover"];
	    }
	}
	export class HistoryState {
	    canUndo: boolean;
	    canRedo: boolean;
	    undoCount: number;
	    redoCount: number;
	    lastAction?: string;
	    actionApplied?: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canUndo = source["canUndo"];
	        this.canRedo = source["canRedo"];
	        this.undoCount = source["undoCount"];
	        this.redoCount = source["redoCount"];
	        this.lastAction = source["lastAction"];
	        this.actionApplied = source["actionApplied"];
	    }
	}
	export class IPComparison {
	    direct_ip: string;
	    direct_error?: string;
	    proxy_ip: string;
	    proxy_error?: string;
	    is_anonymous: boolean;
	
	    static createFrom(source: any = {}) {
	        return new IPComparison(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.direct_ip = source["direct_ip"];
	        this.direct_error = source["direct_error"];
	        this.proxy_ip = source["proxy_ip"];
	        this.proxy_error = source["proxy_error"];
	        this.is_anonymous = source["is_anonymous"];
	    }
	}
	export class ImportError {
	    row?: number;
	    column?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.row = source["row"];
	        this.column = source["column"];
	        this.message = source["message"];
	    }
	}
	export class ImportNodesRequest {
	    sitemapId: number;
	    parentNodeId?: number;
	    filename: string;
	    fileDataBase64: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportNodesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapId = source["sitemapId"];
	        this.parentNodeId = source["parentNodeId"];
	        this.filename = source["filename"];
	        this.fileDataBase64 = source["fileDataBase64"];
	    }
	}
	export class ImportNodesResponse {
	    totalRows: number;
	    nodesCreated: number;
	    nodesSkipped: number;
	    errors?: ImportError[];
	    processingTime: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportNodesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRows = source["totalRows"];
	        this.nodesCreated = source["nodesCreated"];
	        this.nodesSkipped = source["nodesSkipped"];
	        this.errors = this.convertValues(source["errors"], ImportError);
	        this.processingTime = source["processingTime"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportResult {
	    totalRead: number;
	    totalAdded: number;
	    totalSkipped: number;
	    added?: string[];
	    skipped?: string[];
	    errors?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRead = source["totalRead"];
	        this.totalAdded = source["totalAdded"];
	        this.totalSkipped = source["totalSkipped"];
	        this.added = source["added"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}
	export class State {
	    jobId: number;
	    lastRunAt?: string;
	    nextRunAt?: string;
	    nextRunBase?: string;
	    totalExecutions: number;
	    failedExecutions: number;
	    lastCategoryIndex: number;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.lastRunAt = source["lastRunAt"];
	        this.nextRunAt = source["nextRunAt"];
	        this.nextRunBase = source["nextRunBase"];
	        this.totalExecutions = source["totalExecutions"];
	        this.failedExecutions = source["failedExecutions"];
	        this.lastCategoryIndex = source["lastCategoryIndex"];
	    }
	}
	export class Schedule {
	    type: string;
	    config: any;
	
	    static createFrom(source: any = {}) {
	        return new Schedule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.config = source["config"];
	    }
	}
	export class Job {
	    id: number;
	    name: string;
	    siteId: number;
	    promptId: number;
	    aiProviderId: number;
	    placeholdersValues: Record<string, string>;
	    topicStrategy: string;
	    categoryStrategy: string;
	    requiresValidation: boolean;
	    jitterEnabled: boolean;
	    jitterMinutes: number;
	    status: string;
	    createdAt: string;
	    updatedAt: string;
	    schedule?: Schedule;
	    state?: State;
	    categories: number[];
	    topics: number[];
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.siteId = source["siteId"];
	        this.promptId = source["promptId"];
	        this.aiProviderId = source["aiProviderId"];
	        this.placeholdersValues = source["placeholdersValues"];
	        this.topicStrategy = source["topicStrategy"];
	        this.categoryStrategy = source["categoryStrategy"];
	        this.requiresValidation = source["requiresValidation"];
	        this.jitterEnabled = source["jitterEnabled"];
	        this.jitterMinutes = source["jitterMinutes"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.schedule = this.convertValues(source["schedule"], Schedule);
	        this.state = this.convertValues(source["state"], State);
	        this.categories = source["categories"];
	        this.topics = source["topics"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JobTopicsStatus {
	    count: number;
	    topics: Topic[];
	
	    static createFrom(source: any = {}) {
	        return new JobTopicsStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.topics = this.convertValues(source["topics"], Topic);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LinkNodeToArticleRequest {
	    nodeId: number;
	    articleId: number;
	
	    static createFrom(source: any = {}) {
	        return new LinkNodeToArticleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.articleId = source["articleId"];
	    }
	}
	export class LinkNodeToPageRequest {
	    nodeId: number;
	    wpPageId: number;
	    wpUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new LinkNodeToPageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.wpPageId = source["wpPageId"];
	        this.wpUrl = source["wpUrl"];
	    }
	}
	export class MediaResult {
	    id: number;
	    sourceUrl: string;
	    altText: string;
	
	    static createFrom(source: any = {}) {
	        return new MediaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceUrl = source["sourceUrl"];
	        this.altText = source["altText"];
	    }
	}
	export class Model {
	    id: string;
	    name: string;
	    provider: string;
	    maxTokens: number;
	    inputCost: number;
	    outputCost: number;
	
	    static createFrom(source: any = {}) {
	        return new Model(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.maxTokens = source["maxTokens"];
	        this.inputCost = source["inputCost"];
	        this.outputCost = source["outputCost"];
	    }
	}
	export class MoveNodeRequest {
	    nodeId: number;
	    newParentId?: number;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new MoveNodeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.newParentId = source["newParentId"];
	        this.position = source["position"];
	    }
	}
	export class PaginatedResponse__github_com_davidmovas_postulator_internal_dto_HealthCheckHistory_ {
	    success: boolean;
	    items?: HealthCheckHistory[];
	    total?: number;
	    limit?: number;
	    offset?: number;
	    hasMore?: boolean;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedResponse__github_com_davidmovas_postulator_internal_dto_HealthCheckHistory_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.items = this.convertValues(source["items"], HealthCheckHistory);
	        this.total = source["total"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	        this.hasMore = source["hasMore"];
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Prompt {
	    id: number;
	    name: string;
	    systemPrompt: string;
	    userPrompt: string;
	    placeholders: string[];
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Prompt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.systemPrompt = source["systemPrompt"];
	        this.userPrompt = source["userPrompt"];
	        this.placeholders = source["placeholders"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Provider {
	    id: number;
	    name: string;
	    type: string;
	    apiKey: string;
	    model: string;
	    isActive: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Provider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.isActive = source["isActive"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProxyHealth {
	    node_id: string;
	    status: string;
	    latency_ms: number;
	    last_checked: number;
	    error?: string;
	    external_ip?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyHealth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.node_id = source["node_id"];
	        this.status = source["status"];
	        this.latency_ms = source["latency_ms"];
	        this.last_checked = source["last_checked"];
	        this.error = source["error"];
	        this.external_ip = source["external_ip"];
	    }
	}
	export class ProxyNode {
	    id: string;
	    type: string;
	    host: string;
	    port: number;
	    username?: string;
	    password?: string;
	    enabled: boolean;
	    order: number;
	
	    static createFrom(source: any = {}) {
	        return new ProxyNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.enabled = source["enabled"];
	        this.order = source["order"];
	    }
	}
	export class ProxySettings {
	    enabled: boolean;
	    mode: string;
	    nodes: ProxyNode[];
	    rotation_enabled: boolean;
	    rotation_interval: number;
	    health_check_enabled: boolean;
	    health_check_interval: number;
	    notify_on_failure: boolean;
	    notify_on_recover: boolean;
	    current_node_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.nodes = this.convertValues(source["nodes"], ProxyNode);
	        this.rotation_enabled = source["rotation_enabled"];
	        this.rotation_interval = source["rotation_interval"];
	        this.health_check_enabled = source["health_check_enabled"];
	        this.health_check_interval = source["health_check_interval"];
	        this.notify_on_failure = source["notify_on_failure"];
	        this.notify_on_recover = source["notify_on_recover"];
	        this.current_node_id = source["current_node_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProxyState {
	    status: string;
	    active_node_id?: string;
	    external_ip?: string;
	    latency_ms: number;
	    nodes_health: ProxyHealth[];
	    last_error?: string;
	    last_checked_at: number;
	
	    static createFrom(source: any = {}) {
	        return new ProxyState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.active_node_id = source["active_node_id"];
	        this.external_ip = source["external_ip"];
	        this.latency_ms = source["latency_ms"];
	        this.nodes_health = this.convertValues(source["nodes_health"], ProxyHealth);
	        this.last_error = source["last_error"];
	        this.last_checked_at = source["last_checked_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_AIUsageLogsResult_ {
	    success: boolean;
	    data?: AIUsageLogsResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_AIUsageLogsResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIUsageLogsResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_AIUsageSummary_ {
	    success: boolean;
	    data?: AIUsageSummary;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_AIUsageSummary_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIUsageSummary);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_AppVersion_ {
	    success: boolean;
	    data?: AppVersion;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_AppVersion_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AppVersion);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ArticleListResult_ {
	    success: boolean;
	    data?: ArticleListResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ArticleListResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ArticleListResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Article_ {
	    success: boolean;
	    data?: Article;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Article_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Article);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_AutoCheckResult_ {
	    success: boolean;
	    data?: AutoCheckResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_AutoCheckResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AutoCheckResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_BatchResult_ {
	    success: boolean;
	    data?: BatchResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_BatchResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], BatchResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Category_ {
	    success: boolean;
	    data?: Category;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Category_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Category);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_DashboardSettings_ {
	    success: boolean;
	    data?: DashboardSettings;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_DashboardSettings_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], DashboardSettings);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_DashboardSummary_ {
	    success: boolean;
	    data?: DashboardSummary;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_DashboardSummary_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], DashboardSummary);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_GenerateContentResult_ {
	    success: boolean;
	    data?: GenerateContentResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_GenerateContentResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], GenerateContentResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_GenerateSitemapStructureResponse_ {
	    success: boolean;
	    data?: GenerateSitemapStructureResponse;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_GenerateSitemapStructureResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], GenerateSitemapStructureResponse);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_HealthCheckHistory_ {
	    success: boolean;
	    data?: HealthCheckHistory;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_HealthCheckHistory_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], HealthCheckHistory);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_HealthCheckSettings_ {
	    success: boolean;
	    data?: HealthCheckSettings;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_HealthCheckSettings_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], HealthCheckSettings);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_HistoryState_ {
	    success: boolean;
	    data?: HistoryState;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_HistoryState_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], HistoryState);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_IPComparison_ {
	    success: boolean;
	    data?: IPComparison;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_IPComparison_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], IPComparison);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ImportNodesResponse_ {
	    success: boolean;
	    data?: ImportNodesResponse;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ImportNodesResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ImportNodesResponse);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ImportResult_ {
	    success: boolean;
	    data?: ImportResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ImportResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ImportResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_JobTopicsStatus_ {
	    success: boolean;
	    data?: JobTopicsStatus;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_JobTopicsStatus_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], JobTopicsStatus);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Job_ {
	    success: boolean;
	    data?: Job;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Job_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Job);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_MediaResult_ {
	    success: boolean;
	    data?: MediaResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_MediaResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], MediaResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Prompt_ {
	    success: boolean;
	    data?: Prompt;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Prompt_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Prompt);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Provider_ {
	    success: boolean;
	    data?: Provider;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Provider_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Provider);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ProxyHealth_ {
	    success: boolean;
	    data?: ProxyHealth;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ProxyHealth_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ProxyHealth);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ProxyNode_ {
	    success: boolean;
	    data?: ProxyNode;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ProxyNode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ProxyNode);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ProxySettings_ {
	    success: boolean;
	    data?: ProxySettings;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ProxySettings_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ProxySettings);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ProxyState_ {
	    success: boolean;
	    data?: ProxyState;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ProxyState_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ProxyState);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ScanError {
	    wpId?: number;
	    type?: string;
	    title?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wpId = source["wpId"];
	        this.type = source["type"];
	        this.title = source["title"];
	        this.message = source["message"];
	    }
	}
	export class ScanSiteResponse {
	    sitemapId: number;
	    pagesScanned: number;
	    postsScanned: number;
	    nodesCreated: number;
	    nodesSkipped: number;
	    totalDuration: string;
	    errors?: ScanError[];
	
	    static createFrom(source: any = {}) {
	        return new ScanSiteResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapId = source["sitemapId"];
	        this.pagesScanned = source["pagesScanned"];
	        this.postsScanned = source["postsScanned"];
	        this.nodesCreated = source["nodesCreated"];
	        this.nodesSkipped = source["nodesSkipped"];
	        this.totalDuration = source["totalDuration"];
	        this.errors = this.convertValues(source["errors"], ScanError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_ScanSiteResponse_ {
	    success: boolean;
	    data?: ScanSiteResponse;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_ScanSiteResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ScanSiteResponse);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SiteStats {
	    id: number;
	    siteId: number;
	    date: string;
	    articlesPublished: number;
	    articlesFailed: number;
	    totalWords: number;
	    internalLinksCreated: number;
	    externalLinksCreated: number;
	
	    static createFrom(source: any = {}) {
	        return new SiteStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.date = source["date"];
	        this.articlesPublished = source["articlesPublished"];
	        this.articlesFailed = source["articlesFailed"];
	        this.totalWords = source["totalWords"];
	        this.internalLinksCreated = source["internalLinksCreated"];
	        this.externalLinksCreated = source["externalLinksCreated"];
	    }
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_SiteStats_ {
	    success: boolean;
	    data?: SiteStats;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_SiteStats_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SiteStats);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Site_ {
	    success: boolean;
	    data?: Site;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Site_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Site);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SitemapNode {
	    id: number;
	    sitemapId: number;
	    parentId?: number;
	    title: string;
	    slug: string;
	    description?: string;
	    isRoot: boolean;
	    depth: number;
	    position: number;
	    path: string;
	    contentType: string;
	    articleId?: number;
	    wpPageId?: number;
	    wpUrl?: string;
	    source: string;
	    isSynced: boolean;
	    lastSyncedAt?: string;
	    wpTitle?: string;
	    wpSlug?: string;
	    isModified: boolean;
	    contentStatus: string;
	    positionX?: number;
	    positionY?: number;
	    keywords?: string[];
	    children?: SitemapNode[];
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SitemapNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sitemapId = source["sitemapId"];
	        this.parentId = source["parentId"];
	        this.title = source["title"];
	        this.slug = source["slug"];
	        this.description = source["description"];
	        this.isRoot = source["isRoot"];
	        this.depth = source["depth"];
	        this.position = source["position"];
	        this.path = source["path"];
	        this.contentType = source["contentType"];
	        this.articleId = source["articleId"];
	        this.wpPageId = source["wpPageId"];
	        this.wpUrl = source["wpUrl"];
	        this.source = source["source"];
	        this.isSynced = source["isSynced"];
	        this.lastSyncedAt = source["lastSyncedAt"];
	        this.wpTitle = source["wpTitle"];
	        this.wpSlug = source["wpSlug"];
	        this.isModified = source["isModified"];
	        this.contentStatus = source["contentStatus"];
	        this.positionX = source["positionX"];
	        this.positionY = source["positionY"];
	        this.keywords = source["keywords"];
	        this.children = this.convertValues(source["children"], SitemapNode);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_SitemapNode_ {
	    success: boolean;
	    data?: SitemapNode;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_SitemapNode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SitemapNode);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sitemap {
	    id: number;
	    siteId: number;
	    name: string;
	    description?: string;
	    source: string;
	    status: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Sitemap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.source = source["source"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SitemapWithNodes {
	    sitemap?: Sitemap;
	    nodes: SitemapNode[];
	
	    static createFrom(source: any = {}) {
	        return new SitemapWithNodes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemap = this.convertValues(source["sitemap"], Sitemap);
	        this.nodes = this.convertValues(source["nodes"], SitemapNode);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_SitemapWithNodes_ {
	    success: boolean;
	    data?: SitemapWithNodes;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_SitemapWithNodes_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SitemapWithNodes);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Sitemap_ {
	    success: boolean;
	    data?: Sitemap;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Sitemap_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Sitemap);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SupportedFormatsResponse {
	    formats: string[];
	
	    static createFrom(source: any = {}) {
	        return new SupportedFormatsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.formats = source["formats"];
	    }
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_SupportedFormatsResponse_ {
	    success: boolean;
	    data?: SupportedFormatsResponse;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_SupportedFormatsResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SupportedFormatsResponse);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SyncNodeResult {
	    nodeId: number;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncNodeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class SyncNodesResponse {
	    results: SyncNodeResult[];
	
	    static createFrom(source: any = {}) {
	        return new SyncNodesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], SyncNodeResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_SyncNodesResponse_ {
	    success: boolean;
	    data?: SyncNodesResponse;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_SyncNodesResponse_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SyncNodesResponse);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_Topic_ {
	    success: boolean;
	    data?: Topic;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_Topic_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Topic);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TorDetectionResult {
	    found: boolean;
	    port: number;
	    service_type: string;
	
	    static createFrom(source: any = {}) {
	        return new TorDetectionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.found = source["found"];
	        this.port = source["port"];
	        this.service_type = source["service_type"];
	    }
	}
	export class Response__github_com_davidmovas_postulator_internal_dto_TorDetectionResult_ {
	    success: boolean;
	    data?: TorDetectionResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__github_com_davidmovas_postulator_internal_dto_TorDetectionResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], TorDetectionResult);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Category_ {
	    success: boolean;
	    data?: Category[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Category_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Category);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_HealthCheckHistory_ {
	    success: boolean;
	    data?: HealthCheckHistory[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_HealthCheckHistory_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], HealthCheckHistory);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Job_ {
	    success: boolean;
	    data?: Job[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Job_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Job);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Model_ {
	    success: boolean;
	    data?: Model[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Model_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Model);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Prompt_ {
	    success: boolean;
	    data?: Prompt[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Prompt_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Prompt);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Provider_ {
	    success: boolean;
	    data?: Provider[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Provider_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Provider);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_SiteStats_ {
	    success: boolean;
	    data?: SiteStats[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_SiteStats_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SiteStats);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Site_ {
	    success: boolean;
	    data?: Site[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Site_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Site);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_SitemapNode_ {
	    success: boolean;
	    data?: SitemapNode[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_SitemapNode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SitemapNode);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Sitemap_ {
	    success: boolean;
	    data?: Sitemap[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Sitemap_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Sitemap);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Statistics {
	    categoryId: number;
	    date: string;
	    articlesPublished: number;
	    totalWords: number;
	
	    static createFrom(source: any = {}) {
	        return new Statistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.categoryId = source["categoryId"];
	        this.date = source["date"];
	        this.articlesPublished = source["articlesPublished"];
	        this.totalWords = source["totalWords"];
	    }
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Statistics_ {
	    success: boolean;
	    data?: Statistics[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Statistics_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Statistics);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response____github_com_davidmovas_postulator_internal_dto_Topic_ {
	    success: boolean;
	    data?: Topic[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____github_com_davidmovas_postulator_internal_dto_Topic_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Topic);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response___github_com_davidmovas_postulator_internal_dto_AIUsageByOperation_ {
	    success: boolean;
	    data?: AIUsageByOperation[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response___github_com_davidmovas_postulator_internal_dto_AIUsageByOperation_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIUsageByOperation);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response___github_com_davidmovas_postulator_internal_dto_AIUsageByPeriod_ {
	    success: boolean;
	    data?: AIUsageByPeriod[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response___github_com_davidmovas_postulator_internal_dto_AIUsageByPeriod_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIUsageByPeriod);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response___github_com_davidmovas_postulator_internal_dto_AIUsageByProvider_ {
	    success: boolean;
	    data?: AIUsageByProvider[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response___github_com_davidmovas_postulator_internal_dto_AIUsageByProvider_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIUsageByProvider);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response___github_com_davidmovas_postulator_internal_dto_AIUsageBySite_ {
	    success: boolean;
	    data?: AIUsageBySite[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response___github_com_davidmovas_postulator_internal_dto_AIUsageBySite_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIUsageBySite);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response___github_com_davidmovas_postulator_internal_dto_ProxyHealth_ {
	    success: boolean;
	    data?: ProxyHealth[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response___github_com_davidmovas_postulator_internal_dto_ProxyHealth_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ProxyHealth);
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response___string_ {
	    success: boolean;
	    data?: string[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response___string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response_int_ {
	    success: boolean;
	    data?: number;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response_int_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Response_string_ {
	    success: boolean;
	    data?: string;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response_string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = this.convertValues(source["error"], Error);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ScanIntoSitemapRequest {
	    sitemapId: number;
	    parentNodeId?: number;
	    titleSource: string;
	    contentFilter: string;
	    includeDrafts: boolean;
	    maxDepth: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanIntoSitemapRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sitemapId = source["sitemapId"];
	        this.parentNodeId = source["parentNodeId"];
	        this.titleSource = source["titleSource"];
	        this.contentFilter = source["contentFilter"];
	        this.includeDrafts = source["includeDrafts"];
	        this.maxDepth = source["maxDepth"];
	    }
	}
	export class ScanSiteRequest {
	    siteId: number;
	    sitemapName: string;
	    titleSource: string;
	    contentFilter: string;
	    includeDrafts: boolean;
	    maxDepth: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanSiteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.sitemapName = source["sitemapName"];
	        this.titleSource = source["titleSource"];
	        this.contentFilter = source["contentFilter"];
	        this.includeDrafts = source["includeDrafts"];
	        this.maxDepth = source["maxDepth"];
	    }
	}
	
	
	export class SetNodeKeywordsRequest {
	    nodeId: number;
	    keywords: string[];
	
	    static createFrom(source: any = {}) {
	        return new SetNodeKeywordsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.keywords = source["keywords"];
	    }
	}
	
	
	
	
	
	
	
	
	
	export class SyncNodesRequest {
	    siteId: number;
	    nodeIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new SyncNodesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.nodeIds = source["nodeIds"];
	    }
	}
	
	
	
	
	export class UpdateNodePositionsRequest {
	    nodeId: number;
	    positionX: number;
	    positionY: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateNodePositionsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeId = source["nodeId"];
	        this.positionX = source["positionX"];
	        this.positionY = source["positionY"];
	    }
	}
	export class UpdateNodeRequest {
	    id: number;
	    title: string;
	    slug: string;
	    description?: string;
	    keywords?: string[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateNodeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.slug = source["slug"];
	        this.description = source["description"];
	        this.keywords = source["keywords"];
	    }
	}
	export class UpdateNodesToWPRequest {
	    siteId: number;
	    nodeIds: number[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateNodesToWPRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.nodeIds = source["nodeIds"];
	    }
	}
	export class UpdateSitemapRequest {
	    id: number;
	    name: string;
	    description?: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSitemapRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.status = source["status"];
	    }
	}

}

