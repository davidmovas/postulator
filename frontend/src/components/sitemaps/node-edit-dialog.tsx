"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { useApiCall } from "@/hooks/use-api-call";
import { sitemapService } from "@/services/sitemaps";
import {
    SitemapNode,
    CreateNodeInput,
    UpdateNodeInput,
} from "@/models/sitemaps";
import { X, Plus, Trash2, AlertTriangle, Home } from "lucide-react";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { cn } from "@/lib/utils";

interface NodeEditDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    node?: SitemapNode;
    sitemapId?: number;
    parentId?: number;
    onUpdate?: () => void;
    onCreate?: (input: CreateNodeInput) => void;
    onDelete?: () => void;
    onAddChild?: () => void;
    onRecordUpdate?: (nodeId: number, previousData: Partial<SitemapNode>, newData: Partial<SitemapNode>) => void;
}

export function NodeEditDialog({
    open,
    onOpenChange,
    node,
    sitemapId,
    parentId,
    onUpdate,
    onCreate,
    onDelete,
    onAddChild,
    onRecordUpdate,
}: NodeEditDialogProps) {
    const { execute, isLoading } = useApiCall();
    const isEditMode = !!node;
    const isRootNode = node?.isRoot ?? false;

    const [title, setTitle] = useState("");
    const [slug, setSlug] = useState("");
    const [description, setDescription] = useState("");
    const [keywords, setKeywords] = useState<string[]>([]);
    const [newKeyword, setNewKeyword] = useState("");
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

    // Track if form has changes for auto-save
    const initialValuesRef = useRef<{ title: string; slug: string; description: string; keywords: string[] } | null>(null);

    useEffect(() => {
        if (open) {
            if (node) {
                setTitle(node.title);
                setSlug(node.slug);
                setDescription(node.description || "");
                setKeywords(node.keywords || []);
                initialValuesRef.current = {
                    title: node.title,
                    slug: node.slug,
                    description: node.description || "",
                    keywords: node.keywords || [],
                };
            } else {
                setTitle("");
                setSlug("");
                setDescription("");
                setKeywords([]);
                initialValuesRef.current = null;
            }
            setNewKeyword("");
        }
    }, [open, node]);

    const generateSlug = (text: string) => {
        return text
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, "-")
            .replace(/^-|-$/g, "");
    };

    const handleTitleChange = (value: string) => {
        setTitle(value);
        if (!isEditMode || !slug) {
            setSlug(generateSlug(value));
        }
    };

    const handleSlugChange = (value: string) => {
        // Remove leading slash and normalize
        let normalizedSlug = value.toLowerCase().replace(/[^a-z0-9-]/g, "");
        setSlug(normalizedSlug);
    };

    const handleAddKeyword = () => {
        const keyword = newKeyword.trim();
        if (keyword && !keywords.includes(keyword)) {
            setKeywords([...keywords, keyword]);
            setNewKeyword("");
        }
    };

    const handleRemoveKeyword = (keyword: string) => {
        setKeywords(keywords.filter((k) => k !== keyword));
    };

    const handleKeywordKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === "Enter") {
            e.preventDefault();
            handleAddKeyword();
        }
    };

    // Check if form has changes
    const hasChanges = useCallback(() => {
        if (!initialValuesRef.current) return false;
        const initial = initialValuesRef.current;
        return (
            title !== initial.title ||
            slug !== initial.slug ||
            description !== initial.description ||
            JSON.stringify(keywords) !== JSON.stringify(initial.keywords)
        );
    }, [title, slug, description, keywords]);

    const handleSave = useCallback(async () => {
        if (isEditMode && node) {
            if (isRootNode) return;

            const input: UpdateNodeInput = {
                id: node.id,
                title,
                slug,
                description: description || undefined,
                keywords,
            };

            // Record the update for undo/redo BEFORE applying
            if (onRecordUpdate && initialValuesRef.current) {
                const previousData: Partial<SitemapNode> = {
                    title: initialValuesRef.current.title,
                    slug: initialValuesRef.current.slug,
                    description: initialValuesRef.current.description || undefined,
                    keywords: initialValuesRef.current.keywords,
                };
                const newData: Partial<SitemapNode> = {
                    title,
                    slug,
                    description: description || undefined,
                    keywords,
                };
                onRecordUpdate(node.id, previousData, newData);
            }

            await execute(() => sitemapService.updateNode(input), {
                errorTitle: "Failed to update node",
            });

            onUpdate?.();
        } else if (sitemapId !== undefined) {
            const input: CreateNodeInput = {
                sitemapId,
                parentId,
                title,
                slug,
                description: description || undefined,
                keywords,
            };

            onCreate?.(input);
        }
    }, [isEditMode, node, isRootNode, title, slug, description, keywords, execute, onUpdate, sitemapId, parentId, onCreate, onRecordUpdate]);

    // Auto-save on close for edit mode
    const handleOpenChange = useCallback(async (newOpen: boolean) => {
        if (!newOpen && isEditMode && !isRootNode && hasChanges() && title.trim() && slug.trim()) {
            await handleSave();
        }
        onOpenChange(newOpen);
    }, [isEditMode, isRootNode, hasChanges, title, slug, handleSave, onOpenChange]);

    const handleDelete = () => {
        if (isRootNode) return;
        setDeleteDialogOpen(true);
    };

    const confirmDelete = () => {
        setDeleteDialogOpen(false);
        onDelete?.();
    };

    return (
        <>
            <Dialog open={open} onOpenChange={handleOpenChange}>
                <DialogContent className="max-w-md gap-0 p-0 overflow-hidden">
                    <DialogHeader className="px-4 py-3 border-b bg-muted/30">
                        <DialogTitle className="flex items-center gap-2 text-base">
                            {isRootNode && <Home className="h-4 w-4 text-primary" />}
                            {isEditMode ? (isRootNode ? "Root Node" : "Edit Node") : "New Node"}
                        </DialogTitle>
                    </DialogHeader>

                    {isRootNode ? (
                        <div className="p-4">
                            <div className="flex items-start gap-3 p-3 bg-muted/50 rounded-lg">
                                <AlertTriangle className="h-5 w-5 text-muted-foreground mt-0.5 shrink-0" />
                                <div className="space-y-1">
                                    <p className="text-sm font-medium">This is the root node</p>
                                    <p className="text-xs text-muted-foreground">
                                        Represents your homepage ({node?.title}). Cannot be edited or deleted.
                                    </p>
                                </div>
                            </div>
                        </div>
                    ) : (
                        <div className="p-4 space-y-4">
                            {/* Title */}
                            <div className="space-y-1.5">
                                <Label htmlFor="title" className="text-xs font-medium text-muted-foreground">
                                    Title
                                </Label>
                                <Input
                                    id="title"
                                    value={title}
                                    onChange={(e) => handleTitleChange(e.target.value)}
                                    placeholder="Page title"
                                    className="h-9"
                                />
                            </div>

                            {/* Slug with leading slash */}
                            <div className="space-y-1.5">
                                <Label htmlFor="slug" className="text-xs font-medium text-muted-foreground">
                                    URL Slug
                                </Label>
                                <div className="flex">
                                    <div className="flex items-center px-3 border border-r-0 rounded-l-md bg-muted text-muted-foreground text-sm">
                                        /
                                    </div>
                                    <Input
                                        id="slug"
                                        value={slug}
                                        onChange={(e) => handleSlugChange(e.target.value)}
                                        placeholder="page-slug"
                                        className="h-9 rounded-l-none"
                                    />
                                </div>
                            </div>

                            {/* Description */}
                            <div className="space-y-1.5">
                                <Label htmlFor="description" className="text-xs font-medium text-muted-foreground">
                                    Description
                                    <span className="text-muted-foreground/60 ml-1">(optional)</span>
                                </Label>
                                <Textarea
                                    id="description"
                                    value={description}
                                    onChange={(e) => setDescription(e.target.value)}
                                    placeholder="Brief description..."
                                    rows={2}
                                    className="resize-none text-sm"
                                />
                            </div>

                            {/* Keywords */}
                            <div className="space-y-1.5">
                                <Label className="text-xs font-medium text-muted-foreground">
                                    Keywords
                                    <span className="text-muted-foreground/60 ml-1">({keywords.length})</span>
                                </Label>
                                <div className="flex gap-2">
                                    <Input
                                        value={newKeyword}
                                        onChange={(e) => setNewKeyword(e.target.value)}
                                        onKeyDown={handleKeywordKeyDown}
                                        placeholder="Add keyword..."
                                        className="h-8 text-sm"
                                    />
                                    <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        onClick={handleAddKeyword}
                                        disabled={!newKeyword.trim()}
                                        className="h-8 px-2"
                                    >
                                        <Plus className="h-4 w-4" />
                                    </Button>
                                </div>
                                {keywords.length > 0 && (
                                    <div className="flex flex-wrap gap-1.5 pt-1">
                                        {keywords.map((keyword) => (
                                            <Badge
                                                key={keyword}
                                                variant="secondary"
                                                className="h-6 pl-2 pr-1 text-xs font-normal gap-1"
                                            >
                                                {keyword}
                                                <button
                                                    type="button"
                                                    onClick={() => handleRemoveKeyword(keyword)}
                                                    className="ml-0.5 hover:text-destructive rounded-full p-0.5 hover:bg-destructive/10"
                                                >
                                                    <X className="h-3 w-3" />
                                                </button>
                                            </Badge>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

                    {/* Footer */}
                    <div className="flex items-center justify-between px-4 py-3 border-t bg-muted/30">
                        <div className="flex gap-2">
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                    handleOpenChange(false);
                                    setTimeout(() => onAddChild?.(), 100);
                                }}
                                className="h-8"
                            >
                                <Plus className="mr-1 h-3.5 w-3.5" />
                                Add Child
                            </Button>
                            {isEditMode && !isRootNode && (
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={handleDelete}
                                    className="h-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                                >
                                    <Trash2 className="h-3.5 w-3.5" />
                                </Button>
                            )}
                        </div>
                        <div className="flex items-center gap-2">
                            {isEditMode && !isRootNode && (
                                <span className="text-xs text-muted-foreground">Auto-saved</span>
                            )}
                            {!isEditMode && (
                                <Button
                                    size="sm"
                                    onClick={handleSave}
                                    disabled={!title.trim() || !slug.trim() || isLoading}
                                    className="h-8"
                                >
                                    Create
                                </Button>
                            )}
                        </div>
                    </div>
                </DialogContent>
            </Dialog>

            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Node</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete &quot;{node?.title}&quot;? This will also
                            delete all child nodes. This action cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={confirmDelete}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                            Delete
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
}
