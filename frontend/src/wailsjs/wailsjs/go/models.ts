export namespace dto {
	
	export class BaseResponse {
	    success: boolean;
	    message?: string;
	    error?: string;
	    data?: any;
	
	    static createFrom(source: any = {}) {
	        return new BaseResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.error = source["error"];
	        this.data = source["data"];
	    }
	}
	export class CreateArticleManualRequest {
	    site_id: number;
	    topic_id: number;
	    publish: boolean;
	    custom_prompt?: string;
	    metadata?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new CreateArticleManualRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.topic_id = source["topic_id"];
	        this.publish = source["publish"];
	        this.custom_prompt = source["custom_prompt"];
	        this.metadata = source["metadata"];
	    }
	}
	export class CreateScheduleRequest {
	    site_id: number;
	    cron_expr: string;
	    posts_per_day: number;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateScheduleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.cron_expr = source["cron_expr"];
	        this.posts_per_day = source["posts_per_day"];
	        this.is_active = source["is_active"];
	    }
	}
	export class CreateSiteRequest {
	    name: string;
	    url: string;
	    username: string;
	    password: string;
	    api_key?: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateSiteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.api_key = source["api_key"];
	        this.is_active = source["is_active"];
	    }
	}
	export class CreateTopicRequest {
	    title: string;
	    description?: string;
	    keywords?: string;
	    prompt?: string;
	    category?: string;
	    tags?: string;
	    is_active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateTopicRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	        this.keywords = source["keywords"];
	        this.prompt = source["prompt"];
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
	export class PreviewArticleRequest {
	    site_id: number;
	    topic_id: number;
	    custom_prompt?: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewArticleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site_id = source["site_id"];
	        this.topic_id = source["topic_id"];
	        this.custom_prompt = source["custom_prompt"];
	    }
	}
	export class SettingRequest {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
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
	export class UpdateSiteRequest {
	    id: number;
	    name: string;
	    url: string;
	    username: string;
	    password: string;
	    api_key?: string;
	    is_active: boolean;
	
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
	        this.api_key = source["api_key"];
	        this.is_active = source["is_active"];
	    }
	}

}

