import { useToast } from "@/components/ui/use-toast";

export interface ApiError {
    code: string;
    message: string;
    context?: Record<string, unknown>;
}

export interface ApiResponse<T> {
    success: boolean;
    data?: T;
    error?: ApiError;
}

export class AppError extends Error {
    constructor(
        public code: string,
        message: string,
        public context?: Record<string, unknown>
    ) {
        super(message);
        this.name = 'AppError';
    }
}

// Кастомный хук для работы с ошибками
export function useErrorHandling() {
    const { toast } = useToast();

    // Error handler that shows toast and optionally throws
    const handleApiError = (error: ApiError, options?: {
        showToast?: boolean;
        throwError?: boolean;
        customTitle?: string;
    }): never | void => {
        const { showToast = true, throwError = true, customTitle } = options || {};

        if (showToast) {
            toast({
                title: customTitle || 'Error',
                description: error.message,
                variant: 'destructive',
            });
        }

        if (throwError) {
            throw new AppError(error.code, error.message, error.context);
        }
    };

    // Unwrap API response or throw error
    const unwrapResponse = <T>(response: ApiResponse<T>): T => {
        if (!response.success || !response.data) {
            if (response.error) {
                handleApiError(response.error);
            }
            throw new AppError('UNKNOWN_ERROR', 'An unknown error occurred');
        }
        return response.data;
    };

    // Unwrap array response
    const unwrapArrayResponse = <T>(response: ApiResponse<T[]>): T[] => {
        if (!response.success) {
            if (response.error) {
                handleApiError(response.error);
            }
            throw new AppError('UNKNOWN_ERROR', 'An unknown error occurred');
        }
        // Return empty array if data is null/undefined
        return response.data || [];
    };

    // Success toast helper
    const showSuccess = (message: string, title = 'Success') => {
        toast({
            title,
            description: message,
        });
    };

    // Generic error toast
    const showError = (message: string, title = 'Error') => {
        toast({
            title,
            description: message,
            variant: 'destructive',
        });
    };

    // Wrapper for async operations with error handling
    const withErrorHandling = async <T>(
        operation: () => Promise<T>,
        options?: {
            successMessage?: string;
            errorMessage?: string;
            showSuccess?: boolean;
        }
    ): Promise<T | null> => {
        try {
            const result = await operation();

            if (options?.showSuccess && options?.successMessage) {
                showSuccess(options.successMessage);
            }

            return result;
        } catch (error) {
            if (error instanceof AppError) {
                showError(error.message, options?.errorMessage || 'Error');
            } else if (error instanceof Error) {
                showError(error.message, options?.errorMessage || 'Error');
            } else {
                showError('An unexpected error occurred', options?.errorMessage || 'Error');
            }
            return null;
        }
    };

    return {
        handleApiError,
        unwrapResponse,
        unwrapArrayResponse,
        showSuccess,
        showError,
        withErrorHandling
    };
}

export function unwrapResponse<T>(response: ApiResponse<T>): T {
    if (!response.success || !response.data) {
        if (response.error) {
            throw new AppError(response.error.code, response.error.message, response.error.context);
        }
        throw new AppError('UNKNOWN_ERROR', 'An unknown error occurred');
    }
    return response.data;
}

export function unwrapArrayResponse<T>(response: ApiResponse<T[]>): T[] {
    if (!response.success) {
        if (response.error) {
            throw new AppError(response.error.code, response.error.message, response.error.context);
        }
        //TODO: TEMP LOG
        console.log(response);

        throw new AppError('UNKNOWN_ERROR', 'An unknown error occurred');
    }
    return response.data || [];
}