"use client";

import { useState, useEffect } from "react";
import { Job, JobCreateInput, JobUpdateInput } from "@/models/jobs";
import { useApiCall } from "./use-api-call";

interface UseJobFormProps {
    initialData?: Job;
    siteId?: number;
}

export function useJobForm({ initialData, siteId }: UseJobFormProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<Partial<JobCreateInput | JobUpdateInput>>({
        topicStrategy: "unique",
        categoryStrategy: "fixed",
        requiresValidation: false,
        jitterEnabled: true,
        jitterMinutes: 30,
        placeholdersValues: {},
        categories: [],
        topics: [],
        status: 'active',
        schedule: { type: "manual", config: {} }
    });

    // Инициализация формы данными при редактировании
    useEffect(() => {
        if (initialData) {
            setFormData({
                id: initialData.id,
                name: initialData.name,
                siteId: initialData.siteId,
                promptId: initialData.promptId,
                aiProviderId: initialData.aiProviderId,
                topicStrategy: initialData.topicStrategy,
                categoryStrategy: initialData.categoryStrategy,
                requiresValidation: initialData.requiresValidation,
                jitterEnabled: initialData.jitterEnabled,
                jitterMinutes: initialData.jitterMinutes,
                schedule: initialData.schedule,
                placeholdersValues: initialData.placeholdersValues,
                categories: initialData.categories,
                topics: initialData.topics
            });
        } else if (siteId) {
            setFormData(prev => ({ ...prev, siteId }));
        }
    }, [initialData, siteId]);

    const updateFormData = (updates: Partial<JobCreateInput | JobUpdateInput>) => {
        setFormData(prev => ({ ...prev, ...updates }));
    };

    const resetForm = () => {
        setFormData({
            topicStrategy: "unique",
            categoryStrategy: "fixed",
            requiresValidation: false,
            jitterEnabled: true,
            jitterMinutes: 30,
            placeholdersValues: {},
            categories: [],
            topics: [],
            schedule: { type: "manual", config: {} }
        });
    };

    return {
        formData,
        isLoading,
        updateFormData,
        resetForm,
        execute
    };
}