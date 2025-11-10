"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useApiCall } from "@/hooks/use-api-call";
import { jobService } from "@/services/jobs";
import { promptService } from "@/services/prompts";
import { siteService } from "@/services/sites";
import { topicService } from "@/services/topics";
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

export default function CreateGlobalJobPage() {
    const router = useRouter();
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<Partial<JobCreateInput>>({
        topicStrategy: "unique",
        categoryStrategy: "fixed",
        requiresValidation: false,
        jitterEnabled: true,
        jitterMinutes: 30,
        placeholdersValues: {},
        categories: [],
        topics: []
    });

    const [sites, setSites] = useState<any[]>([]);
    const [prompts, setPrompts] = useState<any[]>([]);
    const [providers, setProviders] = useState<any[]>([]);
    const [topics, setTopics] = useState<any[]>([]);
    const [categories, setCategories] = useState<any[]>([]);

    useEffect(() => {
        loadSites();
        loadPrompts();
        loadProviders();
        loadTopics();
    }, []);

    useEffect(() => {
        if (formData.siteId) {
            loadCategories();
            loadSiteTopics();
        }
    }, [formData.siteId]);

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

    const loadTopics = async () => {
        const topicsData = await topicService.listTopics();
        setTopics(topicsData);
    };

    const loadSiteTopics = async () => {
        if (!formData.siteId) return;
        const topicsData = await topicService.getSiteTopics(formData.siteId);
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
            <div className="flex justify-end gap-3 pt-6 border-t">
                <Button
                    variant="outline"
                    onClick={() => router.push("/jobs")}
                    disabled={isLoading}
                >
                    Cancel
                </Button>
                <Button
                    onClick={handleSubmit}
                    disabled={isLoading || !formData.siteId || !formData.name || !formData.promptId || !formData.aiProviderId}
                >
                    {isLoading ? "Creating..." : "Create Job"}
                </Button>
            </div>
        </div>
    );
}