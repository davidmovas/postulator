"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Site } from "@/models/sites";
import { useApiCall } from "@/hooks/use-api-call";
import { siteService } from "@/services/sites";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

interface EditSiteModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    site: Site | null;
    onSuccess?: () => void;
}

interface EditSiteFormData {
    id: number;
    name: string;
    url: string;
    wpUsername: string;
    status: string;
    wpPassword?: string;
    autoHealthCheck?: boolean;
}

export function EditSiteModal({ open, onOpenChange, site, onSuccess }: EditSiteModalProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<EditSiteFormData>({
        id: 0,
        name: "",
        url: "",
        wpUsername: "",
        status: "active",
        autoHealthCheck: site?.autoHealthCheck || false,
    });

    const resetForm = () => {
        setFormData({
            id: 0,
            name: "",
            url: "",
            wpUsername: "",
            status: "active",
            autoHealthCheck: site?.autoHealthCheck || false,
        })
    }

    useEffect(() => {
        if (site) {
            setFormData({
                id: site.id,
                name: site.name || "",
                url: site.url || "",
                wpUsername: site.wpUsername || "",
                status: site.status || "active"
            });
        }
    }, [site, open]);

    const isFormValid = formData.name.trim() &&
        formData.url.trim() &&
        formData.wpUsername.trim();

    const handleSubmit = async () => {
        if (!isFormValid || !site) return;

        const result = await execute<void>(
            () => siteService.updateSite(formData),
            {
                successMessage: "Site updated successfully",
                showSuccessToast: true
            }
        );

        if (result !== null) {
            onOpenChange(false);
            resetForm();
            onSuccess?.();
        }
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!newOpen) {
            resetForm();
        }
        onOpenChange(newOpen);
    };

    if (!site) return null;

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Edit Site</DialogTitle>
                    <DialogDescription>
                        Update your WordPress site information.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="edit-url">Site URL</Label>
                        <Input
                            id="edit-url"
                            value={formData.url}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                url: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="edit-name">Site Name</Label>
                        <Input
                            id="edit-name"
                            value={formData.name}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                name: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="edit-wpUsername">WordPress Username</Label>
                        <Input
                            id="edit-wpUsername"
                            value={formData.wpUsername}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                wpUsername: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="edit-status">Status</Label>
                        <Select
                            value={formData.status}
                            onValueChange={(value) => setFormData(prev => ({
                                ...prev,
                                status: value
                            }))}
                            disabled={isLoading}
                        >
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder="Select status" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="active">Active</SelectItem>
                                <SelectItem value="inactive">Inactive</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    <div className="flex items-center space-x-2">
                        <Switch
                            id="edit-autoCheckHealth"
                            checked={formData.autoHealthCheck}
                            onCheckedChange={(checked) => setFormData(prev => ({
                                ...prev,
                                autoHealthCheck: checked
                            }))}
                            disabled={isLoading}
                        />
                        <Label htmlFor="edit-autoCheckHealth" className="cursor-pointer">
                            Enable auto health check
                        </Label>
                    </div>
                </div>

                <DialogFooter>
                    <Button
                        variant="outline"
                        onClick={() => handleOpenChange(false)}
                        disabled={isLoading}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSubmit}
                        disabled={!isFormValid || isLoading}
                    >
                        {isLoading ? "Updating..." : "Update Site"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}