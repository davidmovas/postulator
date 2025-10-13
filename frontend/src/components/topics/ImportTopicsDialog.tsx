"use client";

import React, { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useErrorHandling } from "@/lib/error-handling";
import { importAndAssignToSite } from "@/services/topic";
import { syncCategories } from "@/services/site";
import { ALLOWED_IMPORT_EXTENSIONS, DEFAULT_TOPIC_STRATEGY, TOPIC_STRATEGIES } from "@/constants/topics";
import { RefreshCw, FolderOpen, FileCheck, X } from "lucide-react";

export interface ImportTopicsDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: number | null;
    onImported?: () => void | Promise<void>;
}

const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50MB limit

export function ImportTopicsDialog({ open, onOpenChange, siteId, onImported }: ImportTopicsDialogProps) {
    const { withErrorHandling, showError } = useErrorHandling();

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

    const validateFile = (file: File): boolean => {
        const ext = file.name.split('.').pop()?.toLowerCase();
        if (!ext || !(ALLOWED_IMPORT_EXTENSIONS as readonly string[]).includes(ext)) {
            showError(`Invalid file type. Allowed: ${ALLOWED_IMPORT_EXTENSIONS.join(", ")}`);
            return false;
        }

        if (file.size > MAX_FILE_SIZE) {
            showError(`File too large. Maximum size: ${MAX_FILE_SIZE / 1024 / 1024}MB`);
            return false;
        }

        return true;
    };

    const handleFileSelect = async (file: File) => {
        if (!validateFile(file)) {
            return;
        }

        setFileName(file.name);
        setFileSize(file.size);

        const nativePath = (file as any).path || (file as any).nativePath;

        if (nativePath) {
            setFilePath(nativePath);
        } else {
            // Fallback: use the file name as identifier
            // The backend should handle file resolution if full path is not available
            setFilePath(file.name);
            console.warn("Native file path not resolved, using file name as fallback");
        }
    };

    const handleImport = async () => {
        if (!siteId || !filePath.trim()) {
            showError("Please select a file and ensure site is selected");
            return;
        }

        setIsImporting(true);

        const ok = await withErrorHandling(
            async () => {
                await importAndAssignToSite(
                    filePath.trim(),
                    siteId,
                    parseInt(categoryId, 10) || 1,
                    strategy
                );
            },
            {
                successMessage: "Topics imported and assigned successfully",
                showSuccess: true
            }
        );

        setIsImporting(false);

        if (ok !== null) {
            onOpenChange(false);
            reset();
            if (onImported) await onImported();
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

    const formatFileSize = (bytes: number): string => {
        if (bytes < 1024) return bytes + " B";
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
        return (bytes / 1024 / 1024).toFixed(1) + " MB";
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
                        Select a file to import topics and assign them to this site.
                        Supported formats: {ALLOWED_IMPORT_EXTENSIONS.join(", ")}.
                        Maximum file size: {MAX_FILE_SIZE / 1024 / 1024}MB.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 pt-2">
                    {/* File picker with drag & drop */}
                    <div className="space-y-2">
                        <Label>File</Label>
                        <div
                            onDragOver={(e) => {
                                e.preventDefault();
                                e.stopPropagation();
                            }}
                            onDrop={async (e) => {
                                e.preventDefault();
                                e.stopPropagation();
                                const file = e.dataTransfer.files?.[0];
                                if (file) {
                                    await handleFileSelect(file);
                                }
                            }}
                            className="border-2 border-dashed rounded-lg p-4 text-sm transition-colors hover:border-primary/50 hover:bg-accent/20 cursor-pointer"
                            onClick={() => {
                                const input = document.getElementById("import-file-input") as HTMLInputElement | null;
                                input?.click();
                            }}
                        >
                            {filePath ? (
                                <div className="flex items-start gap-3">
                                    <FileCheck className="h-5 w-5 text-green-600 flex-shrink-0 mt-0.5" />
                                    <div className="flex-1 min-w-0">
                                        <div className="text-foreground font-medium truncate">{fileName}</div>
                                        <div className="text-xs text-muted-foreground mt-1">
                                            {formatFileSize(fileSize)}
                                            {filePath !== fileName && (
                                                <span className="block truncate mt-0.5" title={filePath}>
                          Path: {filePath}
                        </span>
                                            )}
                                        </div>
                                    </div>
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            reset();
                                        }}
                                        className="flex-shrink-0"
                                    >
                                        <X className="h-4 w-4" />
                                    </Button>
                                </div>
                            ) : (
                                <div className="flex items-center justify-between">
                                    <div>
                                        <div className="text-foreground">Drag & drop a file here</div>
                                        <div className="text-xs text-muted-foreground mt-1">
                                            or click to browse • {ALLOWED_IMPORT_EXTENSIONS.join(", ")}
                                        </div>
                                    </div>
                                    <Button
                                        size="sm"
                                        variant="outline"
                                        onClick={(e) => {
                                            e.stopPropagation();
                                            const input = document.getElementById("import-file-input") as HTMLInputElement | null;
                                            input?.click();
                                        }}
                                    >
                                        <FolderOpen className="h-4 w-4 mr-2" />
                                        Browse
                                    </Button>
                                </div>
                            )}
                        </div>
                        <input
                            id="import-file-input"
                            type="file"
                            className="hidden"
                            accept={ALLOWED_IMPORT_EXTENSIONS.map((ext) => `.${ext}`).join(",")}
                            onChange={async (e) => {
                                const file = e.target.files?.[0];
                                if (file) {
                                    await handleFileSelect(file);
                                }
                                // Reset input to allow selecting the same file again
                                e.target.value = "";
                            }}
                        />
                    </div>

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
                        {isImporting ? "Importing..." : "Import Topics"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

export default ImportTopicsDialog;