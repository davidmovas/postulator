"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useApiCall } from "@/hooks/use-api-call";
import { siteService } from "@/services/sites";
import { Site } from "@/models/sites";
import { AlertTriangle, X } from "lucide-react";

interface DeleteSiteModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    site: Site | null;
    onSuccess?: () => void;
}

export function DeleteSiteModal({ open, onOpenChange, site, onSuccess }: DeleteSiteModalProps) {
    const { execute, isLoading } = useApiCall();
    const [confirmationText, setConfirmationText] = useState("");

    const resetForm = () => {
        setConfirmationText("");
    };

    const handleSubmit = async () => {
        if (!site || !isConfirmed) return;

        const result = await execute<void>(
            () => siteService.deleteSite(site.id),
            {
                onSuccess: () => {
                    onSuccess?.();
                },
                showSuccessToast: true,
                successMessage: "Site deleted successfully",
                errorTitle: "Failed to delete site"
            }
        );

        if (result !== null) {
            onOpenChange(false);
            resetForm();
        }
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!newOpen) {
            resetForm();
        }
        onOpenChange(newOpen);
    };

    const isConfirmed = confirmationText === site?.name;

    if (!site) return null;

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[440px]">
                <DialogHeader className="flex flex-row items-start gap-3">
                    <div className="flex-1 space-y-2">
                        <DialogTitle className="text-destructive text-lg">
                            Delete Site
                        </DialogTitle>
                        <DialogDescription className="text-sm leading-6">
                            This action cannot be undone. This will permanently delete the site{" "}
                            <span className="font-semibold text-foreground">{site.name}</span> and all associated data.
                        </DialogDescription>
                    </div>
                </DialogHeader>

                <div className="space-y-5 py-2">
                    <div className="bg-muted/40 border rounded-lg p-4 space-y-2">
                        <p className="font-semibold text-foreground">{site.url}</p>
                    </div>

                    <div className="space-y-3">
                        <div className="space-y-2">
                            <Label htmlFor="confirmation" className="text-sm font-medium">
                                To confirm, type site name below
                            </Label>
                            <Input
                                id="confirmation"
                                placeholder={`Type "${site.name}" to confirm`}
                                value={confirmationText}
                                onChange={(e) => setConfirmationText(e.target.value)}
                                disabled={isLoading}
                                className={
                                    !isConfirmed && confirmationText
                                        ? "border-destructive focus-visible:ring-destructive/50"
                                        : "focus-visible:ring-destructive/20"
                                }
                                autoComplete="off"
                                autoFocus
                            />
                        </div>
                    </div>
                </div>

                <DialogFooter className="flex flex-col-reverse sm:flex-row gap-2">
                    <Button
                        variant="outline"
                        onClick={() => handleOpenChange(false)}
                        disabled={isLoading}
                        className="sm:flex-1"
                    >
                        Cancel
                    </Button>
                    <Button
                        variant="destructive"
                        onClick={handleSubmit}
                        disabled={!isConfirmed || isLoading}
                        className="sm:flex-1"
                    >
                        {isLoading ? (
                            <>
                                <div className="h-4 w-4 border-2 border-current border-t-transparent rounded-full animate-spin mr-2" />
                                Deleting...
                            </>
                        ) : (
                            "Delete Site"
                        )}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}