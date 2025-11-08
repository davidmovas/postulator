"use client";

import { useState, useCallback, useRef } from "react";
import { useApiCall } from "./use-api-call";
import { settingsService } from "@/services/settings";

export function useSettingsForm() {
    const { execute } = useApiCall();
    const [isSaving, setIsSaving] = useState(false);
    const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    const saveChanges = useCallback(async (section: string, data: any) => {
        setIsSaving(true);

        if (saveTimeoutRef.current) {
            clearTimeout(saveTimeoutRef.current);
        }

        saveTimeoutRef.current = setTimeout(async () => {
            try {
                if (section === 'healthCheck') {
                    await execute(
                        () => settingsService.updateHealthCheckSettings(data),
                        {
                            successMessage: "Settings updated",
                            showSuccessToast: false
                        }
                    );
                }
            } catch (error) {
                console.error(`Failed to save ${section} settings:`, error);
            } finally {
                setIsSaving(false);
                saveTimeoutRef.current = null;
            }
        }, 1000);
    }, [execute]);

    return {
        isSaving,
        saveChanges
    };
}