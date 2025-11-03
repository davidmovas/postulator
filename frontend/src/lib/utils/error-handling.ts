import { useToast } from "@/components/ui/use-toast";
import { AppError, ErrorCode } from "@/types/errors";

export interface ApiError {
    code: ErrorCode;
    message: string;
    userMessage?: string;
    context?: Record<string, unknown>;
}

export interface ApiResponse<T> {
    success: boolean;
    data?: T;
    error?: ApiError;
}

export class ToastError extends Error {
    constructor(
        message: string,
        public title: string = 'Error',
        public variant: 'destructive' | 'success' = 'destructive'
    ) {
        super(message);
        this.name = 'ToastError';
    }
}

export function useErrorHandling() {
    const { toast } = useToast();

    const handleError = (error: unknown, options?: {
        title?: string;
        fallbackMessage?: string;
        throwError?: boolean;
    }): never | void => {
        const title = options?.title || 'Error';
        let message = options?.fallbackMessage || 'An unexpected error occurred';

        if (error instanceof AppError) {
            message = error.userMessage || error.message;
        } else if (error instanceof Error) {
            message = error.message;
        } else if (typeof error === 'string') {
            message = error;
        }

        toast({
            title,
            description: message,
            variant: 'destructive',
        });

        if (options?.throwError !== false) {
            throw error;
        }
    };

    const showSuccess = (message: string, title: string = 'Success') => {
        toast({
            title,
            description: message,
            variant: 'success',
        });
    };

    const withToast = async <T>(
        operation: () => Promise<T>,
        options?: {
            successMessage?: string;
            errorTitle?: string;
            showSuccess?: boolean;
            rethrow?: boolean;
        }
    ): Promise<T | null> => {
        try {
            const result = await operation();

            if (options?.showSuccess && options?.successMessage) {
                showSuccess(options.successMessage);
            }

            return result;
        } catch (error) {
            handleError(error, {
                title: options?.errorTitle,
                throwError: options?.rethrow
            });
            return null;
        }
    };

    return {
        handleError,
        showSuccess,
        withToast,
    };
}

export function unwrapResponse<T>(response: ApiResponse<T>): T {
    if (!response.success || response.error) {
        const error = response.error || {
            code: 'UNKNOWN_ERROR' as ErrorCode,
            message: 'An unknown error occurred'
        };

        throw new AppError(error);
    }

    if (response.data === undefined) {
        throw new AppError({
            code: 'INTERNAL',
            message: 'No data received from server'
        });
    }

    return response.data;
}

export function unwrapArrayResponse<T>(response: ApiResponse<T[]>): T[] {
    const data = unwrapResponse(response);
    return data || [];
}