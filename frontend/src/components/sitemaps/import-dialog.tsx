"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Upload, FileSpreadsheet, FileJson, FileText, AlertCircle, CheckCircle2, Loader2, ChevronDown, BookOpen } from "lucide-react";
import { cn } from "@/lib/utils";
import { sitemapService } from "@/services/sitemaps";
import { ImportNodesResult } from "@/models/sitemaps";

interface ImportDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    sitemapId: number;
    parentNodeId?: number;
    onSuccess: () => void;
}

type ImportState = "idle" | "loading" | "success" | "error";

const FILE_ICONS: Record<string, React.ElementType> = {
    ".csv": FileText,
    ".json": FileJson,
    ".xlsx": FileSpreadsheet,
    ".xls": FileSpreadsheet,
};

export function ImportDialog({
    open,
    onOpenChange,
    sitemapId,
    parentNodeId,
    onSuccess,
}: ImportDialogProps) {
    const [supportedFormats, setSupportedFormats] = useState<string[]>([]);
    const [selectedFile, setSelectedFile] = useState<File | null>(null);
    const [importState, setImportState] = useState<ImportState>("idle");
    const [result, setResult] = useState<ImportNodesResult | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [isDragOver, setIsDragOver] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (open) {
            sitemapService.getSupportedImportFormats().then(setSupportedFormats);
            // Reset state when opening
            setSelectedFile(null);
            setImportState("idle");
            setResult(null);
            setError(null);
        }
    }, [open]);

    const getAcceptedExtensions = () => {
        return supportedFormats.map((f) => (f.startsWith(".") ? f : `.${f}`)).join(",");
    };

    const isValidFile = (file: File) => {
        const ext = "." + file.name.split(".").pop()?.toLowerCase();
        return supportedFormats.some((f) => {
            const format = f.startsWith(".") ? f : `.${f}`;
            return format.toLowerCase() === ext;
        });
    };

    const handleFileSelect = (file: File) => {
        if (isValidFile(file)) {
            setSelectedFile(file);
            setError(null);
        } else {
            setError(`Invalid file type. Supported formats: ${supportedFormats.join(", ")}`);
        }
    };

    const handleDrop = useCallback((e: React.DragEvent) => {
        e.preventDefault();
        setIsDragOver(false);

        const file = e.dataTransfer.files[0];
        if (file) {
            handleFileSelect(file);
        }
    }, [supportedFormats]);

    const handleDragOver = useCallback((e: React.DragEvent) => {
        e.preventDefault();
        setIsDragOver(true);
    }, []);

    const handleDragLeave = useCallback((e: React.DragEvent) => {
        e.preventDefault();
        setIsDragOver(false);
    }, []);

    const handleImport = async () => {
        if (!selectedFile) return;

        setImportState("loading");
        setError(null);

        try {
            // Read file as base64
            const fileData = await readFileAsBase64(selectedFile);

            const importResult = await sitemapService.importNodes({
                sitemapId,
                parentNodeId,
                filename: selectedFile.name,
                fileDataBase64: fileData,
            });

            setResult(importResult);
            setImportState("success");

            // Mark as successful - parent will handle navigation on close
            onSuccess();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Import failed");
            setImportState("error");
        }
    };

    const readFileAsBase64 = (file: File): Promise<string> => {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => {
                const result = reader.result as string;
                // Remove data URL prefix (e.g., "data:application/json;base64,")
                const base64 = result.split(",")[1];
                resolve(base64);
            };
            reader.onerror = () => reject(new Error("Failed to read file"));
            reader.readAsDataURL(file);
        });
    };

    const getFileIcon = (filename: string) => {
        const ext = "." + filename.split(".").pop()?.toLowerCase();
        const Icon = FILE_ICONS[ext] || FileText;
        return <Icon className="h-8 w-8 text-muted-foreground" />;
    };

    const handleClose = () => {
        onOpenChange(false);
    };

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-lg">
                <DialogHeader>
                    <DialogTitle>Import Nodes</DialogTitle>
                </DialogHeader>

                <div className="space-y-4">
                    {importState === "idle" && (
                        <>
                            {/* Drop zone */}
                            <div
                                className={cn(
                                    "border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors",
                                    isDragOver
                                        ? "border-primary bg-primary/5"
                                        : "border-muted-foreground/25 hover:border-muted-foreground/50",
                                    selectedFile && "border-primary bg-primary/5"
                                )}
                                onDrop={handleDrop}
                                onDragOver={handleDragOver}
                                onDragLeave={handleDragLeave}
                                onClick={() => fileInputRef.current?.click()}
                            >
                                <input
                                    ref={fileInputRef}
                                    type="file"
                                    accept={getAcceptedExtensions()}
                                    onChange={(e) => {
                                        const file = e.target.files?.[0];
                                        if (file) handleFileSelect(file);
                                    }}
                                    className="hidden"
                                />

                                {selectedFile ? (
                                    <div className="flex flex-col items-center gap-2">
                                        {getFileIcon(selectedFile.name)}
                                        <span className="text-sm font-medium">{selectedFile.name}</span>
                                        <span className="text-xs text-muted-foreground">
                                            {(selectedFile.size / 1024).toFixed(1)} KB
                                        </span>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                setSelectedFile(null);
                                            }}
                                        >
                                            Choose different file
                                        </Button>
                                    </div>
                                ) : (
                                    <div className="flex flex-col items-center gap-2">
                                        <Upload className="h-8 w-8 text-muted-foreground" />
                                        <p className="text-sm text-muted-foreground">
                                            Drop a file here or click to browse
                                        </p>
                                        <p className="text-xs text-muted-foreground">
                                            Supported: {supportedFormats.join(", ")}
                                        </p>
                                    </div>
                                )}
                            </div>

                            {/* Format Documentation */}
                            <FormatDocumentation />

                            {error && (
                                <div className="flex items-center gap-2 text-sm text-destructive">
                                    <AlertCircle className="h-4 w-4" />
                                    {error}
                                </div>
                            )}
                        </>
                    )}

                    {importState === "loading" && (
                        <div className="flex flex-col items-center gap-4 py-8">
                            <Loader2 className="h-8 w-8 animate-spin text-primary" />
                            <p className="text-sm text-muted-foreground">Importing nodes...</p>
                        </div>
                    )}

                    {importState === "success" && result && (
                        <div className="space-y-4">
                            <div className="flex flex-col items-center gap-2 py-4">
                                <CheckCircle2 className="h-10 w-10 text-green-500" />
                                <p className="text-sm font-medium">Import completed</p>
                            </div>

                            <div className="grid grid-cols-3 gap-4 text-center">
                                <div className="bg-muted/50 rounded-lg p-3">
                                    <p className="text-2xl font-bold">{result.totalRows}</p>
                                    <p className="text-xs text-muted-foreground">Total rows</p>
                                </div>
                                <div className="bg-muted/50 rounded-lg p-3">
                                    <p className="text-2xl font-bold text-green-600">{result.nodesCreated}</p>
                                    <p className="text-xs text-muted-foreground">Created</p>
                                </div>
                                <div className="bg-muted/50 rounded-lg p-3">
                                    <p className="text-2xl font-bold text-yellow-600">{result.nodesSkipped}</p>
                                    <p className="text-xs text-muted-foreground">Skipped</p>
                                </div>
                            </div>

                            {result.errors.length > 0 && (
                                <div className="space-y-2">
                                    <p className="text-sm font-medium text-destructive">
                                        {result.errors.length} error(s):
                                    </p>
                                    <div className="max-h-32 overflow-y-auto space-y-1">
                                        {result.errors.map((err, i) => (
                                            <div
                                                key={i}
                                                className="text-xs text-destructive bg-destructive/10 rounded px-2 py-1"
                                            >
                                                {err.row && `Row ${err.row}: `}
                                                {err.message}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            <p className="text-xs text-muted-foreground text-center">
                                Processed in {result.processingTime}
                            </p>
                        </div>
                    )}

                    {importState === "error" && (
                        <div className="flex flex-col items-center gap-4 py-8">
                            <AlertCircle className="h-10 w-10 text-destructive" />
                            <p className="text-sm text-destructive">{error}</p>
                            <Button variant="outline" onClick={() => setImportState("idle")}>
                                Try again
                            </Button>
                        </div>
                    )}
                </div>

                <DialogFooter>
                    {importState === "idle" && (
                        <>
                            <Button variant="outline" onClick={() => onOpenChange(false)}>
                                Cancel
                            </Button>
                            <Button onClick={handleImport} disabled={!selectedFile}>
                                Import
                            </Button>
                        </>
                    )}
                    {importState === "success" && (
                        <Button onClick={handleClose}>
                            {result?.errors.length ? "Close" : "Done"}
                        </Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

// Format Documentation Component
function FormatDocumentation() {
    const [isOpen, setIsOpen] = useState(false);

    return (
        <Collapsible open={isOpen} onOpenChange={setIsOpen}>
            <CollapsibleTrigger asChild>
                <Button
                    variant="ghost"
                    size="sm"
                    className="w-full justify-between text-muted-foreground hover:text-foreground"
                >
                    <span className="flex items-center gap-2">
                        <BookOpen className="h-4 w-4" />
                        File format documentation
                    </span>
                    <ChevronDown
                        className={cn(
                            "h-4 w-4 transition-transform duration-200",
                            isOpen && "rotate-180"
                        )}
                    />
                </Button>
            </CollapsibleTrigger>
            <CollapsibleContent className="pt-2">
                <div className="rounded-lg border bg-muted/30 p-3">
                    <Tabs defaultValue="csv" className="w-full">
                        <TabsList className="grid w-full grid-cols-3 h-8">
                            <TabsTrigger value="csv" className="text-xs gap-1">
                                <FileText className="h-3 w-3" />
                                CSV
                            </TabsTrigger>
                            <TabsTrigger value="json" className="text-xs gap-1">
                                <FileJson className="h-3 w-3" />
                                JSON
                            </TabsTrigger>
                            <TabsTrigger value="xlsx" className="text-xs gap-1">
                                <FileSpreadsheet className="h-3 w-3" />
                                Excel
                            </TabsTrigger>
                        </TabsList>

                        <TabsContent value="csv" className="mt-3 space-y-3">
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Required columns:</p>
                                <ul className="text-xs text-muted-foreground space-y-0.5 ml-3">
                                    <li><code className="bg-muted px-1 rounded">path</code> or <code className="bg-muted px-1 rounded">url</code> - URL path</li>
                                </ul>
                            </div>
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Optional columns:</p>
                                <ul className="text-xs text-muted-foreground space-y-0.5 ml-3">
                                    <li><code className="bg-muted px-1 rounded">title</code> or <code className="bg-muted px-1 rounded">name</code> - Page title</li>
                                    <li><code className="bg-muted px-1 rounded">keywords</code> or <code className="bg-muted px-1 rounded">tags</code> - Comma-separated</li>
                                </ul>
                            </div>
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Example:</p>
                                <pre className="text-[10px] bg-muted p-2 rounded overflow-x-auto">
{`path,title,keywords
/services,Services,service
/services/web,Web Dev,"web,dev"
/about,About Us,`}</pre>
                            </div>
                        </TabsContent>

                        <TabsContent value="json" className="mt-3 space-y-3">
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Structure:</p>
                                <p className="text-xs text-muted-foreground">
                                    Array of objects, or object with <code className="bg-muted px-1 rounded">nodes</code>/<code className="bg-muted px-1 rounded">pages</code>/<code className="bg-muted px-1 rounded">items</code> array
                                </p>
                            </div>
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Object fields:</p>
                                <ul className="text-xs text-muted-foreground space-y-0.5 ml-3">
                                    <li><code className="bg-muted px-1 rounded">path</code> / <code className="bg-muted px-1 rounded">url</code> / <code className="bg-muted px-1 rounded">slug</code> - URL path (required)</li>
                                    <li><code className="bg-muted px-1 rounded">title</code> / <code className="bg-muted px-1 rounded">name</code> - Page title</li>
                                    <li><code className="bg-muted px-1 rounded">keywords</code> / <code className="bg-muted px-1 rounded">tags</code> - Array or comma-string</li>
                                </ul>
                            </div>
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Example:</p>
                                <pre className="text-[10px] bg-muted p-2 rounded overflow-x-auto">
{`[
  {
    "path": "/services",
    "title": "Services",
    "keywords": ["service"]
  },
  {
    "path": "/services/web",
    "title": "Web Development"
  }
]`}</pre>
                            </div>
                        </TabsContent>

                        <TabsContent value="xlsx" className="mt-3 space-y-3">
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Sheet:</p>
                                <p className="text-xs text-muted-foreground">
                                    First sheet is used. First row must contain column headers.
                                </p>
                            </div>
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Columns (same as CSV):</p>
                                <ul className="text-xs text-muted-foreground space-y-0.5 ml-3">
                                    <li><code className="bg-muted px-1 rounded">path</code> / <code className="bg-muted px-1 rounded">url</code> - URL path (required)</li>
                                    <li><code className="bg-muted px-1 rounded">title</code> / <code className="bg-muted px-1 rounded">name</code> - Page title</li>
                                    <li><code className="bg-muted px-1 rounded">keywords</code> / <code className="bg-muted px-1 rounded">tags</code> - Comma-separated</li>
                                </ul>
                            </div>
                            <div className="space-y-1.5">
                                <p className="text-xs font-medium">Example layout:</p>
                                <div className="text-[10px] bg-muted p-2 rounded overflow-x-auto">
                                    <table className="w-full border-collapse">
                                        <thead>
                                            <tr className="border-b border-muted-foreground/20">
                                                <th className="text-left px-2 py-1 font-medium">path</th>
                                                <th className="text-left px-2 py-1 font-medium">title</th>
                                                <th className="text-left px-2 py-1 font-medium">keywords</th>
                                            </tr>
                                        </thead>
                                        <tbody className="text-muted-foreground">
                                            <tr>
                                                <td className="px-2 py-0.5">/services</td>
                                                <td className="px-2 py-0.5">Services</td>
                                                <td className="px-2 py-0.5">service</td>
                                            </tr>
                                            <tr>
                                                <td className="px-2 py-0.5">/services/web</td>
                                                <td className="px-2 py-0.5">Web Dev</td>
                                                <td className="px-2 py-0.5">web, dev</td>
                                            </tr>
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        </TabsContent>
                    </Tabs>

                    <div className="mt-3 pt-3 border-t space-y-1.5">
                        <p className="text-xs font-medium">Path hierarchy:</p>
                        <p className="text-xs text-muted-foreground">
                            Paths define the tree structure automatically. For example, <code className="bg-muted px-1 rounded">/services/web</code> becomes
                            a child of <code className="bg-muted px-1 rounded">/services</code>. Parent nodes are created automatically if missing.
                        </p>
                    </div>
                </div>
            </CollapsibleContent>
        </Collapsible>
    );
}
