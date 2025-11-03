export type ErrorCode =
    | 'INTERNAL'
    | 'VALIDATION'
    | 'NOT_FOUND'
    | 'ALREADY_EXISTS'
    | 'DATABASE'
    | 'SITE_UNREACHABLE'
    | 'SITE_AUTH'
    | 'WORDPRESS'
    | 'AI'
    | 'AI_RATE_LIMIT'
    | 'IMPORT'
    | 'JOB_EXECUTION'
    | 'SCHEDULER'
    | string;

export class AppError {
    code: string;
    message: string;
    userMessage?: string;
    context?: Record<string, any>;
    isUserFacing: boolean;

    constructor(error?: any) {
        this.code = error?.code || 'INTERNAL';
        this.message = error?.message || 'Unknown error occurred';
        this.userMessage = error?.userMessage;
        this.context = error?.context;
        this.isUserFacing = this.determineUserFacing(this.code);
    }

    private determineUserFacing(code: ErrorCode): boolean {
        const nonUserFacing: ErrorCode[] = ['INTERNAL', 'DATABASE'];
        return !nonUserFacing.includes(code);
    }
}

export class Response<T> {
    success: boolean;
    data?: T;
    error?: AppError;

    constructor(response: any) {
        this.success = response.success;
        this.data = response.data;
        this.error = response.error ? new AppError(response.error) : undefined;
    }

    unwrap(): T {
        if (!this.success || this.error) {
            throw this.error || new AppError({ message: 'Request failed' });
        }
        if (this.data === undefined) {
            throw new AppError({ message: 'No data received' });
        }
        return this.data;
    }
}

export class PaginatedResponse<T> {
    success: boolean;
    items: T[];
    total: number;
    limit: number;
    offset: number;
    hasMore: boolean;
    error?: AppError;

    constructor(response: any) {
        this.success = response.success;
        this.items = response.items || [];
        this.total = response.total || 0;
        this.limit = response.limit || 0;
        this.offset = response.offset || 0;
        this.hasMore = response.hasMore || false;
        this.error = response.error ? new AppError(response.error) : undefined;
    }

    unwrap(): T[] {
        if (!this.success || this.error) {
            throw this.error || new AppError({ message: 'Request failed' });
        }
        return this.items;
    }
}