"use client";

import { useEffect, useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { useApiCall } from "@/hooks/use-api-call";
import { useToast } from "@/components/ui/use-toast";
import { importerService } from "@/services/importer";
import { UploadIcon, XIcon, PaperclipIcon } from "lucide-react";
import { OnFileDrop, OnFileDropOff } from "@/wailsjs/wailsjs/runtime/runtime";

interface ImportTopicsModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: number;
    onSuccess?: () => void;
}

const ALLOWED_EXTENSIONS = ["txt", "csv", "xlsx", "json"] as const;

function validateFileExtension(path: string): boolean {
    const ext = path.split('.').pop()?.toLowerCase();
    return !!ext && ALLOWED_EXTENSIONS.includes(ext as any);
}

export function ImportTopicsModal({ open, onOpenChange, siteId, onSuccess }: ImportTopicsModalProps) {
    const { execute, isLoading } = useApiCall();
    const { toast } = useToast();

    const [filePath, setFilePath] = useState("");

    // Prevent browser from opening the dropped file while modal is open
    useEffect(() => {
        if (!open) return;
        const prevent = (e: DragEvent) => {
            e.preventDefault();
            e.stopPropagation();
        };
        window.addEventListener("dragover", prevent as any);
        window.addEventListener("drop", prevent as any);
        return () => {
            window.removeEventListener("dragover", prevent as any);
            window.removeEventListener("drop", prevent as any);
        };
    }, [open]);

    // Subscribe to Wails global desktop file drop to get a real path
    useEffect(() => {
        if (!open) return;
        let off: (() => void) | null = null;
        try {
            off = OnFileDrop((_, __, paths: string[]) => {
                if (!paths || paths.length === 0) return;
                const path = paths[0];
                if (!validateFileExtension(path)) {
                    toast({
                        title: "Invalid file type",
                        description: `Allowed: ${ALLOWED_EXTENSIONS.join(', ')}`,
                        variant: "destructive",
                    });
                    return;
                }
                setFilePath(path);
            }, false);
        } catch (e) {
            // noop if not running in desktop
        }
        return () => {
            try { OnFileDropOff(); } catch {}
            if (typeof off === 'function') {
                try { off(); } catch {}
            }
        };
    }, [open, toast]);

    const reset = () => setFilePath("");

    const handleSubmit = async () => {
        const path = filePath.trim();
        if (!path) {
            toast({ title: "No file selected", description: "Drop a file to select it", variant: "destructive" });
            return;
        }
        if (!validateFileExtension(path)) {
            toast({ title: "Invalid file type", description: `Allowed: ${ALLOWED_EXTENSIONS.join(', ')}`, variant: "destructive" });
            return;
        }

        const result = await execute(() => importerService.importAndAssignToSite(path, siteId), {
            errorTitle: "Failed to import topics",
            showSuccessToast: false,
        });

        if (result) {
            const description = `Added: ${result.totalAdded} | Skipped: ${result.totalSkipped}`;

            toast({
                title: "Import result",
                description,
                variant: "success",
            });

            onOpenChange(false);
            reset();
            onSuccess?.();
        }
    };

    const handleOpenChange = (newOpen: boolean) => {
        if (!newOpen) {
            reset();
        }
        onOpenChange(newOpen);
    };

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-[600px]">
                <DialogHeader>
                    <DialogTitle>Import Topics</DialogTitle>
                    <DialogDescription>
                        Drag & drop a file anywhere in the app or into the zone below. Supported: {ALLOWED_EXTENSIONS.join(', ')}
                    </DialogDescription>
                </DialogHeader>

                <div className="flex flex-col gap-3 py-2">
                    {/* Drop Zone */}
                    <div
                        className="border border-dashed rounded-xl p-6 min-h-32 flex items-center justify-center text-center bg-accent/20 hover:bg-accent/30 transition-colors"
                    >
                        {!filePath ? (
                            <div className="flex flex-col items-center gap-2">
                                <div className="bg-background flex size-11 items-center justify-center rounded-full border" aria-hidden="true">
                                    <UploadIcon className="size-4 opacity-60" />
                                </div>
                                <p className="text-sm font-medium">Drop your file here</p>
                                <p className="text-muted-foreground text-xs">Supported: {ALLOWED_EXTENSIONS.join(', ')}</p>
                            </div>
                        ) : (
                            <div className="flex items-center justify-between w-full gap-3">
                                <div className="flex items-center gap-2 min-w-0">
                                    <PaperclipIcon className="size-4 opacity-60 shrink-0" />
                                    <div className="min-w-0">
                                        <div className="text-sm font-medium">File selected</div>
                                        <div className="text-xs text-muted-foreground truncate">{filePath.replace(/^.*[\\\\\/]/, '')}</div>
                                    </div>
                                </div>
                                <Button size="icon" variant="ghost" onClick={reset} disabled={isLoading} aria-label="Clear file">
                                    <XIcon className="size-4" />
                                </Button>
                            </div>
                        )}
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
                        disabled={isLoading || !filePath.trim()}
                        className="bg-violet-600 hover:bg-violet-700 text-white"
                    >
                        {isLoading ? "Importing..." : "Import"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
