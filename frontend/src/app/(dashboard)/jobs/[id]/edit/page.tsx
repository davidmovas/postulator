"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { jobService } from "@/services/jobs";
import { promptService } from "@/services/prompts";
import { siteService } from "@/services/sites";
import { topicService } from "@/services/topics";
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

export default function EditJobPage() {
    const params = useParams();
    const router = useRouter();
    const jobId = parseInt(params.id as string);

    const [job, setJob] = useState<any>(null);
    const { formData, isLoading, updateFormData, execute } = useJobForm({ initialData: job });

    const [site, setSite] = useState<any>(null);
    const [sites, setSites] = useState<any[]>([]);
    const [prompts, setPrompts] = useState<any[]>([]);
    const [providers, setProviders] = useState<any[]>([]);
    const [topics, setTopics] = useState<any[]>([]);
    const [categories, setCategories] = useState<any[]>([]);

    // Загрузка данных джобы
    useEffect(() => {
        loadJob();
    }, [jobId]);

    // Загрузка зависимых данных
    useEffect(() => {
        if (job) {
            loadSiteData();
            loadPrompts();
            loadProviders();
            loadTopics();
            loadCategories();
        }
    }, [job]);

    const loadJob = async () => {
        try {
            const jobData = await jobService.getJob(jobId);
            setJob(jobData);
        } catch (error) {
            console.error("Failed to load job:", error);
        }
    };

    const loadSiteData = async () => {
        if (job?.siteId) {
            const siteData = await siteService.getSite(job.siteId);
            setSite(siteData);
        }
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
        if (job?.siteId) {
            const topicsData = await topicService.getSiteTopics(job.siteId);
            setTopics(topicsData);
        }
    };

    const loadCategories = async () => {
        if (job?.siteId) {
            const categoriesData = await categoryService.listSiteCategories(job.siteId);
            setCategories(categoriesData);
        }
    };

    const handleSyncCategories = async () => {
        if (!job?.siteId) return;

        await execute(
            () => categoryService.syncFromWordPress(job.siteId),
            {
                successMessage: "Categories synced successfully",
                showSuccessToast: true,
                onSuccess: loadCategories
            }
        );
    };

    const handleSubmit = async () => {
        if (!formData.name || !formData.promptId || !formData.aiProviderId || !formData.siteId) {
            console.error("Please fill all required fields");
            return;
        }

        const result = await execute<void>(
            () => jobService.updateJob(formData as any),
            {
                successMessage: "Job updated successfully",
                showSuccessToast: true
            }
        );

        if (result !== undefined) {
            router.push("/jobs");
        }
    };

    const isFormValid = formData.name && formData.promptId && formData.aiProviderId && formData.siteId;

    if (!job) {
        return (
            <div className="p-6">
                <div className="animate-pulse">
                    <div className="h-8 bg-muted rounded w-1/3 mb-4"></div>
                    <div className="h-4 bg-muted rounded w-1/2 mb-8"></div>
                    <div className="space-y-4">
                        <div className="h-32 bg-muted rounded"></div>
                        <div className="h-32 bg-muted rounded"></div>
                        <div className="h-32 bg-muted rounded"></div>
                    </div>
                </div>
            </div>
        );
    }

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
                    <h1 className="text-3xl font-bold tracking-tight">Edit Job</h1>
                    <p className="text-muted-foreground">
                        Update automated content generation job
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
                    disabled={!isFormValid || isLoading}
                >
                    {isLoading ? "Updating..." : "Update Job"}
                </Button>
            </div>
        </div>
    );
}