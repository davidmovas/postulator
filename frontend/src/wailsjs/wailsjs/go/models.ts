export namespace dto {
	
	export class AIProvider {
	    id: number;
	    name: string;
	    model: string;
	    isActive: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AIProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.model = source["model"];
	        this.isActive = source["isActive"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class AIProviderCreate {
	    name: string;
	    apiKey: string;
	    model: string;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AIProviderCreate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.isActive = source["isActive"];
	    }
	}
	export class AIProviderUpdate {
	    id: number;
	    name: string;
	    apiKey?: string;
	    model: string;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AIProviderUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.isActive = source["isActive"];
	    }
	}
	export class Category {
	    id: number;
	    siteId: number;
	    wpCategoryId: number;
	    name: string;
	    slug?: string;
	    count: number;
	    createdAt: string;
	
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
	        this.count = source["count"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Error {
	    code: string;
	    message: string;
	    context?: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new Error(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.context = source["context"];
	    }
	}
	export class Execution {
	    id: number;
	    jobId: number;
	    topicId: number;
	    generatedTitle?: string;
	    generatedContent?: string;
	    status: string;
	    errorMessage?: string;
	    articleId?: number;
	    startedAt: string;
	    generatedAt?: string;
	    validatedAt?: string;
	    publishedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Execution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.jobId = source["jobId"];
	        this.topicId = source["topicId"];
	        this.generatedTitle = source["generatedTitle"];
	        this.generatedContent = source["generatedContent"];
	        this.status = source["status"];
	        this.errorMessage = source["errorMessage"];
	        this.articleId = source["articleId"];
	        this.startedAt = source["startedAt"];
	        this.generatedAt = source["generatedAt"];
	        this.validatedAt = source["validatedAt"];
	        this.publishedAt = source["publishedAt"];
	    }
	}
	export class ImportResult {
	    totalRead: number;
	    totalAdded: number;
	    totalSkipped: number;
	    added: string[];
	    skipped: string[];
	    errors: string[];
	
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
	export class Job {
	    id: number;
	    name: string;
	    siteId: number;
	    categoryId: number;
	    promptId: number;
	    aiProviderId: number;
	    aiModel: string;
	    requiresValidation: boolean;
	    scheduleType: string;
	    scheduleTime?: string;
	    scheduleDay?: number;
	    jitterEnabled: boolean;
	    jitterMinutes: number;
	    status: string;
	    lastRunAt?: string;
	    nextRunAt?: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.siteId = source["siteId"];
	        this.categoryId = source["categoryId"];
	        this.promptId = source["promptId"];
	        this.aiProviderId = source["aiProviderId"];
	        this.aiModel = source["aiModel"];
	        this.requiresValidation = source["requiresValidation"];
	        this.scheduleType = source["scheduleType"];
	        this.scheduleTime = source["scheduleTime"];
	        this.scheduleDay = source["scheduleDay"];
	        this.jitterEnabled = source["jitterEnabled"];
	        this.jitterMinutes = source["jitterMinutes"];
	        this.status = source["status"];
	        this.lastRunAt = source["lastRunAt"];
	        this.nextRunAt = source["nextRunAt"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ModelsByProvider {
	    openai: string[];
	    anthropic: string[];
	    google: string[];
	
	    static createFrom(source: any = {}) {
	        return new ModelsByProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.openai = source["openai"];
	        this.anthropic = source["anthropic"];
	        this.google = source["google"];
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
	export class PromptRenderResult {
	    system: string;
	    user: string;
	
	    static createFrom(source: any = {}) {
	        return new PromptRenderResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = source["system"];
	        this.user = source["user"];
	    }
	}
	export class Response__Postulator_internal_dto_AIProvider_ {
	    success: boolean;
	    data?: AIProvider;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_AIProvider_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIProvider);
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
	export class Response__Postulator_internal_dto_ImportResult_ {
	    success: boolean;
	    data?: ImportResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_ImportResult_(source);
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
	export class Response__Postulator_internal_dto_Job_ {
	    success: boolean;
	    data?: Job;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_Job_(source);
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
	export class Response__Postulator_internal_dto_ModelsByProvider_ {
	    success: boolean;
	    data?: ModelsByProvider;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_ModelsByProvider_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], ModelsByProvider);
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
	export class Response__Postulator_internal_dto_PromptRenderResult_ {
	    success: boolean;
	    data?: PromptRenderResult;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_PromptRenderResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], PromptRenderResult);
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
	export class Response__Postulator_internal_dto_Prompt_ {
	    success: boolean;
	    data?: Prompt;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_Prompt_(source);
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
	export class Site {
	    id: number;
	    name: string;
	    url: string;
	    wpUsername: string;
	    status: string;
	    lastHealthCheck?: string;
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
	        this.status = source["status"];
	        this.lastHealthCheck = source["lastHealthCheck"];
	        this.healthStatus = source["healthStatus"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Response__Postulator_internal_dto_Site_ {
	    success: boolean;
	    data?: Site;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_Site_(source);
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
	export class Response__Postulator_internal_dto_Topic_ {
	    success: boolean;
	    data?: Topic;
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response__Postulator_internal_dto_Topic_(source);
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
	export class Response____Postulator_internal_dto_AIProvider_ {
	    success: boolean;
	    data?: AIProvider[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_AIProvider_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], AIProvider);
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
	export class Response____Postulator_internal_dto_Category_ {
	    success: boolean;
	    data?: Category[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_Category_(source);
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
	export class Response____Postulator_internal_dto_Execution_ {
	    success: boolean;
	    data?: Execution[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_Execution_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], Execution);
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
	export class Response____Postulator_internal_dto_Job_ {
	    success: boolean;
	    data?: Job[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_Job_(source);
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
	export class Response____Postulator_internal_dto_Prompt_ {
	    success: boolean;
	    data?: Prompt[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_Prompt_(source);
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
	export class SiteTopic {
	    id: number;
	    siteId: number;
	    topicId: number;
	    categoryId: number;
	    strategy: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SiteTopic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.topicId = source["topicId"];
	        this.categoryId = source["categoryId"];
	        this.strategy = source["strategy"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Response____Postulator_internal_dto_SiteTopic_ {
	    success: boolean;
	    data?: SiteTopic[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_SiteTopic_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], SiteTopic);
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
	export class Response____Postulator_internal_dto_Site_ {
	    success: boolean;
	    data?: Site[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_Site_(source);
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
	export class Response____Postulator_internal_dto_Topic_ {
	    success: boolean;
	    data?: Topic[];
	    error?: Error;
	
	    static createFrom(source: any = {}) {
	        return new Response____Postulator_internal_dto_Topic_(source);
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
	
	

}

