"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter } from "next/navigation";
import { useQueryId } from "@/hooks/use-query-param";
import { Button } from "@/components/ui/button";
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

function EditJobPageContent() {
    const router = useRouter();
    const jobId = useQueryId();

    const [job, setJob] = useState<any>(null);
    const { formData, isLoading, updateFormData, execute } = useJobForm({ initialData: job });

    const [site, setSite] = useState<any>(null);
    const [sites, setSites] = useState<any[]>([]);
    const [prompts, setPrompts] = useState<any[] | null>(null);
    const [providers, setProviders] = useState<any[] | null>(null);
    const [topics, setTopics] = useState<any[] | null>(null);
    const [categories, setCategories] = useState<any[] | null>(null);

    const validation = useJobValidation(formData, prompts);

    useEffect(() => {
        loadJob();
    }, [jobId]);

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
        const siteId = formData.siteId || job?.siteId;
        if (siteId) {
            const strategy = (formData.topicStrategy as string) || DEFAULT_TOPIC_STRATEGY;
            // Use current jobId to exclude topics from OTHER unique-strategy jobs (but allow this job's topics)
            const topicsData = await topicService.getSelectableTopicsForJob(siteId, strategy, jobId);
            setTopics(topicsData);
        }
    };

    const loadCategories = async () => {
        const siteId = formData.siteId || job?.siteId;
        if (siteId) {
            const categoriesData = await categoryService.listSiteCategories(siteId);
            setCategories(categoriesData);
        }
    };

    const handleSyncCategories = async () => {
        const siteId = formData.siteId || job?.siteId;
        if (!siteId) return;

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
        if (!formData.name || !formData.promptId || !formData.aiProviderId || !formData.siteId) {
            console.error("Please fill all required fields");
            return;
        }

        const result = await execute<void>(
            () => jobService.updateJob({ ...formData, id: jobId } as any),
            {
                successMessage: "Job updated successfully",
                showSuccessToast: true
            }
        );

        if (result !== null) {
            router.push("/jobs");
        }
    };

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
                    site={site}
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
                    initialScheduleType={job.schedule?.type}
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
                        {isLoading ? "Saving..." : "Save Changes"}
                    </Button>
                </div>
            </div>
        </div>
    );
}

export default function EditJobPage() {
    return (
        <Suspense fallback={
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
        }>
            <EditJobPageContent />
        </Suspense>
    );
}
