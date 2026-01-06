"use client";

import { useEffect } from "react";
import { SitemapNode } from "@/models/sitemaps";
import { Plus, Pencil, Trash2, GitBranchPlus, ExternalLink, Upload, FileText, FileX, Trash, Globe } from "lucide-react";
import { RiWordpressLine } from "@remixicon/react";
import { cn } from "@/lib/utils";
import { BrowserOpenURL } from "@/wailsjs/wailsjs/runtime/runtime";

interface MenuPosition {
    x: number;
    y: number;
}

interface CanvasContextMenuProps {
    selectedNode: SitemapNode | null;
    selectedNodes?: SitemapNode[];
    position: MenuPosition | null;
    siteUrl?: string;
    onClose: () => void;
    onAddNode: (parentId?: number) => void;
    onAddOrphanNode: () => void;
    onEditNode: (node: SitemapNode) => void;
    onDeleteNode: (nodeId: number) => void;
    onAddChildNode: (parentId: number) => void;
    onSyncFromWP?: (nodeIds: number[]) => void;
    onUpdateToWP?: (nodeIds: number[]) => void;
    onGenerateContent?: (nodes: SitemapNode[]) => void;
    onPublish?: (nodeId: number) => void;
    onUnpublish?: (nodeId: number) => void;
    onBatchPublish?: (nodeIds: number[]) => void;
    onBatchUnpublish?: (nodeIds: number[]) => void;
    onDeleteFromWP?: (nodes: SitemapNode[]) => void;
}

