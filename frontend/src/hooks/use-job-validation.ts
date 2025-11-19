import { useMemo } from "react";
import { DailySchedule, IntervalSchedule, JobCreateInput, OnceSchedule } from "@/models/jobs";

const AUTO_FILLED_PLACEHOLDERS = ["title", "topic", "category"];

export function useJobValidation(formData: Partial<JobCreateInput>, prompts: any[] | null) {
    const selectedPrompt = prompts?.find(p => p.id === formData.promptId);

    return useMemo(() => {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Basic required fields
        if (!formData.name?.trim()) {
            errors.push("Job name is required");
        }

        if (!formData.siteId) {
            errors.push("Site selection is required");
        }

        if (!formData.promptId) {
            errors.push("Prompt selection is required");
        }

        if (!formData.aiProviderId) {
            errors.push("AI Provider selection is required");
        }

        // Content strategy validation
        if (!formData.topics || formData.topics.length === 0) {
            errors.push("At least one topic must be selected");
        }

        if (!formData.categories || formData.categories.length === 0) {
            errors.push("At least one category must be selected");
        }

        // Placeholders validation
        if (selectedPrompt) {
            const requiredPlaceholders = extractRequiredPlaceholders(selectedPrompt);
            const missingPlaceholders = requiredPlaceholders.filter(
                placeholder => !formData.placeholdersValues?.[placeholder]
            );

            if (missingPlaceholders.length > 0) {
                errors.push(`Missing values for placeholders: ${missingPlaceholders.join(", ")}`);
            }
        }

        // Schedule validation
        if (formData.schedule?.type === "once") {
            const executeAt = (formData.schedule.config as OnceSchedule)?.executeAt;
            if (!executeAt) {
                errors.push("Execution date and time are required for 'Run Once' schedule");
            } else if (new Date(executeAt).getTime() <= Date.now()) {
                warnings.push("Scheduled execution time is in the past");
            }
        }

        if (formData.schedule?.type === "interval") {
            const startAt = (formData.schedule.config as IntervalSchedule)?.startAt;
            if (startAt && new Date(startAt).getTime() <= Date.now()) {
                warnings.push("Interval start time is in the past");
            }
        }

        if (formData.schedule?.type === "daily") {
            const weekdays = (formData.schedule.config as DailySchedule)?.weekdays;
            if (!weekdays || weekdays.length === 0) {
                errors.push("At least one weekday must be selected for daily schedule");
            }
        }

        return {errors, warnings, isValid: errors.length === 0};
    }, [formData, selectedPrompt]);
}

function extractRequiredPlaceholders(prompt: any): string[] {
    if (!prompt) return [];

    const allPlaceholders = extractPlaceholdersFromPrompts(
        prompt.systemPrompt,
        prompt.userPrompt
    );

    return allPlaceholders.filter(
        placeholder => !AUTO_FILLED_PLACEHOLDERS.includes(placeholder.toLowerCase())
    );
}

// Заглушка - замените на вашу реальную функцию
function extractPlaceholdersFromPrompts(systemPrompt: string, userPrompt: string): string[] {
    const placeholders: string[] = [];
    const regex = /{(\w+)}/g;

    let match;
    while ((match = regex.exec(systemPrompt)) !== null) {
        placeholders.push(match[1]);
    }

    while ((match = regex.exec(userPrompt)) !== null) {
        placeholders.push(match[1]);
    }

    return [...new Set(placeholders)];
}