export namespace dto {
	
	export class Article {
	    id: number;
	    siteId: number;
	    jobId?: number;
	    topicId: number;
	    title: string;
	    originalTitle: string;
	    content: string;
	    excerpt?: string;
	    wpPostId: number;
	    wpPostUrl: string;
	    wpCategoryIds: number[];
	    status: string;
	    wordCount?: number;
	    source: string;
	    isEdited: boolean;
	    createdAt: string;
	    publishedAt?: string;
	    updatedAt: string;
	    lastSyncedAt?: string;
	
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
	        this.status = source["status"];
	        this.wordCount = source["wordCount"];
	        this.source = source["source"];
	        this.isEdited = source["isEdited"];
	        this.createdAt = source["createdAt"];
	        this.publishedAt = source["publishedAt"];
	        this.updatedAt = source["updatedAt"];
	        this.lastSyncedAt = source["lastSyncedAt"];
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
	export class PaginatedResponse__github_com_davidmovas_postulator_internal_dto_Article_ {
	    success: boolean;
	    items?: Article[];
	    total?: number;
	    limit?: number;
	    offset?: number;
	    hasMore?: boolean;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedResponse__github_com_davidmovas_postulator_internal_dto_Article_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.items = this.convertValues(source["items"], Article);
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
	
	
	
	
	

}

