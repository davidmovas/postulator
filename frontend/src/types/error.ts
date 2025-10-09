export interface ApiError {
    code: string;
    message: string;
    details?: string;
    fields?: Record<string, string>;
    technical?: string;
}