"use client";

import { useState, useMemo, useCallback } from "react";
import { SitemapNode } from "@/models/sitemaps";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
    ChevronRight,
    ChevronDown,
    Search,
    Plus,
    Home,
    Circle,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface SitemapSidebarProps {
    nodes: SitemapNode[];
    selectedNodeIds: Set<number>;
    onNodeSelect: (node: SitemapNode) => void;
    onNodesSelect: (nodeIds: number[]) => void;
    onAddChild: (parentId: number) => void;
    searchInputRef?: React.RefObject<HTMLInputElement | null>;
}

interface TreeNode extends SitemapNode {
    children: TreeNode[];
}

function buildTree(nodes: SitemapNode[]): TreeNode[] {
    const nodeMap = new Map<number, TreeNode>();
    const roots: TreeNode[] = [];

    // Create TreeNode objects
    nodes.forEach((node) => {
        nodeMap.set(node.id, { ...node, children: [] });
    });

    // Build tree structure
    nodes.forEach((node) => {
        const treeNode = nodeMap.get(node.id)!;
        if (node.parentId) {
            const parent = nodeMap.get(node.parentId);
            if (parent) {
                parent.children.push(treeNode);
            } else {
                roots.push(treeNode);
            }
        } else {
            roots.push(treeNode);
        }
    });

    // Sort children by position
    const sortChildren = (nodes: TreeNode[]) => {
        nodes.sort((a, b) => a.position - b.position);
        nodes.forEach((node) => sortChildren(node.children));
    };

    sortChildren(roots);
    return roots;
}

// Get all descendant IDs of a node
function getAllDescendantIds(node: TreeNode): number[] {
    const ids: number[] = [node.id];
    for (const child of node.children) {
        ids.push(...getAllDescendantIds(child));
    }
    return ids;
}

interface TreeItemProps {
    node: TreeNode;
    level: number;
    selectedNodeIds: Set<number>;
    onNodeClick: (node: TreeNode, e: React.MouseEvent) => void;
    onAddChild: (parentId: number) => void;
    searchQuery: string;
}

