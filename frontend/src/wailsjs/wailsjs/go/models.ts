export namespace dto {
	
	export class CreatePromptRequest {
	    name: string;
	    system: string;
	    user: string;
	    is_default: boolean;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreatePromptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.system = source["system"];
	        this.user = source["user"];
	        this.is_default = source["is_default"];
	        this.is_active = source["is_active"];
	    }
	}
	export class CreateSitePromptRequest {
	    site_id: number;
	    prompt_id: number;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateSitePromptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.prompt_id = source["prompt_id"];
	        this.is_active = source["is_active"];
	    }
	}
	export class CreateSiteRequest {
	    name: string;
	    url: string;
	    username: string;
	    password: string;
	    is_active: boolean;
	    strategy?: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateSiteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.is_active = source["is_active"];
	        this.strategy = source["strategy"];
	    }
	}
	export class CreateSiteTopicRequest {
	    site_id: number;
	    topic_id: number;
	    priority: number;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateSiteTopicRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.topic_id = source["topic_id"];
	        this.priority = source["priority"];
	        this.is_active = source["is_active"];
	    }
	}
	export class CreateTopicRequest {
	    title: string;
	    keywords?: string;
	    category?: string;
	    tags?: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateTopicRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.keywords = source["keywords"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.is_active = source["is_active"];
	    }
	}
	export class PaginationRequest {
	    page: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.limit = source["limit"];
	    }
	}
	export class PaginationResponse {
	    page: number;
	    limit: number;
	    total: number;
	    total_pages: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginationResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.limit = source["limit"];
	        this.total = source["total"];
	        this.total_pages = source["total_pages"];
	    }
	}
	export class PromptResponse {
	    id: number;
	    name: string;
	    system: string;
	    user: string;
	    is_default: boolean;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new PromptResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.system = source["system"];
	        this.user = source["user"];
	        this.is_default = source["is_default"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class PromptListResponse {
	    prompts: PromptResponse[];
	    pagination?: PaginationResponse;
	
	    static createFrom(source: any = {}) {
	        return new PromptListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompts = this.convertValues(source["prompts"], PromptResponse);
	        this.pagination = this.convertValues(source["pagination"], PaginationResponse);
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
	
	export class SetDefaultPromptRequest {
	    id: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new SetDefaultPromptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	    }
	}
	export class SiteResponse {
	    id: number;
	    name: string;
	    url: string;
	    username: string;
	    password: string;
	    is_active: boolean;
	    // Go type: time
	    last_check: any;
	    status: string;
	    strategy: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SiteResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.is_active = source["is_active"];
	        this.last_check = this.convertValues(source["last_check"], null);
	        this.status = source["status"];
	        this.strategy = source["strategy"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class SiteListResponse {
	    sites: SiteResponse[];
	    pagination?: PaginationResponse;
	
	    static createFrom(source: any = {}) {
	        return new SiteListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sites = this.convertValues(source["sites"], SiteResponse);
	        this.pagination = this.convertValues(source["pagination"], PaginationResponse);
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
	export class SitePromptResponse {
	    id: number;
	    site_id: number;
	    site_name?: string;
	    prompt_id: number;
	    prompt_name?: string;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SitePromptResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.site_id = source["site_id"];
	        this.site_name = source["site_name"];
	        this.prompt_id = source["prompt_id"];
	        this.prompt_name = source["prompt_name"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class SitePromptListResponse {
	    site_prompts: SitePromptResponse[];
	    pagination?: PaginationResponse;
	
	    static createFrom(source: any = {}) {
	        return new SitePromptListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_prompts = this.convertValues(source["site_prompts"], SitePromptResponse);
	        this.pagination = this.convertValues(source["pagination"], PaginationResponse);
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
	
	
	export class SiteTopicResponse {
	    id: number;
	    site_id: number;
	    site_name?: string;
	    topic_id: number;
	    topic_title?: string;
	    priority: number;
	    is_active: boolean;
	    usage_count: number;
	    // Go type: time
	    last_used_at?: any;
	    round_robin_pos: number;
	
	    static createFrom(source: any = {}) {
	        return new SiteTopicResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.site_id = source["site_id"];
	        this.site_name = source["site_name"];
	        this.topic_id = source["topic_id"];
	        this.topic_title = source["topic_title"];
	        this.priority = source["priority"];
	        this.is_active = source["is_active"];
	        this.usage_count = source["usage_count"];
	        this.last_used_at = this.convertValues(source["last_used_at"], null);
	        this.round_robin_pos = source["round_robin_pos"];
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
	export class SiteTopicListResponse {
	    site_topics: SiteTopicResponse[];
	    pagination?: PaginationResponse;
	
	    static createFrom(source: any = {}) {
	        return new SiteTopicListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_topics = this.convertValues(source["site_topics"], SiteTopicResponse);
	        this.pagination = this.convertValues(source["pagination"], PaginationResponse);
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
	
	export class StrategyAvailabilityResponse {
	    site_id: number;
	    strategy: string;
	    can_continue: boolean;
	    total_topics: number;
	    active_topics: number;
	    unused_topics: number;
	    remaining_count: number;
	
	    static createFrom(source: any = {}) {
	        return new StrategyAvailabilityResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.strategy = source["strategy"];
	        this.can_continue = source["can_continue"];
	        this.total_topics = source["total_topics"];
	        this.active_topics = source["active_topics"];
	        this.unused_topics = source["unused_topics"];
	        this.remaining_count = source["remaining_count"];
	    }
	}
	export class TestConnectionResponse {
	    success: boolean;
	    status: string;
	    message: string;
	    details?: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new TestConnectionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.details = source["details"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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
	export class TestSiteConnectionRequest {
	    site_id: number;
	
	    static createFrom(source: any = {}) {
	        return new TestSiteConnectionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	    }
	}
	export class TopicResponse {
	    id: number;
	    title: string;
	    keywords: string;
	    category: string;
	    tags: string;
	    is_active: boolean;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new TopicResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.keywords = source["keywords"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.is_active = source["is_active"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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
	export class TopicListResponse {
	    topics: TopicResponse[];
	    pagination?: PaginationResponse;
	
	    static createFrom(source: any = {}) {
	        return new TopicListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.topics = this.convertValues(source["topics"], TopicResponse);
	        this.pagination = this.convertValues(source["pagination"], PaginationResponse);
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
	
	export class TopicSelectionRequest {
	    site_id: number;
	    strategy?: string;
	
	    static createFrom(source: any = {}) {
	        return new TopicSelectionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.strategy = source["strategy"];
	    }
	}
	export class TopicSelectionResponse {
	    topic?: TopicResponse;
	    site_topic?: SiteTopicResponse;
	    strategy: string;
	    can_continue: boolean;
	    remaining_count: number;
	
	    static createFrom(source: any = {}) {
	        return new TopicSelectionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.topic = this.convertValues(source["topic"], TopicResponse);
	        this.site_topic = this.convertValues(source["site_topic"], SiteTopicResponse);
	        this.strategy = source["strategy"];
	        this.can_continue = source["can_continue"];
	        this.remaining_count = source["remaining_count"];
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
	export class TopicStatsResponse {
	    site_id: number;
	    total_topics: number;
	    active_topics: number;
	    used_topics: number;
	    unused_topics: number;
	    unique_topics_left: number;
	    round_robin_position: number;
	    most_used_topic_id: number;
	    most_used_topic_count: number;
	    last_used_topic_id: number;
	    // Go type: time
	    last_used_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new TopicStatsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.total_topics = source["total_topics"];
	        this.active_topics = source["active_topics"];
	        this.used_topics = source["used_topics"];
	        this.unused_topics = source["unused_topics"];
	        this.unique_topics_left = source["unique_topics_left"];
	        this.round_robin_position = source["round_robin_position"];
	        this.most_used_topic_id = source["most_used_topic_id"];
	        this.most_used_topic_count = source["most_used_topic_count"];
	        this.last_used_topic_id = source["last_used_topic_id"];
	        this.last_used_at = this.convertValues(source["last_used_at"], null);
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
	export class TopicUsageResponse {
	    id: number;
	    site_id: number;
	    topic_id: number;
	    article_id: number;
	    strategy: string;
	    // Go type: time
	    used_at: any;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new TopicUsageResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.site_id = source["site_id"];
	        this.topic_id = source["topic_id"];
	        this.article_id = source["article_id"];
	        this.strategy = source["strategy"];
	        this.used_at = this.convertValues(source["used_at"], null);
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class TopicUsageListResponse {
	    usage_history: TopicUsageResponse[];
	    pagination?: PaginationResponse;
	
	    static createFrom(source: any = {}) {
	        return new TopicUsageListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.usage_history = this.convertValues(source["usage_history"], TopicUsageResponse);
	        this.pagination = this.convertValues(source["pagination"], PaginationResponse);
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
	
	export class UpdatePromptRequest {
	    id: number;
	    name: string;
	    type: string;
	    content: string;
	    description?: string;
	    is_default: boolean;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdatePromptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.content = source["content"];
	        this.description = source["description"];
	        this.is_default = source["is_default"];
	        this.is_active = source["is_active"];
	    }
	}
	export class UpdateSitePromptRequest {
	    id: number;
	    site_id: number;
	    prompt_id: number;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSitePromptRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.site_id = source["site_id"];
	        this.prompt_id = source["prompt_id"];
	        this.is_active = source["is_active"];
	    }
	}
	export class UpdateSiteRequest {
	    id: number;
	    name: string;
	    url: string;
	    username: string;
	    password: string;
	    is_active: boolean;
	    strategy?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSiteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.is_active = source["is_active"];
	        this.strategy = source["strategy"];
	    }
	}
	export class UpdateSiteTopicRequest {
	    id: number;
	    site_id: number;
	    topic_id: number;
	    priority: number;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSiteTopicRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.site_id = source["site_id"];
	        this.topic_id = source["topic_id"];
	        this.priority = source["priority"];
	        this.is_active = source["is_active"];
	    }
	}
	export class UpdateTopicRequest {
	    id: number;
	    title: string;
	    keywords?: string;
	    category?: string;
	    tags?: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateTopicRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.keywords = source["keywords"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.is_active = source["is_active"];
	    }
	}

}

export namespace handlers {
	
	export class Handler {
	
	
	    static createFrom(source: any = {}) {
	        return new Handler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

