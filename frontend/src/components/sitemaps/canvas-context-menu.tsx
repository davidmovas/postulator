"use client";

import { useCallback, useEffect, useState } from "react";
import { SitemapNode } from "@/models/sitemaps";
import { Plus, Pencil, Trash2, GitBranchPlus } from "lucide-react";
import { cn } from "@/lib/utils";

interface MenuPosition {
    x: number;
    y: number;
}

interface CanvasContextMenuProps {
    selectedNode: SitemapNode | null;
    position: MenuPosition | null;
    onClose: () => void;
    onAddNode: (parentId?: number) => void;
    onAddOrphanNode: () => void;
    onEditNode: (node: SitemapNode) => void;
    onDeleteNode: (nodeId: number) => void;
    onAddChildNode: (parentId: number) => void;
}

export function CanvasContextMenu({
    selectedNode,
    position,
    onClose,
    onAddNode,
    onAddOrphanNode,
    onEditNode,
    onDeleteNode,
    onAddChildNode,
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

    const menuItems = selectedNode
        ? [
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