function TreeItem({
    node,
    level,
    selectedNodeIds,
    onNodeClick,
    onAddChild,
    searchQuery,
}: TreeItemProps) {
    const [expanded, setExpanded] = useState(true);
    const hasChildren = node.children.length > 0;
    const isSelected = selectedNodeIds.has(node.id);

    const matchesSearch =
        !searchQuery ||
        node.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        node.slug.toLowerCase().includes(searchQuery.toLowerCase());

    const hasMatchingDescendant = useMemo(() => {
        if (!searchQuery) return true;

        const checkDescendants = (n: TreeNode): boolean => {
            if (
                n.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
                n.slug.toLowerCase().includes(searchQuery.toLowerCase())
            ) {
                return true;
            }
            return n.children.some(checkDescendants);
        };

        return checkDescendants(node);
    }, [node, searchQuery]);

    if (!matchesSearch && !hasMatchingDescendant) {
        return null;
    }

    const getIcon = () => {
        if (node.isRoot) {
            return <Home className="h-4 w-4 text-primary" />;
        }
        // Status-based colors for the circle indicator
        const getStatusColor = () => {
            switch (node.contentStatus) {
                case "published":
                    return "text-green-500 fill-green-500";
                case "draft":
                    return "text-yellow-500 fill-yellow-500";
                case "pending":
                    return "text-blue-500 fill-blue-500";
                default:
                    return "text-muted-foreground/50";
            }
        };
        return <Circle className={cn("h-2 w-2", getStatusColor())} />;
    };

    return (
        <div>
            <div
                className={cn(
                    "flex items-center gap-1 py-1 px-2 rounded-md cursor-pointer group",
                    isSelected ? "bg-accent" : "hover:bg-accent/50",
                    !matchesSearch && "opacity-50"
                )}
                style={{ paddingLeft: `${level * 16 + 8}px` }}
                onClick={(e) => onNodeClick(node, e)}
            >
                <button
                    type="button"
                    className={cn(
                        "p-0.5 rounded hover:bg-accent-foreground/10",
                        !hasChildren && "invisible"
                    )}
                    onClick={(e) => {
                        e.stopPropagation();
                        setExpanded(!expanded);
                    }}
                >
                    {expanded ? (
                        <ChevronDown className="h-3 w-3" />
                    ) : (
                        <ChevronRight className="h-3 w-3" />
                    )}
                </button>

                <div className="flex items-center gap-2 flex-1 min-w-0">
                    {getIcon()}
                    <div className="flex flex-col min-w-0">
                        <span className="text-sm truncate">{node.isRoot ? "/" : node.title}</span>
                        {!node.isRoot && (
                            <span className="text-[10px] text-muted-foreground truncate">/{node.slug}</span>
                        )}
                    </div>
                </div>

                <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 opacity-0 group-hover:opacity-100"
                    onClick={(e) => {
                        e.stopPropagation();
                        onAddChild(node.id);
                    }}
                >
                    <Plus className="h-3 w-3" />
                </Button>
            </div>

            {expanded && hasChildren && (
                <div>
                    {node.children.map((child) => (
                        <TreeItem
                            key={child.id}
                            node={child}
                            level={level + 1}
                            selectedNodeIds={selectedNodeIds}
                            onNodeClick={onNodeClick}
                            onAddChild={onAddChild}
                            searchQuery={searchQuery}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}

export function SitemapSidebar({
    nodes,
    selectedNodeIds,
    onNodeSelect,
    onNodesSelect,
    onAddChild,
    searchInputRef,
}: SitemapSidebarProps) {
    const [searchQuery, setSearchQuery] = useState("");

    const tree = useMemo(() => buildTree(nodes), [nodes]);

    // Build a map for quick lookup
    const treeNodeMap = useMemo(() => {
        const map = new Map<number, TreeNode>();
        const addToMap = (node: TreeNode) => {
            map.set(node.id, node);
            node.children.forEach(addToMap);
        };
        tree.forEach(addToMap);
        return map;
    }, [tree]);

    const handleNodeClick = useCallback((node: TreeNode, e: React.MouseEvent) => {
        if (e.shiftKey) {
            // Shift+click: select node and all its descendants
            const allIds = getAllDescendantIds(node);

            // If already all selected, deselect all; otherwise select all
            const allSelected = allIds.every(id => selectedNodeIds.has(id));

            if (allSelected) {
                // Deselect all descendants
                const newSelection = Array.from(selectedNodeIds).filter(id => !allIds.includes(id));
                onNodesSelect(newSelection);
            } else {
                // Add all descendants to selection
                const newSelection = new Set(selectedNodeIds);
                allIds.forEach(id => newSelection.add(id));
                onNodesSelect(Array.from(newSelection));
            }
        } else if (e.ctrlKey || e.metaKey) {
            // Ctrl+click: toggle single node selection
            const newSelection = new Set(selectedNodeIds);
            if (newSelection.has(node.id)) {
                newSelection.delete(node.id);
            } else {
                newSelection.add(node.id);
            }
            onNodesSelect(Array.from(newSelection));
        } else {
            // Regular click: open edit dialog
            onNodeSelect(node);
        }
    }, [selectedNodeIds, onNodesSelect, onNodeSelect]);

    const selectedCount = selectedNodeIds.size;

    return (
        <div className="h-full flex flex-col bg-background">
            <div className="p-3 border-b">
                <div className="relative">
                    <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        ref={searchInputRef}
                        placeholder="Search nodes..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-8 h-8"
                    />
                </div>
            </div>

            <ScrollArea className="flex-1">
                <div className="p-2">
                    {tree.length === 0 ? (
                        <div className="text-center py-8 text-sm text-muted-foreground">
                            No nodes yet
                        </div>
                    ) : (
                        tree.map((node) => (
                            <TreeItem
                                key={node.id}
                                node={node}
                                level={0}
                                selectedNodeIds={selectedNodeIds}
                                onNodeClick={handleNodeClick}
                                onAddChild={onAddChild}
                                searchQuery={searchQuery}
                            />
                        ))
                    )}
                </div>
            </ScrollArea>

            <div className="p-3 border-t text-xs text-muted-foreground text-center">
                {selectedCount > 0 ? (
                    <span>{selectedCount} selected / {nodes.length} nodes</span>
                ) : (
                    <span>{nodes.length} node{nodes.length !== 1 ? "s" : ""}</span>
                )}
            </div>
        </div>
    );
}
