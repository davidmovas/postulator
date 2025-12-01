"use client";

import { useEffect, useState, useCallback, useRef } from "react";
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
import { dialogsService } from "@/services/dialogs";
import { UploadIcon, XIcon, FileTextIcon, FolderOpenIcon, Loader2 } from "lucide-react";
import { OnFileDrop, OnFileDropOff } from "@/wailsjs/wailsjs/runtime/runtime";
import { cn } from "@/lib/utils";

interface ImportTopicsModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: number;
    onSuccess?: () => void;
}

const ALLOWED_EXTENSIONS = ["txt", "csv", "xlsx", "json"] as const;
const FILE_FILTERS = [
    { displayName: "All Supported Files", pattern: "*.txt;*.csv;*.xlsx;*.json" },
    { displayName: "Text Files", pattern: "*.txt" },
    { displayName: "CSV Files", pattern: "*.csv" },
    { displayName: "Excel Files", pattern: "*.xlsx" },
    { displayName: "JSON Files", pattern: "*.json" },
];

function validateFileExtension(path: string): boolean {
    const ext = path.split('.').pop()?.toLowerCase();
    return !!ext && ALLOWED_EXTENSIONS.includes(ext as typeof ALLOWED_EXTENSIONS[number]);
}

function getFileName(path: string): string {
    return path.replace(/^.*[\\\/]/, '');
}

function getFileExtension(path: string): string {
    return path.split('.').pop()?.toUpperCase() || '';
}

