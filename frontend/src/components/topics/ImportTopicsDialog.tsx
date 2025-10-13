"use client";

import React, { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useErrorHandling } from "@/lib/error-handling";
import { importAndAssignToSite, importTopics, ImportResult } from "@/services/topic";
import { syncCategories } from "@/services/site";
import { ALLOWED_IMPORT_EXTENSIONS, DEFAULT_TOPIC_STRATEGY, TOPIC_STRATEGIES } from "@/constants/topics";
import { RefreshCw, Upload, FileCheck, X } from "lucide-react";
import { OnFileDrop, OnFileDropOff } from "@/wailsjs/wailsjs/runtime/runtime";

const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB limit

const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / 1024 / 1024).toFixed(1) + " MB";
};

const validateFileExtension = (path: string): boolean => {
    const ext = path.split('.').pop()?.toLowerCase();
    return ext ? (ALLOWED_IMPORT_EXTENSIONS as readonly string[]).includes(ext) : false;
};

const getFileSizeFromPath = async (path: string): Promise<number> => {
    // In a real implementation, you might want to call a backend method to get file size
    // For now, returning 0 as placeholder
    return 0;
};

const formatImportResult = (result: ImportResult): string => {
    const parts = [
        `Total read: ${result.totalRead}`,
        `Added: ${result.totalAdded}`,
        `Skipped: ${result.totalSkipped}`
    ];

    if (result.errors && result.errors.length > 0) {
        parts.push(`Errors: ${result.errors.join(", ")}`);
    }

    return parts.join(" | ");
};

interface FileDropZoneProps {
    filePath: string;
    fileName: string;
    fileSize: number;
    onFileSelect: (path: string) => void;
    onClear: () => void;
    isDisabled?: boolean;
}

