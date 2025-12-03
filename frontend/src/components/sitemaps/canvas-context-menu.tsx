"use client";

import { useEffect } from "react";
import { SitemapNode } from "@/models/sitemaps";
import { Plus, Pencil, Trash2, GitBranchPlus, ExternalLink, Upload } from "lucide-react";
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
    const canView = selectedNode && selectedNode.contentStatus === "published" && siteUrl && !isMultiSelect;
    const hasWPNodes = targetNodes.some((n) => n.wpPageId != null);
    const hasModifiedNodes = targetNodes.some((n) => n.isModified);

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
                            label: isMultiSelect ? `Update ${targetNodes.filter((n) => n.isModified).length}` : "Update",
                            onClick: () => {
                                const modifiedNodeIds = targetNodes.filter((n) => n.isModified).map((n) => n.id);
                                onUpdateToWP(modifiedNodeIds);
                                onClose();
                            },
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
