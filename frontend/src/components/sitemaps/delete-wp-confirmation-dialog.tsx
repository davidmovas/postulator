"use client";

import { useMemo, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { SitemapNode } from "@/models/sitemaps";
import { AlertTriangle, Trash } from "lucide-react";

interface DeleteWPConfirmationDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    nodes: SitemapNode[];
    allNodes: SitemapNode[];
    onConfirm: () => Promise<void>;
}

export function DeleteWPConfirmationDialog({
    open,
    onOpenChange,
    nodes,
    allNodes,
    onConfirm,
}: DeleteWPConfirmationDialogProps) {
    const [loading, setLoading] = useState(false);

    // Calculate children that will be reparented
    const childrenInfo = useMemo(() => {
        const nodeIds = new Set(nodes.map((n) => n.id));
        let totalChildren = 0;

        for (const node of nodes) {
            // Find direct children of this node
            const children = allNodes.filter(
                (n) => n.parentId === node.id && !nodeIds.has(n.id)
            );
            totalChildren += children.length;
        }

        return totalChildren;
    }, [nodes, allNodes]);

    const handleConfirm = async () => {
        setLoading(true);
        try {
            await onConfirm();
            onOpenChange(false);
        } finally {
            setLoading(false);
        }
    };

    const nodeCount = nodes.length;
    const isSingle = nodeCount === 1;

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Trash className="h-5 w-5 text-destructive" />
                        Delete from WordPress
                    </DialogTitle>
                    <DialogDescription>
                        {isSingle
                            ? `You are about to delete "${nodes[0]?.title}" from WordPress.`
                            : `You are about to delete ${nodeCount} pages from WordPress.`}
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    <Alert variant="destructive">
                        <AlertTriangle className="h-4 w-4" />
                        <AlertDescription>
                            This action cannot be undone. The {isSingle ? "page" : "pages"} will be permanently
                            deleted from your WordPress site.
                        </AlertDescription>
                    </Alert>

                    {childrenInfo > 0 && (
                        <div className="text-sm text-muted-foreground bg-muted p-3 rounded-md">
                            <strong>{childrenInfo}</strong> child {childrenInfo === 1 ? "page" : "pages"} will be
                            moved to the parent level (like WordPress behavior).
                        </div>
                    )}

                    {nodeCount > 1 && (
                        <div className="text-sm max-h-32 overflow-y-auto border rounded-md p-2">
                            <div className="font-medium mb-1">Pages to delete:</div>
                            <ul className="list-disc list-inside space-y-0.5">
                                {nodes.map((node) => (
                                    <li key={node.id} className="text-muted-foreground truncate">
                                        {node.title}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    )}
                </div>

                <DialogFooter>
                    <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={loading}>
                        Cancel
                    </Button>
                    <Button variant="destructive" onClick={handleConfirm} disabled={loading}>
                        {loading ? "Deleting..." : isSingle ? "Delete" : `Delete ${nodeCount} Pages`}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