export function CanvasContextMenu({
    selectedNode,
    selectedNodes = [],
    position,
    siteUrl,
    onClose,
    onAddNode,
    onAddOrphanNode,
    onEditNode,
    onDeleteNode,
    onAddChildNode,
    onSyncFromWP,
    onUpdateToWP,
    onGenerateContent,
    onPublish,
    onUnpublish,
    onBatchPublish,
    onBatchUnpublish,
    onDeleteFromWP,
}: CanvasContextMenuProps) {
    // Close on click outside or escape
    useEffect(() => {
        const handleClick = () => onClose();
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === "Escape") onClose();
        };

        if (position) {
            document.addEventListener("click", handleClick);
            document.addEventListener("keydown", handleKeyDown);
        }

        return () => {
            document.removeEventListener("click", handleClick);
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [position, onClose]);

    if (!position) return null;

    // Build full URL for the node
    const getFullUrl = (node: SitemapNode) => {
        if (!siteUrl) return null;
        const baseUrl = siteUrl.replace(/\/$/, "");
        if (node.isRoot) return baseUrl;
        return `${baseUrl}${node.path}`;
    };

    // Get all nodes to operate on (multiselect or single)
    const targetNodes = selectedNodes.length > 1 ? selectedNodes : selectedNode ? [selectedNode] : [];
    const isMultiSelect = selectedNodes.length > 1;

    // Check capabilities
    const canView = selectedNode && selectedNode.publishStatus === "published" && siteUrl && !isMultiSelect;
    const hasWPNodes = targetNodes.some((n) => n.wpPageId != null);
    const hasModifiedNodes = targetNodes.some((n) => n.isModifiedLocally);
    // Check if there are nodes that can be generated (not root, not already generated)
    const generatableNodes = targetNodes.filter(
        (n) => !n.isRoot && n.generationStatus !== "generated"
    );
    // Check if node can be published (has WP content and is draft/pending)
    const canPublish = selectedNode && !isMultiSelect &&
        selectedNode.wpPageId != null &&
        (selectedNode.publishStatus === "draft" || selectedNode.publishStatus === "pending");
    // Check if node can be unpublished (has WP content and is published)
    const canUnpublish = selectedNode && !isMultiSelect &&
        selectedNode.wpPageId != null &&
        selectedNode.publishStatus === "published";

    // Batch publish/unpublish - filter nodes that can be published or unpublished
    const publishableNodes = targetNodes.filter(
        (n) => n.wpPageId != null && (n.publishStatus === "draft" || n.publishStatus === "pending")
    );
    const unpublishableNodes = targetNodes.filter(
        (n) => n.wpPageId != null && n.publishStatus === "published"
    );
    // Nodes that can be deleted from WordPress (have wpPageId and not root)
    const deletableFromWPNodes = targetNodes.filter(
        (n) => n.wpPageId != null && !n.isRoot
    );

    const menuItems = selectedNode
        ? [
              ...(canView
                  ? [
                        {
                            icon: ExternalLink,
                            label: "View",
                            onClick: () => {
                                const url = getFullUrl(selectedNode);
                                if (url) BrowserOpenURL(url);
                                onClose();
                            },
                        },
                    ]
                  : []),
              {
                  icon: Pencil,
                  label: "Edit",
                  onClick: () => {
                      onEditNode(selectedNode);
                      onClose();
                  },
              },
              {
                  icon: GitBranchPlus,
                  label: "Add Child",
                  onClick: () => {
                      onAddChildNode(selectedNode.id);
                      onClose();
                  },
              },
              // Generate content option
              ...(generatableNodes.length > 0 && onGenerateContent
                  ? [
                        {
                            icon: FileText,
                            label: isMultiSelect
                                ? `Generate Content (${generatableNodes.length})`
                                : "Generate Content",
                            onClick: () => {
                                onGenerateContent(generatableNodes);
                                onClose();
                            },
                            separator: true,
                        },
                    ]
                  : []),
              // Sync operations for WP-linked nodes
              ...(hasWPNodes && onSyncFromWP
                  ? [
                        {
                            icon: RiWordpressLine,
                            label: isMultiSelect ? `Sync ${targetNodes.filter((n) => n.wpPageId != null).length}` : "Sync",
                            onClick: () => {
                                const wpNodeIds = targetNodes.filter((n) => n.wpPageId != null).map((n) => n.id);
                                onSyncFromWP(wpNodeIds);
                                onClose();
                            },
                            separator: true,
                        },
                    ]
                  : []),
              ...(hasModifiedNodes && onUpdateToWP
                  ? [
                        {
                            icon: Upload,
                            label: isMultiSelect ? `Update ${targetNodes.filter((n) => n.isModifiedLocally).length}` : "Update",
                            onClick: () => {
                                const modifiedNodeIds = targetNodes.filter((n) => n.isModifiedLocally).map((n) => n.id);
                                onUpdateToWP(modifiedNodeIds);
                                onClose();
                            },
                        },
                    ]
                  : []),
              // Single node publish option for draft/pending nodes
              ...(canPublish && onPublish
                  ? [
                        {
                            icon: Globe,
                            label: "Publish",
                            onClick: () => {
                                onPublish(selectedNode.id);
                                onClose();
                            },
                        },
                    ]
                  : []),
              // Batch publish for multi-select
              ...(isMultiSelect && publishableNodes.length > 0 && onBatchPublish
                  ? [
                        {
                            icon: Globe,
                            label: `Publish (${publishableNodes.length})`,
                            onClick: () => {
                                onBatchPublish(publishableNodes.map((n) => n.id));
                                onClose();
                            },
                        },
                    ]
                  : []),
              // Single node unpublish option for published nodes
              ...(canUnpublish && onUnpublish
                  ? [
                        {
                            icon: FileX,
                            label: "Unpublish (Draft)",
                            onClick: () => {
                                onUnpublish(selectedNode.id);
                                onClose();
                            },
                        },
                    ]
                  : []),
              // Batch unpublish for multi-select
              ...(isMultiSelect && unpublishableNodes.length > 0 && onBatchUnpublish
                  ? [
                        {
                            icon: FileX,
                            label: `Unpublish (${unpublishableNodes.length})`,
                            onClick: () => {
                                onBatchUnpublish(unpublishableNodes.map((n) => n.id));
                                onClose();
                            },
                        },
                    ]
                  : []),
              // Delete from WordPress option
              ...(deletableFromWPNodes.length > 0 && onDeleteFromWP
                  ? [
                        {
                            icon: Trash,
                            label: isMultiSelect
                                ? `Delete from WP (${deletableFromWPNodes.length})`
                                : "Delete from WordPress",
                            onClick: () => {
                                onDeleteFromWP(deletableFromWPNodes);
                                onClose();
                            },
                            destructive: true,
                            separator: true,
                        },
                    ]
                  : []),
              ...(selectedNode.isRoot
                  ? []
                  : [
                        {
                            icon: Trash2,
                            label: "Delete",
                            onClick: () => {
                                onDeleteNode(selectedNode.id);
                                onClose();
                            },
                            destructive: true,
                            separator: true,
                        },
                    ]),
          ]
        : [
              {
                  icon: GitBranchPlus,
                  label: "Add Node to Root",
                  onClick: () => {
                      onAddNode();
                      onClose();
                  },
              },
              {
                  icon: Plus,
                  label: "Add Standalone Node",
                  onClick: () => {
                      onAddOrphanNode();
                      onClose();
                  },
              },
          ];

    return (
        <div
            className="fixed z-50 min-w-[160px] bg-popover border rounded-md shadow-md py-1 animate-in fade-in-0 zoom-in-95"
            style={{
                left: position.x,
                top: position.y,
            }}
            onClick={(e) => e.stopPropagation()}
        >
            {menuItems.map((item, index) => (
                <div key={index}>
                    {item.separator && <div className="h-px bg-border my-1" />}
                    <button
                        className={cn(
                            "w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent transition-colors text-left",
                            item.destructive && "text-destructive hover:text-destructive"
                        )}
                        onClick={item.onClick}
                    >
                        <item.icon className="h-4 w-4" />
                        {item.label}
                    </button>
                </div>
            ))}
        </div>
    );
}
