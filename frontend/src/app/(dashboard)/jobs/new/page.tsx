"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useApiCall } from "@/hooks/use-api-call";
import { jobService } from "@/services/jobs";
import { promptService } from "@/services/prompts";
import { siteService } from "@/services/sites";
import { topicService } from "@/services/topics";
import { DEFAULT_TOPIC_STRATEGY } from "@/constants/topics";
import { categoryService } from "@/services/categories";
import { providerService } from "@/services/providers";
import { JobCreateInput } from "@/models/jobs";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";

import { BasicInfoSection } from "@/components/jobs/form-sections/basic-info-section";
import { ContentStrategySection } from "@/components/jobs/form-sections/content-strategy-section";
import { PlaceholdersSection } from "@/components/jobs/form-sections/placeholders-section";
import { ScheduleSection } from "@/components/jobs/form-sections/schedule-section";
import { AdvancedSettingsSection } from "@/components/jobs/form-sections/advanced-settings-section";
import { useJobValidation } from "@/hooks/use-job-validation";

export default function CreateGlobalJobPage() {
    const router = useRouter();
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<Partial<JobCreateInput>>({
        topicStrategy: DEFAULT_TOPIC_STRATEGY,
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

    const [sites, setSites] = useState<any[]>([]);
    const [prompts, setPrompts] = useState<any[] | null>(null);
    const [providers, setProviders] = useState<any[] | null>(null);
    const [topics, setTopics] = useState<any[] | null>(null);
    const [categories, setCategories] = useState<any[] | null>(null);

    const validation = useJobValidation(formData, prompts);

    useEffect(() => {
        loadSites();
        loadPrompts();
        loadProviders();
    }, []);

    useEffect(() => {
        if (formData.siteId) {
            loadCategories();
            loadSelectableTopics();
            updateFormData({ topics: [] as any });
        }
    }, [formData.siteId]);

    useEffect(() => {
        if (formData.siteId) {
            loadSelectableTopics();
            updateFormData({ topics: [] as any });
        }
    }, [formData.topicStrategy]);

    const loadSites = async () => {
        const sitesData = await siteService.listSites();
        setSites(sitesData);
    };

    const loadPrompts = async () => {
        const promptsData = await promptService.listPrompts();
        setPrompts(promptsData);
    };

    const loadProviders = async () => {
        const providersData = await providerService.listProviders();
        setProviders(providersData.filter(p => p.isActive));
    };

    const loadSelectableTopics = async () => {
        if (!formData.siteId) return;
        const strategy = formData.topicStrategy || DEFAULT_TOPIC_STRATEGY;
        // Use 0 as jobId for new jobs to exclude topics from other unique-strategy jobs
        const topicsData = await topicService.getSelectableTopicsForJob(formData.siteId, strategy, 0);
        setTopics(topicsData);
    };

    const loadCategories = async () => {
        if (!formData.siteId) return;
        const categoriesData = await categoryService.listSiteCategories(formData.siteId);
        setCategories(categoriesData);
    };

    const handleSyncCategories = async () => {
        if (!formData.siteId) return;

        await execute(
            () => categoryService.syncFromWordPress(formData.siteId!),
            {
                successMessage: "Categories synced successfully",
                showSuccessToast: true,
                onSuccess: loadCategories
            }
        );
    };

    const handleSubmit = async () => {
        if (!formData.name || !formData.promptId || !formData.aiProviderId || !formData.siteId) {
            return;
        }

        const result = await execute<void>(
            () => jobService.createJob(formData as JobCreateInput),
            {
                successMessage: "Job created successfully",
                showSuccessToast: true
            }
        );

        if (result !== null) {
            router.push("/jobs");
        }
    };

    const updateFormData = (updates: Partial<JobCreateInput>) => {
        setFormData(prev => ({ ...prev, ...updates }));
    };

    return (
        <div className="p-6 space-y-6 max-w-4xl mx-auto">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link href="/jobs">
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                </Link>
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Create New Job</h1>
                    <p className="text-muted-foreground">
                        Create automated content generation job
                    </p>
                </div>
            </div>

            {/* Form Sections */}
            <div className="space-y-6">
                <BasicInfoSection
                    formData={formData}
                    onUpdate={updateFormData}
                    prompts={prompts}
                    providers={providers}
                    sites={sites}
                />

                {formData.siteId && (
                    <>
                        <ContentStrategySection
                            formData={formData}
                            onUpdate={updateFormData}
                            topics={topics}
                            categories={categories}
                            onSyncCategories={handleSyncCategories}
                        />

                        <PlaceholdersSection
                            formData={formData}
                            onUpdate={updateFormData}
                            prompts={prompts}
                        />
                    </>
                )}

                <ScheduleSection
                    formData={formData}
                    onUpdate={updateFormData}
                />

                <AdvancedSettingsSection
                    formData={formData}
                    onUpdate={updateFormData}
                />
            </div>

            {/* Actions */}
            <div className="flex justify-between gap-3 pt-6 border-t">
                <div>
                    {validation.errors.length > 0 && (
                        <div className="p-4 border border-destructive/50 bg-destructive/10 rounded-md">
                            <h4 className="font-medium text-destructive mb-2">Please fix the following errors:</h4>
                            <ul className="text-sm text-destructive list-disc list-inside space-y-1">
                                {validation.errors.map((error, index) => (
                                    <li key={index}>{error}</li>
                                ))}
                            </ul>
                        </div>
                    )}
                </div>

                <div className="flex gap-3">
                    <Button
                        variant="outline"
                        onClick={() => router.push("/jobs")}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!validation.isValid || isLoading}
                    >
                        {isLoading ? "Creating..." : "Create Job"}
                    </Button>
                </div>
            </div>
        </div>
    );
}