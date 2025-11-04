"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useApiCall } from "@/hooks/use-api-call";
import { siteService } from "@/services/sites";

interface ChangePasswordModalProps {
    open: boolean;

    onOpenChange: (open: boolean) => void;
    siteId: number;
    onSuccess?: () => void;
}

export function ChangePasswordModal({ open, onOpenChange, siteId, onSuccess }: ChangePasswordModalProps) {
    const { execute, isLoading } = useApiCall();

    const [password, setPassword] = useState("");

    const resetForm = () => {
        setPassword("");
    };

    const handleSubmit = async () => {
        if (!password.trim()) return;

        const result = await execute<void>(
            () => siteService.updateSitePassword(siteId, password),
            {
                successMessage: "Password updated successfully",
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

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[400px]">
                <DialogHeader>
                    <DialogTitle>Change WordPress Password</DialogTitle>
                    <DialogDescription>
                        Update the WordPress password for this site.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="password">New Application Password<span className="text-lg text-red-600">*</span></Label>
                        <Input
                            id="password"
                            type="password"
                            placeholder="Enter new password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            disabled={isLoading}
                        />
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
                        disabled={!password.trim() || isLoading}
                    >
                        {isLoading ? "Updating..." : "Update Password"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}