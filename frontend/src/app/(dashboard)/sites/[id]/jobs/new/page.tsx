"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
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

export default function CreateSiteJobPage() {
    const params = useParams();
    const router = useRouter();
    const siteId = parseInt(params.id as string);

    const { formData, isLoading, updateFormData } = useJobForm({ siteId });
    const { execute } = useApiCall();

    const [site, setSite] = useState<any>(null);
    const [prompts, setPrompts] = useState<any[] | null>(null);
    const [providers, setProviders] = useState<any[] | null>(null);
    const [topics, setTopics] = useState<any[] | null>(null);
    const [categories, setCategories] = useState<any[] | null>(null);
    
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
            router.push(`/sites/${siteId}/jobs`);
        }
    };

    const isFormValid = formData.name && formData.promptId && formData.aiProviderId;

    return (
        <div className="p-6 space-y-6 max-w-4xl mx-auto">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Link href={`/sites/${siteId}/jobs`}>
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
            <div className="flex justify-end gap-3 pt-6 border-t">
                <Button
                    variant="outline"
                    onClick={() => router.push(`/sites/${siteId}/jobs`)}
                    disabled={isLoading}
                >
                    Cancel
                </Button>
                <Button
                    onClick={handleSubmit}
                    disabled={!isFormValid || isLoading}
                >
                    {isLoading ? "Creating..." : "Create Job"}
                </Button>
            </div>
        </div>
    );
}