"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { Site } from "@/models/sites";
import { siteService } from "@/services/sites";
import { useApiCall } from "@/hooks/use-api-call";
import { useContextModal } from "@/context/modal-context";
import { ConfirmationModal } from "@/modals/confirm-modal";
import { EditSiteModal } from "@/components/sites/modals/edit-site-modal";
import { ChangePasswordModal } from "@/components/sites/modals/reset-site-password-modal";
import { SiteHeader } from "@/components/sites/site-header";
import { SiteActions } from "@/components/sites/site-actions";
import { SiteInfo } from "@/components/sites/site-info";

export default function SitePage() {
    const params = useParams();
    const router = useRouter();
    const { execute, isLoading } = useApiCall();
    const { confirmationModal } = useContextModal();

    const [site, setSite] = useState<Site | null>(null);
    const [editModalOpen, setEditModalOpen] = useState(false);
    const [passwordModalOpen, setPasswordModalOpen] = useState(false);

    const siteId = parseInt(params.id as string);

    const loadSite = async () => {
        const result = await execute<Site>(
            () => siteService.getSite(siteId),
            {
                errorTitle: "Failed to load site"
            }
        );

        if (result) {
            setSite(result);
        }
    };

    useEffect(() => {
        if (siteId) {
            loadSite();
        }
    }, [siteId]);

    const handleCheckHealth = async () => {
        await execute<string>(
            () => siteService.checkHealth(siteId),
            {
                successMessage: "Health check completed",
                showSuccessToast: true,
                onSuccess: () => {
                    setTimeout(() => loadSite(), 1000);
                }
            }
        );
    };

    const handleOpenWordPress = () => {
        if (site) {
            window.open(site.url + '/wp-admin', '_blank');
        }
    };

    const handleViewArticles = () => {
        router.push(`/sites/${siteId}/articles`);
    };

    const handleViewJobs = () => {
        router.push(`/sites/${siteId}/jobs`);
    };

    const handleViewTopics = () => {
        router.push(`/sites/${siteId}/topics`);
    }

    const handleViewCategories = () => {
        router.push(`/sites/${siteId}/categories`);
    }

    const handleDelete = () => {
        confirmationModal.open({
            title: "Delete Site",
            description: (
                <div className="space-y-3">
                    <p className="text-sm leading-6">
                        Are you sure you want to delete this site?
                    </p>
                    <div className="bg-muted/50 border rounded-lg p-3">
                        <p className="font-medium text-muted-foreground">{site?.name}</p>
                    </div>
                    <p className="text-xs text-muted-foreground">
                        This action cannot be undone.
                    </p>
                </div>
            ),
            confirmText: "Delete",
            cancelText: "Cancel",
            variant: "destructive",
            onConfirm: async () => {
                await execute<void>(
                    () => siteService.deleteSite(siteId),
                    {
                        successMessage: "Site deleted successfully",
                        showSuccessToast: true,
                        onSuccess: () => {
                            router.push("/sites");
                        }
                    }
                );
            }
        });
    };

    const handleSuccess = () => {
        loadSite();
    };

    if (isLoading && !site) {
        return (
            <div className="p-6 space-y-6">
                <div className="h-32 bg-muted/30 rounded-lg animate-pulse" />
                <div className="h-64 bg-muted/30 rounded-lg animate-pulse" />
            </div>
        );
    }

    if (!site) {
        return (
            <div className="p-6">
                <div className="text-center py-8">
                    <h2 className="text-2xl font-bold text-destructive">Site not found</h2>
                    <p className="text-muted-foreground mt-2">The requested site could not be loaded.</p>
                </div>
            </div>
        );
    }

    return (
        <div className="p-6 space-y-6">
            <SiteHeader site={site} />

            <SiteActions
                onCheckHealth={handleCheckHealth}
                onEdit={() => setEditModalOpen(true)}
                onChangePassword={() => setPasswordModalOpen(true)}
                onViewArticles={handleViewArticles}
                onViewJobs={handleViewJobs}
                onViewTopics={handleViewTopics}
                onViewCategories={handleViewCategories}
                onOpenWordPress={handleOpenWordPress}
                onDelete={handleDelete}
                isLoading={isLoading}
            />

            <SiteInfo site={site} />

            <EditSiteModal
                open={editModalOpen}
                onOpenChange={setEditModalOpen}
                site={site}
                onSuccess={handleSuccess}
            />

            <ChangePasswordModal
                open={passwordModalOpen}
                onOpenChange={setPasswordModalOpen}
                siteId={siteId}
                onSuccess={handleSuccess}
            />

            <ConfirmationModal
                open={confirmationModal.isOpen}
                onOpenChange={confirmationModal.close}
                data={confirmationModal.data}
            />
        </div>
    );
}