function FileDropZone({ filePath, fileName, fileSize, onFileSelect, onClear, isDisabled }: FileDropZoneProps) {
    useEffect(() => {
        if (isDisabled) return;

        let isActive = true;

        const dropHandler = (x: number, y: number, paths: string[]) => {
            if (!isActive || !paths || paths.length === 0) return;
            onFileSelect(paths[0]);
        };

        try {
            OnFileDrop(dropHandler, false);
        } catch (error) {
            console.error("Failed to setup OnFileDrop:", error);
        }

        return () => {
            isActive = false;
            try {
                OnFileDropOff();
            } catch (error) {
                console.error("Failed to cleanup OnFileDrop:", error);
            }
        };
    }, [isDisabled, onFileSelect]);

    return (
        <div className="space-y-2">
            <Label>File</Label>
            <div className="border-2 border-dashed rounded-lg p-6 text-sm transition-colors hover:border-primary/50 hover:bg-accent/20">
                {filePath ? (
                    <div className="flex items-start gap-3">
                        <FileCheck className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                        <div className="flex-1 min-w-0">
                            <div className="text-foreground font-medium truncate">{fileName}</div>
                        </div>
                        <Button
                            size="sm"
                            variant="ghost"
                            onClick={onClear}
                            className="flex-shrink-0"
                            disabled={isDisabled}
                        >
                            <X className="h-4 w-4" />
                        </Button>
                    </div>
                ) : (
                    <div className="flex flex-col items-center justify-center py-4 text-center space-y-2">
                        <Upload className="h-10 w-10 text-muted-foreground/50" />
                        <div>
                            <div className="text-foreground font-medium mb-1">
                                Drag and drop a file anywhere in the app
                            </div>
                            <div className="text-xs text-muted-foreground">
                                Supported: {ALLOWED_IMPORT_EXTENSIONS.join(", ")}
                            </div>
                            <div className="text-xs text-muted-foreground mt-1">
                                Maximum size: {MAX_FILE_SIZE / 1024 / 1024}MB
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}

// Component for importing topics and assigning to a site
export interface ImportAndAssignTopicsDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: number | null;
    onImported?: () => void | Promise<void>;
}

export function ImportAndAssignTopicsDialog({ open, onOpenChange, siteId, onImported }: ImportAndAssignTopicsDialogProps) {
    const { withErrorHandling, showError, showSuccess } = useErrorHandling();

    const [filePath, setFilePath] = useState<string>("");
    const [fileName, setFileName] = useState<string>("");
    const [fileSize, setFileSize] = useState<number>(0);

    // TODO: Category selection is temporary - defaults to 1 until full category implementation
    const [categoryId, setCategoryId] = useState<string>("1");
    const [strategy, setStrategy] = useState<string>(DEFAULT_TOPIC_STRATEGY);
    const [isImporting, setIsImporting] = useState(false);

    const reset = () => {
        setFilePath("");
        setFileName("");
        setFileSize(0);
        setCategoryId("1");
        setStrategy(DEFAULT_TOPIC_STRATEGY);
    };

    const handleFileSelect = async (path: string) => {
        if (!validateFileExtension(path)) {
            showError(`Invalid file type. Allowed: ${ALLOWED_IMPORT_EXTENSIONS.join(", ")}`);
            return;
        }

        const name = path.split(/[\\/]/).pop() || path;
        const size = await getFileSizeFromPath(path);

        setFilePath(path);
        setFileName(name);
        setFileSize(size);
    };

    const handleImport = async () => {
        if (!siteId || !filePath.trim()) {
            showError("Please select a file and ensure site is selected");
            return;
        }

        setIsImporting(true);

        try {
            const result = await importAndAssignToSite(
                filePath.trim(),
                siteId,
                parseInt(categoryId, 10) || 1,
                strategy
            );

            if (result.errors && result.errors.length > 0) {
                showError(`Import completed with errors: ${formatImportResult(result)}`);
            } else {
                showSuccess(`Topics imported successfully! ${formatImportResult(result)}`);
            }

            onOpenChange(false);
            reset();
            if (onImported) await onImported();
        } catch (error) {
            showError(`Import failed: ${error}`);
        } finally {
            setIsImporting(false);
        }
    };

    const handleSyncCategories = async () => {
        if (!siteId) return;

        await withErrorHandling(
            async () => {
                await syncCategories(siteId);
            },
            {
                successMessage: "Categories synced successfully",
                showSuccess: true
            }
        );
    };

    return (
        <Dialog
            open={open}
            onOpenChange={(o) => {
                if (!o) {
                    onOpenChange(false);
                    reset();
                } else {
                    onOpenChange(true);
                }
            }}
        >
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Import and Assign Topics</DialogTitle>
                    <DialogDescription>
                        Import topics from a file and assign them to this site.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 pt-2">
                    <FileDropZone
                        filePath={filePath}
                        fileName={fileName}
                        fileSize={fileSize}
                        onFileSelect={handleFileSelect}
                        onClear={reset}
                        isDisabled={isImporting}
                    />

                    {/* Category + Sync in one row */}
                    <div className="flex items-end gap-3">
                        <div className="flex-1 space-y-2">
                            <Label htmlFor="category">
                                Category
                                <span className="text-xs text-muted-foreground ml-2">(temporary - defaults to 1)</span>
                            </Label>
                            <Input
                                id="category"
                                type="number"
                                min="1"
                                value={categoryId}
                                onChange={(e) => setCategoryId(e.target.value)}
                                disabled={isImporting}
                            />
                        </div>
                        <Button
                            variant="secondary"
                            onClick={handleSyncCategories}
                            disabled={!siteId || isImporting}
                        >
                            <RefreshCw className="h-4 w-4 mr-2" />
                            Sync Categories
                        </Button>
                    </div>

                    {/* Strategy */}
                    <div className="space-y-2">
                        <Label htmlFor="strategy">Import Strategy</Label>
                        <select
                            id="strategy"
                            value={strategy}
                            onChange={(e) => setStrategy(e.target.value)}
                            disabled={isImporting}
                            className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                            {TOPIC_STRATEGIES.map((s) => (
                                <option key={s} value={s} className="capitalize">
                                    {s}
                                </option>
                            ))}
                        </select>
                    </div>
                </div>

                <DialogFooter className="pt-4">
                    <Button
                        variant="ghost"
                        onClick={() => {
                            onOpenChange(false);
                            reset();
                        }}
                        disabled={isImporting}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleImport}
                        disabled={!filePath.trim() || !siteId || isImporting}
                    >
                        {isImporting ? "Importing..." : "Import & Assign"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

// Component for just importing topics (without site assignment)
export interface ImportTopicsDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onImported?: () => void | Promise<void>;
}

export function ImportTopicsDialog({ open, onOpenChange, onImported }: ImportTopicsDialogProps) {
    const { showError, showSuccess } = useErrorHandling();

    const [filePath, setFilePath] = useState<string>("");
    const [fileName, setFileName] = useState<string>("");
    const [fileSize, setFileSize] = useState<number>(0);
    const [isImporting, setIsImporting] = useState(false);

    const reset = () => {
        setFilePath("");
        setFileName("");
        setFileSize(0);
    };

    const handleFileSelect = async (path: string) => {
        if (!validateFileExtension(path)) {
            showError(`Invalid file type. Allowed: ${ALLOWED_IMPORT_EXTENSIONS.join(", ")}`);
            return;
        }

        const name = path.split(/[\\/]/).pop() || path;
        const size = await getFileSizeFromPath(path);

        setFilePath(path);
        setFileName(name);
        setFileSize(size);
    };

    const handleImport = async () => {
        if (!filePath.trim()) {
            showError("Please select a file");
            return;
        }

        setIsImporting(true);

        try {
            const result = await importTopics(filePath.trim());

            if (result.errors && result.errors.length > 0) {
                showError(`Import completed with errors: ${formatImportResult(result)}`);
            } else {
                showSuccess(`Topics imported successfully! ${formatImportResult(result)}`);
            }

            onOpenChange(false);
            reset();
            if (onImported) await onImported();
        } catch (error) {
            showError(`Import failed: ${error}`);
        } finally {
            setIsImporting(false);
        }
    };

    return (
        <Dialog
            open={open}
            onOpenChange={(o) => {
                if (!o) {
                    onOpenChange(false);
                    reset();
                } else {
                    onOpenChange(true);
                }
            }}
        >
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Import Topics</DialogTitle>
                    <DialogDescription>
                        Import topics from a file to add them to your topics library.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 pt-2">
                    <FileDropZone
                        filePath={filePath}
                        fileName={fileName}
                        fileSize={fileSize}
                        onFileSelect={handleFileSelect}
                        onClear={reset}
                        isDisabled={isImporting}
                    />
                </div>

                <DialogFooter className="pt-4">
                    <Button
                        variant="ghost"
                        onClick={() => {
                            onOpenChange(false);
                            reset();
                        }}
                        disabled={isImporting}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleImport}
                        disabled={!filePath.trim() || isImporting}
                    >
                        {isImporting ? "Importing..." : "Import Topics"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

export default ImportAndAssignTopicsDialog;