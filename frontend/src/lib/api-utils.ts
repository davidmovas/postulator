import { ApiResponse, ApiError } from "@/types/api";

export class ApiException extends Error {
    constructor(public apiError: ApiError) {
        super(apiError.userMessage || apiError.message);
        this.name = 'ApiException';
    }
}

export function unwrapResponse<T>(wailsResponse: any): T {
    const response = adaptWailsResponse<T>(wailsResponse);

    if (!response.success || response.error) {
        throw new ApiException(response.error!);
    }

    if (response.data === undefined) {
        throw new ApiException({
            code: 'INTERNAL',
            message: 'No data in response',
            userMessage: 'No data received from server',
            isUserFacing: false,
        });
    }

    return response.data;
}

export function adaptWailsResponse<T>(wailsResponse: any): ApiResponse<T> {
    return {
        success: wailsResponse.success,
        data: wailsResponse.data,
        error: wailsResponse.error ? {
            code: wailsResponse.error.code,
            message: wailsResponse.error.message,
            userMessage: wailsResponse.error.userMessage || wailsResponse.error.message,
            context: wailsResponse.error.context,
            isUserFacing: wailsResponse.error.isUserFacing,
        } : undefined,
    };
}