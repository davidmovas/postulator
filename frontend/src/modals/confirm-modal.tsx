"use client";

import React, { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { AlertTriangle } from "lucide-react";
import { useApiCall } from "@/hooks/use-api-call";

export interface ConfirmationModalData {
    title: string;
    description: string | React.ReactNode;
    confirmText?: string;
    cancelText?: string;
    variant?: "default" | "destructive";
    onConfirm: () => void | Promise<void>;
}


export interface ConfirmationModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    data: {
        title: string;
        description: string | React.ReactNode;
        confirmText?: string;
        cancelText?: string;
        variant?: "default" | "destructive";
        onConfirm: () => void | Promise<void>;
    } | null;
}

export function ConfirmationModal({ open, onOpenChange, data }: ConfirmationModalProps) {
    const { execute, isLoading } = useApiCall();
    const [isConfirming, setIsConfirming] = useState(false);

    const handleConfirm = async () => {
        if (!data) return;
        setIsConfirming(true);
        try {
            await execute(
                async () => {
                    await data.onConfirm();
                },
                {
                    successMessage: "Action completed successfully",
                    showSuccessToast: true,
                    errorTitle: "Action failed"
                }
            );
            onOpenChange(false);
        } finally {
            setIsConfirming(false);
        }
    };

    const handleOpenChange = (nextOpen: boolean) => {
        // We only handle closing requests from the Dialog; opening is controlled by props
        if (!nextOpen) {
            onOpenChange(false);
        }
    };

    const isDestructive = data?.variant === "destructive";

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <div className="flex items-center gap-3">
                        {isDestructive && (
                            <AlertTriangle className="h-6 w-6 text-destructive" />
                        )}
                        <DialogTitle className={isDestructive ? "text-destructive" : ""}>
                            {data?.title || "Confirm action"}
                        </DialogTitle>
                    </div>
                    {data?.description && (
                        <div
                            className="text-muted-foreground text-sm"
                             data-slot="alert-dialog-description"
                        >
                            {data.description}
                        </div>
                    )}
                </DialogHeader>

                <DialogFooter className="gap-2">
                    <Button
                        variant="outline"
                        onClick={() => onOpenChange(false)}
                        disabled={isLoading || isConfirming}
                    >
                        {data?.cancelText || "Cancel"}
                    </Button>
                    <Button
                        variant={isDestructive ? "destructive" : "default"}
                        onClick={handleConfirm}
                        disabled={isLoading || isConfirming || !data}
                    >
                        {isConfirming ? "Processing..." : data?.confirmText || "Confirm"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}