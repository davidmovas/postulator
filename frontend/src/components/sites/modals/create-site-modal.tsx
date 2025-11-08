"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { SiteCreateInput } from "@/models/sites";
import { useApiCall } from "@/hooks/use-api-call";
import { siteService } from "@/services/sites";
import { Switch } from "@/components/ui/switch";

interface CreateSiteModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSuccess?: () => void;
}

export function CreateSiteModal({ open, onOpenChange, onSuccess }: CreateSiteModalProps) {
    const { execute, isLoading } = useApiCall();

    const [formData, setFormData] = useState<SiteCreateInput>({
        name: "",
        url: "",
        wpUsername: "",
        wpPassword: "",
        autoHealthCheck: false,
    });

    const [nameTouched, setNameTouched] = useState(false);

    const resetForm = () => {
        setFormData({
            name: "",
            url: "",
            wpUsername: "",
            wpPassword: "",
            autoHealthCheck: false,
        });
        setNameTouched(false);
    };

    const extractDomain = (url: string): string => {
        try {
            const hasProtocol = /^https?:\/\//i.test(url);
            const u = new URL(hasProtocol ? url : `http://${url}`);
            let host = u.hostname.toLowerCase();
            if (host.startsWith("www.")) host = host.slice(4);
            return host;
        } catch {
            return "";
        }
    };

    const normalizeUrl = (url: string): string => {
        if (!url) return url;
        try {
            const hasProtocol = /^https?:\/\//i.test(url);
            const u = new URL(hasProtocol ? url : `http://${url}`);
            return u.toString().replace(/\/$/, "");
        } catch {
            return url;
        }
    };

    const handleUrlChange = (url: string) => {
        const normalizedUrl = normalizeUrl(url);
        setFormData(prev => ({
            ...prev,
            url: normalizedUrl
        }));

        if (!nameTouched && !formData.name.trim()) {
            const domain = extractDomain(url);
            if (domain) {
                setFormData(prev => ({ ...prev, name: domain }));
            }
        }
    };

    const handleNameChange = (name: string) => {
        setFormData(prev => ({ ...prev, name }));
        setNameTouched(true);
    };

    const isFormValid = formData.url.trim() &&
        formData.name.trim() &&
        formData.wpUsername.trim();

    const handleSubmit = async () => {
        if (!isFormValid) return;

        const result = await execute<string>(
            () => siteService.createSite(formData),
            {
                successMessage: "Site created successfully",
                showSuccessToast: true
            }
        );

        if (result) {
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

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle>Create New Site</DialogTitle>
                    <DialogDescription>
                        Add a new WordPress site to manage. We`&apos;ll auto-fill the name from the URL.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="url">Site URL<span className="text-lg text-red-600">*</span></Label>
                        <Input
                            id="url"
                            placeholder="https://example.com"
                            value={formData.url}
                            onChange={(e) => handleUrlChange(e.target.value)}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="name">Site Name<span className="text-lg text-red-600">*</span></Label>
                        <Input
                            id="name"
                            placeholder="My WordPress Site"
                            value={formData.name}
                            onChange={(e) => handleNameChange(e.target.value)}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="wpUsername">WordPress Username<span className="text-lg text-red-600">*</span></Label>
                        <Input
                            id="wpUsername"
                            placeholder="admin"
                            value={formData.wpUsername}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                wpUsername: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="wpPassword">Application Password</Label>
                        <Input
                            id="wpPassword"
                            type="password"
                            placeholder="Enter password"
                            value={formData.wpPassword}
                            onChange={(e) => setFormData(prev => ({
                                ...prev,
                                wpPassword: e.target.value
                            }))}
                            disabled={isLoading}
                        />
                    </div>

                    <div className="flex items-center space-x-2">
                        <Switch
                            id="autoCheckHealth"
                            checked={formData.autoHealthCheck}
                            onCheckedChange={(checked) => setFormData(prev => ({
                                ...prev,
                                autoHealthCheck: checked
                            }))}
                            disabled={isLoading}
                        />
                        <Label htmlFor="autoCheckHealth" className="cursor-pointer">
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
                        {isLoading ? "Creating..." : "Create Site"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}