export function ImportTopicsModal({ open, onOpenChange, siteId, onSuccess }: ImportTopicsModalProps) {
    const { execute, isLoading } = useApiCall();
    const { toast } = useToast();

    const [filePath, setFilePath] = useState("");
    const [dragActive, setDragActive] = useState(false);
    const [isSelectingFile, setIsSelectingFile] = useState(false);
    const dropZoneRef = useRef<HTMLDivElement>(null);

    // Prevent default drag behavior on window
    useEffect(() => {
        if (!open) return;
        const prevent = (e: DragEvent) => {
            e.preventDefault();
            e.stopPropagation();
        };
        window.addEventListener("dragover", prevent as EventListener);
        window.addEventListener("drop", prevent as EventListener);
        return () => {
            window.removeEventListener("dragover", prevent as EventListener);
            window.removeEventListener("drop", prevent as EventListener);
        };
    }, [open]);

    // Setup Wails file drop listener
    useEffect(() => {
        if (!open) return;
        try {
            OnFileDrop((_, __, paths: string[]) => {
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
                setDragActive(false);
            }, false);
        } catch {
            // noop if not running in desktop
        }
        return () => {
            try { OnFileDropOff(); } catch {}
        };
    }, [open, toast]);

    // Track drag state for visual feedback
    useEffect(() => {
        if (!open) return;

        const handleDragEnter = () => setDragActive(true);
        const handleDragLeave = (e: DragEvent) => {
            // Only set inactive if leaving the drop zone entirely
            if (e.relatedTarget === null) {
                setDragActive(false);
            }
        };

        window.addEventListener("dragenter", handleDragEnter);
        window.addEventListener("dragleave", handleDragLeave);

        return () => {
            window.removeEventListener("dragenter", handleDragEnter);
            window.removeEventListener("dragleave", handleDragLeave);
        };
    }, [open]);

    const reset = useCallback(() => {
        setFilePath("");
        setDragActive(false);
    }, []);

    const handleBrowseFiles = useCallback(async () => {
        setIsSelectingFile(true);
        try {
            const path = await dialogsService.openFileDialog("Select Topics File", FILE_FILTERS);
            if (path) {
                setFilePath(path);
            }
        } catch (err) {
            toast({
                title: "Error",
                description: "Failed to open file dialog",
                variant: "destructive",
            });
        } finally {
            setIsSelectingFile(false);
        }
    }, [toast]);

    const handleSubmit = async () => {
        const path = filePath.trim();
        if (!path) {
            toast({ title: "No file selected", description: "Select or drop a file", variant: "destructive" });
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
                title: "Import complete",
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
                        Drag & drop a file or click to browse. Supported formats: {ALLOWED_EXTENSIONS.join(', ')}
                    </DialogDescription>
                </DialogHeader>

                <div className="flex flex-col gap-4 py-2">
                    {/* Drop Zone / File Selected */}
                    <div
                        ref={dropZoneRef}
                        onClick={!filePath && !isLoading ? handleBrowseFiles : undefined}
                        className={cn(
                            "border-2 border-dashed rounded-xl p-6 min-h-40 flex flex-col items-center justify-center text-center transition-all duration-200",
                            !filePath && !isLoading && "cursor-pointer hover:border-primary/50 hover:bg-accent/30",
                            dragActive && "border-primary bg-primary/10 scale-[1.02]",
                            !dragActive && !filePath && "border-muted-foreground/25 bg-muted/20",
                            filePath && "border-green-500/50 bg-green-500/10",
                            isLoading && "opacity-60 cursor-not-allowed"
                        )}
                    >
                        {isSelectingFile ? (
                            <div className="flex flex-col items-center gap-3">
                                <Loader2 className="h-10 w-10 text-primary animate-spin" />
                                <p className="text-sm text-muted-foreground">Opening file browser...</p>
                            </div>
                        ) : !filePath ? (
                            <div className="flex flex-col items-center gap-3">
                                <div className={cn(
                                    "flex size-14 items-center justify-center rounded-full border-2 transition-colors",
                                    dragActive ? "border-primary bg-primary/10" : "border-muted-foreground/30 bg-background"
                                )}>
                                    <UploadIcon className={cn(
                                        "size-6 transition-colors",
                                        dragActive ? "text-primary" : "text-muted-foreground/60"
                                    )} />
                                </div>
                                <div>
                                    <p className="text-sm font-medium">
                                        {dragActive ? "Drop your file here" : "Drag & drop or click to browse"}
                                    </p>
                                    <p className="text-xs text-muted-foreground mt-1">
                                        Supports: {ALLOWED_EXTENSIONS.map(e => `.${e}`).join(', ')}
                                    </p>
                                </div>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        handleBrowseFiles();
                                    }}
                                    disabled={isLoading}
                                    className="mt-2"
                                >
                                    <FolderOpenIcon className="h-4 w-4 mr-2" />
                                    Browse Files
                                </Button>
                            </div>
                        ) : (
                            <div className="flex items-center justify-between w-full gap-4">
                                <div className="flex items-center gap-3 min-w-0 flex-1">
                                    <div className="flex size-12 items-center justify-center rounded-lg bg-green-500/20 border border-green-500/30 shrink-0">
                                        <FileTextIcon className="size-6 text-green-600 dark:text-green-400" />
                                    </div>
                                    <div className="min-w-0 flex-1">
                                        <p className="text-sm font-medium truncate">{getFileName(filePath)}</p>
                                        <p className="text-xs text-muted-foreground">
                                            {getFileExtension(filePath)} file selected
                                        </p>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 shrink-0">
                                    <Button
                                        size="sm"
                                        variant="outline"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            handleBrowseFiles();
                                        }}
                                        disabled={isLoading}
                                    >
                                        Change
                                    </Button>
                                    <Button
                                        size="icon"
                                        variant="ghost"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            reset();
                                        }}
                                        disabled={isLoading}
                                        className="h-8 w-8"
                                    >
                                        <XIcon className="size-4" />
                                    </Button>
                                </div>
                            </div>
                        )}
                    </div>

                    {/* File path display */}
                    {filePath && (
                        <p className="text-xs text-muted-foreground truncate px-1">
                            {filePath}
                        </p>
                    )}
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
                        {isLoading ? (
                            <>
                                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                                Importing...
                            </>
                        ) : (
                            "Import"
                        )}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
