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

export interface ApiError {
    code: ErrorCode;
    message: string;
    userMessage: string;
    context?: Record<string, unknown>;
    isUserFacing: boolean;
}

export interface ApiResponse<T> {
    success: boolean;
    data?: T;
    error?: ApiError;
}