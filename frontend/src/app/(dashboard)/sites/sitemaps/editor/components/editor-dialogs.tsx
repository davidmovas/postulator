"use client";

import { RefObject } from "react";
import { Node } from "@xyflow/react";
import { HotkeyConfig } from "@/hooks/use-hotkeys";
import { SitemapNode, GenerationTask } from "@/models/sitemaps";
import { PlannedLink, LinkGraph, LinkStatus } from "@/models/linking";
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
import { NodeEditDialog } from "@/components/sitemaps/node-edit-dialog";
import { NodeLinksDialog } from "@/components/sitemaps/node-links-dialog";
import { CanvasContextMenu } from "@/components/sitemaps/canvas-context-menu";
import { EdgeContextMenu } from "@/components/sitemaps/edge-context-menu";
import { LinkContextMenu } from "@/components/sitemaps/link-context-menu";
import { HotkeysDialog } from "@/components/sitemaps/hotkeys-dialog";
import { BulkCreateDialog } from "@/components/sitemaps/bulk-create-dialog";
import { CommandPalette } from "@/components/sitemaps/command-palette";
import { ImportDialog } from "@/components/sitemaps/import-dialog";
import { ScanDialog } from "@/components/sitemaps/scan-dialog";
import { GenerateDialog } from "@/components/sitemaps/generate-dialog";
import { PageGenerateDialog } from "@/components/sitemaps/page-generate-dialog";
import { SuggestLinksDialog } from "@/components/sitemaps/suggest-links-dialog";
import { ApplyLinksDialog } from "@/components/sitemaps/apply-links-dialog";

interface EditorDialogsProps {
    sitemapId: number;
    sitemapNodes: SitemapNode[];
    nodes: Node[];

    // Unsaved changes dialog
    showUnsavedDialog: boolean;
    setShowUnsavedDialog: (open: boolean) => void;
    confirmNavigation: () => void;

    // Node operations
    selectedNode: SitemapNode | null;
    editDialogOpen: boolean;
    setEditDialogOpen: (open: boolean) => void;
    createDialogOpen: boolean;
    setCreateDialogOpen: (open: boolean) => void;
    parentNodeId: number | null;
    onUpdateNode: (node: SitemapNode) => Promise<void>;
    onDeleteNode: (id: number) => void;
    onAddChild: (parentId: number) => void;
    onCreateNode: (node: SitemapNode) => Promise<void>;
    getSelectedSitemapNodes: () => SitemapNode[];

    // Bulk create
    bulkCreateDialogOpen: boolean;
    setBulkCreateDialogOpen: (open: boolean) => void;
    onBulkCreate: (paths: string[]) => Promise<void>;

    // Import dialog
    importDialogOpen: boolean;
    setImportDialogOpen: (open: boolean) => void;
    loadData: () => Promise<void>;

    // Scan dialog
    scanDialogOpen: boolean;
    setScanDialogOpen: (open: boolean) => void;

    // Generate dialog
    generateDialogOpen: boolean;
    setGenerateDialogOpen: (open: boolean) => void;

    // Page generate dialog
    pageGenerateDialogOpen: boolean;
    setPageGenerateDialogOpen: (open: boolean) => void;
    contextMenuSelectedNodes: SitemapNode[];
    setContextMenuSelectedNodes: (nodes: SitemapNode[]) => void;
    setActiveGenerationTask: (task: GenerationTask | null) => void;
    activeGenerationTask: GenerationTask | null;
    hasApprovedLinks: boolean;

    // Suggest links dialog
    suggestLinksDialogOpen: boolean;
    setSuggestLinksDialogOpen: (open: boolean) => void;
    planId: number | null;
    loadLinkingData: () => Promise<void>;

    // Apply links dialog
    applyLinksDialogOpen: boolean;
    setApplyLinksDialogOpen: (open: boolean) => void;
    approvedLinks: PlannedLink[];

    // Command palette
    commandPaletteOpen: boolean;
    setCommandPaletteOpen: (open: boolean) => void;
    hasUnsavedChanges: boolean;
    onSave: () => Promise<void>;
    onAutoLayout: () => void;
    onAddNode: () => void;
    onGeneratePages: () => void;
    onDeleteSelected: () => void;
    searchInputRef: RefObject<HTMLInputElement | null>;

    // Hotkeys dialog
    hotkeysDialogOpen: boolean;
    setHotkeysDialogOpen: (open: boolean) => void;
    hotkeys: HotkeyConfig[];

    // Node links dialog
    nodeLinksDialogOpen: boolean;
    setNodeLinksDialogOpen: (open: boolean) => void;
    selectedNodeForLinks: SitemapNode | null;
    linkGraph: LinkGraph | null;
    onApproveLink: (id: number) => Promise<boolean>;
    onRejectLink: (id: number) => Promise<boolean>;
    onRemoveLink: (id: number) => Promise<boolean>;
    onGoToNode: (nodeId: number) => void;

