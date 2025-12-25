"use client";

import { useState } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { AlertCircle, CheckCircle2, Loader2, ScanLine, FileText, Heading1 } from "lucide-react";
import { sitemapService } from "@/services/sitemaps";
import { ScanSiteResult, TitleSource, ContentFilter } from "@/models/sitemaps";

// Props for creating a new sitemap via scan
interface ScanDialogCreateProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    mode: "create";
    siteId: number;
    sitemapName: string;
    onSuccess: (result: ScanSiteResult) => void;
}

// Props for scanning into existing sitemap
interface ScanDialogAddProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    mode: "add";
    sitemapId: number;
    parentNodeId?: number;
    onSuccess: (result: ScanSiteResult) => void;
}

type ScanDialogProps = ScanDialogCreateProps | ScanDialogAddProps;

type ScanState = "idle" | "loading" | "success" | "error";

export function ScanDialog(props: ScanDialogProps) {
    const { open, onOpenChange, onSuccess } = props;
    const [scanState, setScanState] = useState<ScanState>("idle");
    const [result, setResult] = useState<ScanSiteResult | null>(null);
    const [error, setError] = useState<string | null>(null);

    // Scan options
    const [titleSource, setTitleSource] = useState<TitleSource>("title");
    const [includeDrafts, setIncludeDrafts] = useState(true);
    // Always use pages only - sitemaps are for page structure, not posts
    const contentFilter: ContentFilter = "pages";

    const handleScan = async () => {
        setScanState("loading");
        setError(null);

        try {
            let scanResult: ScanSiteResult;

            if (props.mode === "create") {
                scanResult = await sitemapService.scanSite({
                    siteId: props.siteId,
                    sitemapName: props.sitemapName,
                    titleSource,
                    contentFilter,
                    includeDrafts,
                    maxDepth: 0, // Unlimited
                });
            } else {
                scanResult = await sitemapService.scanIntoSitemap({
                    sitemapId: props.sitemapId,
                    parentNodeId: props.parentNodeId,
                    titleSource,
                    contentFilter,
                    includeDrafts,
                    maxDepth: 0, // Unlimited
                });
            }

            setResult(scanResult);
            setScanState("success");
            onSuccess(scanResult);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Scan failed");
            setScanState("error");
        }
    };

    const handleClose = () => {
        // Reset state when closing
        setScanState("idle");
        setResult(null);
        setError(null);
        onOpenChange(false);
    };

    return (
        <Dialog open={open} onOpenChange={handleClose}>
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <ScanLine className="h-5 w-5" />
                        {props.mode === "create" ? "Scan Site Structure" : "Import from WordPress"}
                    </DialogTitle>
                    <DialogDescription>
                        {props.mode === "create"
                            ? "Fetch pages from WordPress to build the sitemap structure."
                            : "Scan the WordPress site and add discovered pages to your sitemap."}
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {scanState === "idle" && (
                        <>
                            {/* Title Source */}
                            <div className="space-y-2">
                                <Label>Title Source</Label>
                                <Select
                                    value={titleSource}
                                    onValueChange={(v) => setTitleSource(v as TitleSource)}
                                >
                                    <SelectTrigger>
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="title">
                                            <div className="flex items-center gap-2">
                                                <FileText className="h-4 w-4" />
                                                <span>Page Title</span>
                                            </div>
                                        </SelectItem>
                                        <SelectItem value="h1">
                                            <div className="flex items-center gap-2">
                                                <Heading1 className="h-4 w-4" />
                                                <span>First H1 Tag</span>
                                            </div>
                                        </SelectItem>
                                    </SelectContent>
                                </Select>
                                <p className="text-xs text-muted-foreground">
                                    {titleSource === "title"
                                        ? "Use the WordPress page/post title"
                                        : "Extract the first H1 heading from content"}
                                </p>
                            </div>

                            {/* Include Drafts */}
                            <div className="flex items-center justify-between">
                                <div className="space-y-0.5">
                                    <Label>Include Drafts</Label>
                                    <p className="text-xs text-muted-foreground">
                                        Also scan unpublished draft content
                                    </p>
                                </div>
                                <Switch
                                    checked={includeDrafts}
                                    onCheckedChange={setIncludeDrafts}
                                />
                            </div>
                        </>
                    )}

                    {scanState === "loading" && (
                        <div className="flex flex-col items-center gap-4 py-8">
                            <Loader2 className="h-8 w-8 animate-spin text-primary" />
                            <div className="text-center">
                                <p className="text-sm font-medium">Scanning site structure...</p>
                                <p className="text-xs text-muted-foreground mt-1">
                                    Fetching pages from WordPress
                                </p>
                            </div>
                        </div>
                    )}

                    {scanState === "success" && result && (
                        <div className="space-y-4">
                            <div className="flex flex-col items-center gap-2 py-4">
                                <CheckCircle2 className="h-10 w-10 text-green-500" />
                                <p className="text-sm font-medium">Scan completed</p>
                            </div>

                            <div className="grid grid-cols-3 gap-4 text-center">
                                <div className="bg-muted/50 rounded-lg p-3">
                                    <p className="text-2xl font-bold">{result.pagesScanned}</p>
                                    <p className="text-xs text-muted-foreground">Pages found</p>
                                </div>
                                <div className="bg-muted/50 rounded-lg p-3">
                                    <p className="text-2xl font-bold text-green-600">{result.nodesCreated}</p>
                                    <p className="text-xs text-muted-foreground">Nodes created</p>
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
                                                {err.title && `${err.title}: `}
                                                {err.message}
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            <p className="text-xs text-muted-foreground text-center">
                                Completed in {result.totalDuration}
                            </p>
                        </div>
                    )}

                    {scanState === "error" && (
                        <div className="flex flex-col items-center gap-4 py-8">
                            <AlertCircle className="h-10 w-10 text-destructive" />
                            <p className="text-sm text-destructive text-center">{error}</p>
                            <Button variant="outline" onClick={() => setScanState("idle")}>
                                Try again
                            </Button>
                        </div>
                    )}
                </div>

                <DialogFooter>
                    {scanState === "idle" && (
                        <>
                            <Button variant="outline" onClick={handleClose}>
                                Cancel
                            </Button>
                            <Button onClick={handleScan}>
                                <ScanLine className="mr-2 h-4 w-4" />
                                Start Scan
                            </Button>
                        </>
                    )}
                    {scanState === "success" && (
                        <Button onClick={handleClose}>
                            {result?.errors.length ? "Close" : "Done"}
                        </Button>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
