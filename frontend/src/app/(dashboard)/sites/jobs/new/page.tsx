"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter } from "next/navigation";
import { useQueryId } from "@/hooks/use-query-param";
import { Button } from "@/components/ui/button";
import { useApiCall } from "@/hooks/use-api-call";
import { jobService } from "@/services/jobs";
import { promptService } from "@/services/prompts";
import { siteService } from "@/services/sites";
import { topicService } from "@/services/topics";
import { DEFAULT_TOPIC_STRATEGY } from "@/constants/topics";
import { categoryService } from "@/services/categories";
import { providerService } from "@/services/providers";
import { useJobForm } from "@/hooks/use-job-form";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";

import { BasicInfoSection } from "@/components/jobs/form-sections/basic-info-section";
import { ContentStrategySection } from "@/components/jobs/form-sections/content-strategy-section";
import { PlaceholdersSection } from "@/components/jobs/form-sections/placeholders-section";
import { ScheduleSection } from "@/components/jobs/form-sections/schedule-section";
import { AdvancedSettingsSection } from "@/components/jobs/form-sections/advanced-settings-section";
import { useJobValidation } from "@/hooks/use-job-validation";

function CreateSiteJobPageContent() {
    const router = useRouter();
    const siteId = useQueryId();

    const { formData, isLoading, updateFormData } = useJobForm({ siteId });
    const { execute } = useApiCall();

    const [site, setSite] = useState<any>(null);
    const [prompts, setPrompts] = useState<any[] | null>(null);
    const [providers, setProviders] = useState<any[] | null>(null);
    const [topics, setTopics] = useState<any[] | null>(null);
    const [categories, setCategories] = useState<any[] | null>(null);

    const validation = useJobValidation(formData, prompts);

    useEffect(() => {
        loadSiteData();
        loadPrompts();
        loadProviders();
        loadTopics();
        loadCategories();
    }, [siteId]);

    const loadSiteData = async () => {
        const siteData = await siteService.getSite(siteId);
        setSite(siteData);
    };

    const loadPrompts = async () => {
        const promptsData = await promptService.listPrompts();
        setPrompts(promptsData);
    };

    const loadProviders = async () => {
        const providersData = await providerService.listProviders();
        setProviders(providersData.filter(p => p.isActive));
    };

    const loadTopics = async () => {
        const strategy = (formData.topicStrategy as string) || DEFAULT_TOPIC_STRATEGY;
        const topicsData = await topicService.getSelectableTopics(siteId, strategy);
        setTopics(topicsData);
    };

    useEffect(() => {
        if (!siteId) return;
        loadTopics();
        updateFormData({ topics: [] as any });
    }, [formData.topicStrategy]);

    const loadCategories = async () => {
        const categoriesData = await categoryService.listSiteCategories(siteId);
        setCategories(categoriesData);
    };

    const handleSyncCategories = async () => {
        await execute(
            () => categoryService.syncFromWordPress(siteId),
            {
                successMessage: "Categories synced successfully",
                showSuccessToast: true,
                onSuccess: loadCategories
            }
        );
    };

    const handleSubmit = async () => {
        if (!formData.name || !formData.promptId || !formData.aiProviderId) {
            console.error("Please fill all required fields");
            return;
        }

        const result = await execute<void>(
            () => jobService.createJob(formData as any),
            {
                successMessage: "Job created successfully",
                showSuccessToast: true
            }
        );

        if (result !== null) {
            router.push(`/sites/jobs?id=${siteId}`);
        }
    };

    return (
        <div className="p-6 space-y-6 max-w-4xl mx-auto">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link href={`/sites/jobs?id=${siteId}`}>
                    <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                </Link>
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">Create New Job</h1>
                    <p className="text-muted-foreground">
                        Create automated content generation job for {site?.name}
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
                    site={site}
                />

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

export default function CreateSiteJobPage() {
    return (
        <Suspense fallback={
            <div className="p-6 space-y-6 max-w-4xl mx-auto">
                <div className="h-8 w-48 bg-muted/30 rounded animate-pulse" />
                <div className="h-64 bg-muted/30 rounded-lg animate-pulse" />
            </div>
        }>
            <CreateSiteJobPageContent />
        </Suspense>
    );
}