    // Canvas context menu
    contextMenuNode: SitemapNode | null;
    contextMenuPosition: { x: number; y: number } | null;
    siteUrl?: string;
    closeContextMenu: () => void;
    onEditNode: () => void;
    onAddOrphanNode: (position?: { x: number; y: number }) => void;
    onSyncFromWP: (nodeIds: number[]) => Promise<void>;
    onUpdateToWP: (nodeIds: number[]) => Promise<void>;
    onGenerateContent: (nodes: SitemapNode[]) => void;
    onPublish: (nodeId: number) => Promise<void>;
    onUnpublish: (nodeId: number) => Promise<void>;

    // Edge context menu
    edgeContextMenuPosition: { x: number; y: number } | null;
    closeEdgeContextMenu: () => void;
    onDeleteEdge: () => void;

    // Link context menu
    linkContextMenu: { linkId: number; position: { x: number; y: number }; status: LinkStatus } | null;
    closeLinkContextMenu: () => void;
}

export function EditorDialogs({
    sitemapId,
    sitemapNodes,
    nodes,
    showUnsavedDialog,
    setShowUnsavedDialog,
    confirmNavigation,
    selectedNode,
    editDialogOpen,
    setEditDialogOpen,
    createDialogOpen,
    setCreateDialogOpen,
    parentNodeId,
    onUpdateNode,
    onDeleteNode,
    onAddChild,
    onCreateNode,
    getSelectedSitemapNodes,
    bulkCreateDialogOpen,
    setBulkCreateDialogOpen,
    onBulkCreate,
    importDialogOpen,
    setImportDialogOpen,
    loadData,
    scanDialogOpen,
    setScanDialogOpen,
    generateDialogOpen,
    setGenerateDialogOpen,
    pageGenerateDialogOpen,
    setPageGenerateDialogOpen,
    contextMenuSelectedNodes,
    setContextMenuSelectedNodes,
    setActiveGenerationTask,
    activeGenerationTask,
    hasApprovedLinks,
    suggestLinksDialogOpen,
    setSuggestLinksDialogOpen,
    planId,
    loadLinkingData,
    applyLinksDialogOpen,
    setApplyLinksDialogOpen,
    approvedLinks,
    commandPaletteOpen,
    setCommandPaletteOpen,
    hasUnsavedChanges,
    onSave,
    onAutoLayout,
    onAddNode,
    onGeneratePages,
    onDeleteSelected,
    searchInputRef,
    hotkeysDialogOpen,
    setHotkeysDialogOpen,
    hotkeys,
    nodeLinksDialogOpen,
    setNodeLinksDialogOpen,
    selectedNodeForLinks,
    linkGraph,
    onApproveLink,
    onRejectLink,
    onRemoveLink,
    onGoToNode,
    contextMenuNode,
    contextMenuPosition,
    siteUrl,
    closeContextMenu,
    onEditNode,
    onAddOrphanNode,
    onSyncFromWP,
    onUpdateToWP,
    onGenerateContent,
    onPublish,
    onUnpublish,
    edgeContextMenuPosition,
    closeEdgeContextMenu,
    onDeleteEdge,
    linkContextMenu,
    closeLinkContextMenu,
}: EditorDialogsProps) {
    return (
        <>
            {/* Canvas and edge context menus */}
            <CanvasContextMenu
                selectedNode={contextMenuNode}
                selectedNodes={getSelectedSitemapNodes()}
                position={contextMenuPosition}
                siteUrl={siteUrl}
                onClose={closeContextMenu}
                onAddNode={onAddNode}
                onAddOrphanNode={onAddOrphanNode}
                onEditNode={onEditNode}
                onDeleteNode={onDeleteNode}
                onAddChildNode={onAddChild}
                onSyncFromWP={onSyncFromWP}
                onUpdateToWP={onUpdateToWP}
                onGenerateContent={onGenerateContent}
                onPublish={onPublish}
                onUnpublish={onUnpublish}
            />
            <EdgeContextMenu
                position={edgeContextMenuPosition}
                onClose={closeEdgeContextMenu}
                onDeleteEdge={onDeleteEdge}
            />

            {/* Link context menu for links mode */}
            {linkContextMenu && (
                <LinkContextMenu
                    linkId={linkContextMenu.linkId}
                    position={linkContextMenu.position}
                    status={linkContextMenu.status}
                    onApprove={onApproveLink}
                    onReject={onRejectLink}
                    onRemove={onRemoveLink}
                    onClose={closeLinkContextMenu}
                />
            )}

            {/* Node links dialog for links mode */}
            <NodeLinksDialog
                open={nodeLinksDialogOpen}
                onOpenChange={setNodeLinksDialogOpen}
                node={selectedNodeForLinks}
                linkGraph={linkGraph}
                sitemapNodes={sitemapNodes}
                onApproveLink={onApproveLink}
                onRejectLink={onRejectLink}
                onRemoveLink={onRemoveLink}
                onGoToNode={onGoToNode}
            />

            {/* Node edit dialog */}
            {selectedNode && (
                <NodeEditDialog
                    open={editDialogOpen}
                    onOpenChange={setEditDialogOpen}
                    node={selectedNode}
                    onUpdate={onUpdateNode}
                    onDelete={() => onDeleteNode(selectedNode.id)}
                    onAddChild={() => onAddChild(selectedNode.id)}
                />
            )}

            {/* Node create dialog */}
            <NodeEditDialog
                open={createDialogOpen}
                onOpenChange={setCreateDialogOpen}
                sitemapId={sitemapId}
                parentId={parentNodeId}
                onCreate={onCreateNode}
            />

            {/* Unsaved changes dialog */}
            <AlertDialog open={showUnsavedDialog} onOpenChange={setShowUnsavedDialog}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
                        <AlertDialogDescription>
                            You have unsaved layout changes. Are you sure you want to leave?
                            Your changes will be lost.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Stay</AlertDialogCancel>
                        <AlertDialogAction onClick={confirmNavigation}>
                            Leave Without Saving
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            {/* Bulk create dialog */}
            <BulkCreateDialog
                open={bulkCreateDialogOpen}
                onOpenChange={setBulkCreateDialogOpen}
                onSubmit={onBulkCreate}
            />

            {/* Import dialog */}
            <ImportDialog
                open={importDialogOpen}
                onOpenChange={setImportDialogOpen}
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            {/* Scan dialog */}
            <ScanDialog
                open={scanDialogOpen}
                onOpenChange={setScanDialogOpen}
                mode="add"
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            {/* Generate dialog */}
            <GenerateDialog
                open={generateDialogOpen}
                onOpenChange={setGenerateDialogOpen}
                mode="add"
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            {/* Page generate dialog */}
            <PageGenerateDialog
                open={pageGenerateDialogOpen}
                onOpenChange={(open) => {
                    setPageGenerateDialogOpen(open);
                    if (!open) {
                        setContextMenuSelectedNodes([]);
                    }
                }}
                sitemapId={sitemapId}
                selectedNodes={
                    contextMenuSelectedNodes.length > 0
                        ? contextMenuSelectedNodes
                        : getSelectedSitemapNodes()
                }
                allNodes={sitemapNodes}
                onSuccess={() => loadData()}
                onTaskStarted={setActiveGenerationTask}
                activeTask={activeGenerationTask}
                hasApprovedLinks={hasApprovedLinks}
            />

            {/* Suggest links dialog */}
            {planId && (
                <SuggestLinksDialog
                    open={suggestLinksDialogOpen}
                    onOpenChange={setSuggestLinksDialogOpen}
                    planId={planId}
                    selectedNodes={getSelectedSitemapNodes()}
                    allNodes={sitemapNodes}
                    onSuccess={() => loadLinkingData()}
                />
            )}

            {/* Apply links dialog */}
            {planId && (
                <ApplyLinksDialog
                    open={applyLinksDialogOpen}
                    onOpenChange={setApplyLinksDialogOpen}
                    planId={planId}
                    approvedLinks={approvedLinks}
                    selectedNodes={getSelectedSitemapNodes()}
                    sitemapNodes={sitemapNodes}
                    onSuccess={() => loadLinkingData()}
                />
            )}

            {/* Command palette */}
            <CommandPalette
                open={commandPaletteOpen}
                onOpenChange={setCommandPaletteOpen}
                hasUnsavedChanges={hasUnsavedChanges}
                hasSelectedNodes={nodes.some((n) => n.selected)}
                onSave={onSave}
                onAutoLayout={onAutoLayout}
                onAddNode={onAddNode}
                onBulkCreate={() => setBulkCreateDialogOpen(true)}
                onImport={() => setImportDialogOpen(true)}
                onScan={() => setScanDialogOpen(true)}
                onGeneratePages={onGeneratePages}
                onFocusSearch={() => searchInputRef.current?.focus()}
                onDeleteSelected={onDeleteSelected}
                onShowHotkeys={() => setHotkeysDialogOpen(true)}
            />

            {/* Hotkeys dialog */}
            <HotkeysDialog
                hotkeys={hotkeys}
                open={hotkeysDialogOpen}
                onOpenChange={setHotkeysDialogOpen}
            />
        </>
    );
}
