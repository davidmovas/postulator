import { useToast } from "@/components/ui/use-toast";
import { ApiException } from "@/lib/api-utils";
import { ErrorCode } from "@/types/api";
import { useState, useCallback } from "react";

interface UseApiCallOptions<T> {
    onSuccess?: (data: T) => void;
    onError?: (error: ApiException) => void;
    successMessage?: string;
    errorTitle?: string;
    showSuccessToast?: boolean;
}

const ERROR_TITLES: Record<ErrorCode, string> = {
    VALIDATION: 'Validation Error',
    NOT_FOUND: 'Not Found',
    ALREADY_EXISTS: 'Already Exists',
    SITE_UNREACHABLE: 'Site Unreachable',
    SITE_AUTH: 'Authentication Failed',
    WORDPRESS: 'WordPress Error',
    AI: 'AI Service Error',
    AI_RATE_LIMIT: 'Rate Limit Exceeded',
    IMPORT: 'Import Error',
    JOB_EXECUTION: 'Job Execution Failed',
    SCHEDULER: 'Scheduler Error',
    DATABASE: 'Database Error',
    INTERNAL: 'Internal Error',
};

export function useApiCall() {
    const { toast } = useToast();
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<ApiException | null>(null);

    const execute = useCallback(async <T = any>(
        apiCall: () => Promise<T>,
        options?: UseApiCallOptions<T>
    ): Promise<T | null> => {
        try {
            setIsLoading(true);
            setError(null);

            const result = await apiCall();

            if (options?.showSuccessToast && options?.successMessage) {
                toast({
                    title: "Success",
                    description: options.successMessage,
                    variant: "success",
                });
            }

            options?.onSuccess?.(result);
            return result;

        } catch (err) {
            let apiException: ApiException;

            if (err instanceof ApiException) {
                apiException = err;
            } else {
                apiException = new ApiException({
                    code: 'INTERNAL',
                    message: err instanceof Error ? err.message : 'Unknown error',
                    userMessage: 'An unexpected error occurred',
                    isUserFacing: false,
                });
            }

            const errorTitle = options?.errorTitle || ERROR_TITLES[apiException.apiError.code] || 'Error';
            const errorMessage = apiException.apiError.userMessage;

            toast({
                title: errorTitle,
                description: errorMessage,
                variant: "destructive",
            });

            setError(apiException);
            options?.onError?.(apiException);
            return null;

        } finally {
            setIsLoading(false);
        }
    }, [toast]);

    return {
        execute,
        isLoading,
        error,
    };
}