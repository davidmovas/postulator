"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter } from "next/navigation";
import { useQueryId } from "@/hooks/use-query-param";
import { useApiCall } from "@/hooks/use-api-call";
import { sitemapService } from "@/services/sitemaps";
import { siteService } from "@/services/sites";
import { Sitemap } from "@/models/sitemaps";
import { Site } from "@/models/sites";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
    Plus,
    Map,
    FileInput,
    Sparkles,
    ScanLine,
    MoreHorizontal,
    Pencil,
    Copy,
    Trash2,
    ArrowLeft,
    Type,
} from "lucide-react";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { formatSmartDate } from "@/lib/time";
import { ImportDialog } from "@/components/sitemaps/import-dialog";
import { ScanDialog } from "@/components/sitemaps/scan-dialog";
import { GenerateDialog } from "@/components/sitemaps/generate-dialog";
import { ScanSiteResult, GenerateSitemapStructureResult } from "@/models/sitemaps";

function SitemapsPageContent() {
    const router = useRouter();
    const siteId = useQueryId();
    const { execute, isLoading } = useApiCall();

    const [site, setSite] = useState<Site | null>(null);
    const [sitemaps, setSitemaps] = useState<Sitemap[]>([]);
    const [createDialogOpen, setCreateDialogOpen] = useState(false);
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
    const [duplicateDialogOpen, setDuplicateDialogOpen] = useState(false);
    const [renameDialogOpen, setRenameDialogOpen] = useState(false);
    const [importDialogOpen, setImportDialogOpen] = useState(false);
    const [scanDialogOpen, setScanDialogOpen] = useState(false);
    const [generateDialogOpen, setGenerateDialogOpen] = useState(false);
    const [selectedSitemap, setSelectedSitemap] = useState<Sitemap | null>(null);
    const [createdSitemapId, setCreatedSitemapId] = useState<number | null>(null);

    // Form states
    const [newName, setNewName] = useState("");
    const [newDescription, setNewDescription] = useState("");
    const [newSource, setNewSource] = useState<"manual" | "imported" | "generated" | "scanned">("manual");
    const [duplicateName, setDuplicateName] = useState("");
    const [renameName, setRenameName] = useState("");

    const loadData = async () => {
        const [siteResult, sitemapsResult] = await Promise.all([
            execute<Site>(() => siteService.getSite(siteId), {
                errorTitle: "Failed to load site",
            }),
            execute<Sitemap[]>(() => sitemapService.listSitemaps(siteId), {
                errorTitle: "Failed to load sitemaps",
            }),
        ]);

        if (siteResult) setSite(siteResult);
        if (sitemapsResult) setSitemaps(sitemapsResult);
    };

    useEffect(() => {
        if (siteId) {
            loadData();
        }
    }, [siteId]);

    const handleCreate = async () => {
        if (!site) return;

        // For scanned mode, we don't create sitemap first - the scanner does it
        if (newSource === "scanned") {
            setCreateDialogOpen(false);
            setScanDialogOpen(true);
            return;
        }

        // For generated mode, we don't create sitemap first - the generator does it
        if (newSource === "generated") {
            setCreateDialogOpen(false);
            setGenerateDialogOpen(true);
            return;
        }

        const result = await execute<Sitemap>(
            () =>
                sitemapService.createSitemap({
                    siteId,
                    name: newName,
                    description: newDescription || undefined,
                    source: newSource,
                    siteUrl: site.url,
                }),
            {
                successMessage: newSource === "imported" ? "Sitemap created" : "Sitemap created successfully",
                showSuccessToast: newSource !== "imported",
                errorTitle: "Failed to create sitemap",
            }
        );

        if (result) {
            setCreateDialogOpen(false);
            setNewName("");
            setNewDescription("");

            if (newSource === "imported") {
                // Open import dialog for the newly created sitemap
                setCreatedSitemapId(result.id);
                setImportDialogOpen(true);
            } else {
                setNewSource("manual");
                router.push(`/sites/sitemaps/editor?id=${siteId}&sitemapId=${result.id}`);
            }
        }
    };

    const [importSuccessful, setImportSuccessful] = useState(false);

    const handleImportSuccess = () => {
        // Just mark import as successful - navigation happens on dialog close
        setImportSuccessful(true);
    };

    const handleImportClose = async (open: boolean) => {
        if (!open) {
            if (importSuccessful && createdSitemapId) {
                // Import was successful - navigate to editor
                router.push(`/sites/sitemaps/editor?id=${siteId}&sitemapId=${createdSitemapId}`);
            } else if (createdSitemapId) {
                // User closed dialog without importing - delete the empty sitemap
                await execute(
                    () => sitemapService.deleteSitemap(createdSitemapId),
                    { errorTitle: "Failed to cancel sitemap creation" }
                );
                loadData();
            }
            setCreatedSitemapId(null);
            setNewSource("manual");
            setImportSuccessful(false);
        }
        setImportDialogOpen(open);
    };

    const handleScanSuccess = (result: ScanSiteResult) => {
        // Scan completed - navigate to editor
        setNewName("");
        setNewDescription("");
        setNewSource("manual");
        router.push(`/sites/sitemaps/editor?id=${siteId}&sitemapId=${result.sitemapId}`);
    };

    const handleScanClose = (open: boolean) => {
        if (!open) {
            // Reset form state when closing
            setNewName("");
            setNewDescription("");
            setNewSource("manual");
            loadData();
        }
        setScanDialogOpen(open);
    };

    const handleGenerateSuccess = (result: GenerateSitemapStructureResult) => {
        // Generation completed - navigate to editor
        setNewName("");
        setNewDescription("");
        setNewSource("manual");
        router.push(`/sites/sitemaps/editor?id=${siteId}&sitemapId=${result.sitemapId}`);
    };

    const handleGenerateClose = (open: boolean) => {
        if (!open) {
            // Reset form state when closing
            setNewName("");
            setNewDescription("");
            setNewSource("manual");
            loadData();
        }
        setGenerateDialogOpen(open);
    };

    const handleDuplicate = async () => {
        if (!selectedSitemap) return;

        const result = await execute<Sitemap>(
            () => sitemapService.duplicateSitemap(selectedSitemap.id, duplicateName),
            {
                successMessage: "Sitemap duplicated successfully",
                showSuccessToast: true,
                errorTitle: "Failed to duplicate sitemap",
            }
        );

        if (result) {
            setDuplicateDialogOpen(false);
            setDuplicateName("");
            setSelectedSitemap(null);
            loadData();
        }
    };

    const handleDelete = async () => {
        if (!selectedSitemap) return;

        await execute(
            () => sitemapService.deleteSitemap(selectedSitemap.id),
            {
                successMessage: "Sitemap deleted successfully",
                showSuccessToast: true,
                errorTitle: "Failed to delete sitemap",
            }
        );

        setDeleteDialogOpen(false);
        setSelectedSitemap(null);
        loadData();
    };

    const handleRename = async () => {
        if (!selectedSitemap) return;

        await execute(
            () =>
                sitemapService.updateSitemap({
                    ...selectedSitemap,
                    name: renameName,
                }),
            {
                successMessage: "Sitemap renamed successfully",
                showSuccessToast: true,
                errorTitle: "Failed to rename sitemap",
            }
        );

        setRenameDialogOpen(false);
        setRenameName("");
        setSelectedSitemap(null);
        loadData();
    };

    const openEditor = (sitemapId: number) => {
        router.push(`/sites/sitemaps/editor?id=${siteId}&sitemapId=${sitemapId}`);
    };

    if (isLoading && !site) {
        return (
            <div className="p-6 space-y-6">
                <div className="h-12 bg-muted/30 rounded-lg animate-pulse" />
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                    {[1, 2, 3].map((i) => (
                        <div key={i} className="h-40 bg-muted/30 rounded-lg animate-pulse" />
                    ))}
                </div>
            </div>
        );
    }

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Button variant="ghost" size="icon" onClick={() => router.push(`/sites/view?id=${siteId}`)}>
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                    <div>
                        <h1 className="text-3xl font-bold tracking-tight">Site Structure</h1>
                        <p className="text-muted-foreground">
                            {site?.name} - Manage site structure and page hierarchy
                        </p>
                    </div>
                </div>
                <Button onClick={() => setCreateDialogOpen(true)}>
                    <Plus className="mr-2 h-4 w-4" />
                    New Sitemap
                </Button>
            </div>

            {/* Sitemaps Grid */}
            {sitemaps.length === 0 ? (
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12">
                        <Map className="h-12 w-12 text-muted-foreground mb-4" />
                        <h3 className="text-lg font-semibold">No sitemaps yet</h3>
                        <p className="text-muted-foreground text-center max-w-sm mt-2">
                            Create your first sitemap to start building your site structure.
                        </p>
                        <Button className="mt-4" onClick={() => setCreateDialogOpen(true)}>
                            <Plus className="mr-2 h-4 w-4" />
                            Create Sitemap
                        </Button>
                    </CardContent>
                </Card>
            ) : (
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                    {sitemaps.map((sitemap) => (
                        <Card
                            key={sitemap.id}
                            className="cursor-pointer hover:border-primary/50 transition-colors"
                            onClick={() => openEditor(sitemap.id)}
                        >
                            <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
                                <div className="space-y-1">
                                    <CardTitle className="text-lg">{sitemap.name}</CardTitle>
                                    <CardDescription className="line-clamp-2">
                                        {sitemap.description || "No description"}
                                    </CardDescription>
                                </div>
                                <DropdownMenu>
                                    <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
                                        <Button variant="ghost" size="icon" className="h-8 w-8">
                                            <MoreHorizontal className="h-4 w-4" />
                                        </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end">
                                        <DropdownMenuItem
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                openEditor(sitemap.id);
                                            }}
                                        >
                                            <Pencil className="mr-2 h-4 w-4" />
                                            Edit
                                        </DropdownMenuItem>
                                        <DropdownMenuItem
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                setSelectedSitemap(sitemap);
                                                setRenameName(sitemap.name);
                                                setRenameDialogOpen(true);
                                            }}
                                        >
                                            <Type className="mr-2 h-4 w-4" />
                                            Rename
                                        </DropdownMenuItem>
                                        <DropdownMenuItem
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                setSelectedSitemap(sitemap);
                                                setDuplicateName(`${sitemap.name} (Copy)`);
                                                setDuplicateDialogOpen(true);
                                            }}
                                        >
                                            <Copy className="mr-2 h-4 w-4" />
                                            Duplicate
                                        </DropdownMenuItem>
                                        <DropdownMenuSeparator />
                                        <DropdownMenuItem
                                            className="text-destructive"
                                            onClick={(e) => {
                                                e.stopPropagation();
                                                setSelectedSitemap(sitemap);
                                                setDeleteDialogOpen(true);
                                            }}
                                        >
                                            <Trash2 className="mr-2 h-4 w-4" />
                                            Delete
                                        </DropdownMenuItem>
                                    </DropdownMenuContent>
                                </DropdownMenu>
                            </CardHeader>
                            <CardContent>
                                <p className="text-xs text-muted-foreground">
                                    Updated {formatSmartDate(sitemap.updatedAt)}
                                </p>
                            </CardContent>
                        </Card>
                    ))}
                </div>
            )}

            {/* Create Dialog */}
            <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Create New Sitemap</DialogTitle>
                        <DialogDescription>
                            Create a new sitemap to define your site structure.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="name">Name</Label>
                            <Input
                                id="name"
                                value={newName}
                                onChange={(e) => setNewName(e.target.value)}
                                placeholder="My Sitemap"
                            />
                        </div>
                        <div className="space-y-2">
                            <Label htmlFor="description">Description (optional)</Label>
                            <Textarea
                                id="description"
                                value={newDescription}
                                onChange={(e) => setNewDescription(e.target.value)}
                                placeholder="Describe the purpose of this sitemap..."
                                rows={3}
                            />
                        </div>
                        <div className="space-y-2">
                            <Label>Creation Method</Label>
                            <div className="grid grid-cols-2 gap-3">
                                {[
                                    {
                                        value: "manual",
                                        label: "Manual",
                                        icon: Map,
                                        description: "Build from scratch",
                                        color: "blue",
                                        activeClasses: "bg-blue-500/10 border-blue-500 text-blue-600 dark:text-blue-400",
                                        iconClasses: "text-blue-500",
                                    },
                                    {
                                        value: "imported",
                                        label: "Import",
                                        icon: FileInput,
                                        description: "From CSV/Excel file",
                                        color: "amber",
                                        activeClasses: "bg-amber-500/10 border-amber-500 text-amber-600 dark:text-amber-400",
                                        iconClasses: "text-amber-500",
                                    },
                                    {
                                        value: "generated",
                                        label: "AI Generate",
                                        icon: Sparkles,
                                        description: "AI-powered",
                                        color: "purple",
                                        activeClasses: "bg-purple-500/10 border-purple-500 text-purple-600 dark:text-purple-400",
                                        iconClasses: "text-purple-500",
                                    },
                                    {
                                        value: "scanned",
                                        label: "Scan Site",
                                        icon: ScanLine,
                                        description: "From WordPress",
                                        color: "green",
                                        activeClasses: "bg-green-500/10 border-green-500 text-green-600 dark:text-green-400",
                                        iconClasses: "text-green-500",
                                    },
                                ].map((option) => {
                                    const isSelected = newSource === option.value;
                                    return (
                                        <button
                                            key={option.value}
                                            type="button"
                                            className={`
                                                relative flex flex-col items-start gap-1 rounded-lg border-2 p-3 text-left transition-all
                                                ${isSelected
                                                    ? option.activeClasses
                                                    : "border-muted bg-background hover:border-muted-foreground/30 hover:bg-muted/50"
                                                }
                                                ${option.disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}
                                            `}
                                            onClick={() => !option.disabled && setNewSource(option.value as "manual" | "imported" | "generated" | "scanned")}
                                            disabled={option.disabled}
                                        >
                                            <div className="flex items-center gap-2">
                                                <option.icon className={`h-4 w-4 ${isSelected ? "" : option.iconClasses}`} />
                                                <span className="font-medium text-sm">{option.label}</span>
                                            </div>
                                            <span className={`text-xs ${isSelected ? "opacity-80" : "text-muted-foreground"}`}>
                                                {option.disabled ? "Coming soon" : option.description}
                                            </span>
                                            {isSelected && (
                                                <div className="absolute top-2 right-2 h-2 w-2 rounded-full bg-current" />
                                            )}
                                        </button>
                                    );
                                })}
                            </div>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleCreate} disabled={!newName.trim() || isLoading}>
                            Create
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Duplicate Dialog */}
            <Dialog open={duplicateDialogOpen} onOpenChange={setDuplicateDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Duplicate Sitemap</DialogTitle>
                        <DialogDescription>
                            Create a copy of "{selectedSitemap?.name}" with a new name.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="duplicateName">New Name</Label>
                            <Input
                                id="duplicateName"
                                value={duplicateName}
                                onChange={(e) => setDuplicateName(e.target.value)}
                                placeholder="New sitemap name"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDuplicateDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleDuplicate} disabled={!duplicateName.trim() || isLoading}>
                            Duplicate
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Dialog */}
            <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Delete Sitemap</DialogTitle>
                        <DialogDescription>
                            Are you sure you want to delete "{selectedSitemap?.name}"? This action cannot be
                            undone and will delete all nodes in this sitemap.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="destructive" onClick={handleDelete} disabled={isLoading}>
                            Delete
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Rename Dialog */}
            <Dialog open={renameDialogOpen} onOpenChange={setRenameDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>Rename Sitemap</DialogTitle>
                        <DialogDescription>
                            Enter a new name for "{selectedSitemap?.name}".
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="renameName">Name</Label>
                            <Input
                                id="renameName"
                                value={renameName}
                                onChange={(e) => setRenameName(e.target.value)}
                                placeholder="Sitemap name"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setRenameDialogOpen(false)}>
                            Cancel
                        </Button>
                        <Button onClick={handleRename} disabled={!renameName.trim() || isLoading}>
                            Rename
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Import Dialog */}
            {createdSitemapId && (
                <ImportDialog
                    open={importDialogOpen}
                    onOpenChange={handleImportClose}
                    sitemapId={createdSitemapId}
                    onSuccess={handleImportSuccess}
                />
            )}

            {/* Scan Dialog */}
            <ScanDialog
                open={scanDialogOpen}
                onOpenChange={handleScanClose}
                mode="create"
                siteId={siteId}
                sitemapName={newName}
                onSuccess={handleScanSuccess}
            />

            {/* Generate Dialog */}
            <GenerateDialog
                open={generateDialogOpen}
                onOpenChange={handleGenerateClose}
                mode="create"
                siteId={siteId}
                sitemapName={newName}
                onSuccess={handleGenerateSuccess}
            />
        </div>
    );
}

export default function SitemapsPage() {
    return (
        <Suspense
            fallback={
                <div className="p-6 space-y-6">
                    <div className="h-12 bg-muted/30 rounded-lg animate-pulse" />
                    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                        {[1, 2, 3].map((i) => (
                            <div key={i} className="h-40 bg-muted/30 rounded-lg animate-pulse" />
                        ))}
                    </div>
                </div>
            }
        >
            <SitemapsPageContent />
        </Suspense>
    );
}
