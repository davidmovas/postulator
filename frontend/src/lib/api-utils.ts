import { ApiResponse, ApiError } from "@/types/api";
import { PaginatedResponse } from "@/models/common";

export class ApiException extends Error {
    constructor(public apiError: ApiError) {
        super(apiError.userMessage || apiError.message);
        this.name = 'ApiException';
    }
}

export function unwrapArrayResponse<T>(wailsResponse: any): T[] {
    const response = adaptWailsResponse<T[]>(wailsResponse);

    if (!response.success || response.error) {
        throw new ApiException(response.error!);
    }

    return response.data || [];
}

export function unwrapResponse<T>(wailsResponse: any): T {
    const response = adaptWailsResponse<T>(wailsResponse);

    if (!response.success || response.error) {
        throw new ApiException(response.error!);
    }

    if (response.data === undefined || response.data === null) {
        throw new ApiException({
            code: 'INTERNAL',
            message: 'No data in response',
            userMessage: 'No data received from server',
            isUserFacing: false,
        });
    }

    return response.data;
}

export function unwrapPaginatedResponse<T, DTO>(
    wailsResponse: any,
    mapFn: (dto: DTO) => T
): PaginatedResponse<T> {
    if (!wailsResponse.success || wailsResponse.error) {
        throw new ApiException(wailsResponse.error!);
    }

    const payload = wailsResponse;

    if (!payload) {
        throw new ApiException({
            code: 'INTERNAL',
            message: 'No data in response',
            userMessage: 'No data received from server',
            isUserFacing: false,
        });
    }

    const items = (payload.items || []).map(mapFn);

    return {
        items,
        total: payload.total ?? 0,
        limit: payload.limit ?? 0,
        offset: payload.offset ?? 0,
        hasMore: Boolean(payload.hasMore),
    };
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