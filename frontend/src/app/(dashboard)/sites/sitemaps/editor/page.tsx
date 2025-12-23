"use client";

import { useState, useEffect, useCallback, Suspense, useRef, useMemo } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
    ReactFlow,
    Background,
    Node,
    BackgroundVariant,
    Panel,
    NodeTypes,
    ReactFlowProvider,
    SelectionMode,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { useHotkeys, HotkeyConfig } from "@/hooks/use-hotkeys";
import { useSitemapEditorData } from "@/hooks/use-sitemap-editor-data";
import { useSitemapNodeOperations } from "@/hooks/use-sitemap-node-operations";
import { useSitemapWPOperations } from "@/hooks/use-sitemap-wp-operations";
import { useSitemapCanvas } from "@/hooks/use-sitemap-canvas";
import { sitemapService } from "@/services/sitemaps";
import { CreateNodeInput } from "@/models/sitemaps";
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
import {
    ResizableHandle,
    ResizablePanel,
    ResizablePanelGroup,
} from "@/components/ui/resizable";
import { SitemapNodeCard } from "@/components/sitemaps/sitemap-node-card";
import { NodeEditDialog } from "@/components/sitemaps/node-edit-dialog";
import { SitemapSidebar } from "@/components/sitemaps/sitemap-sidebar";
import { CanvasControls } from "@/components/sitemaps/canvas-controls";
import { CanvasContextMenu } from "@/components/sitemaps/canvas-context-menu";
import { EdgeContextMenu } from "@/components/sitemaps/edge-context-menu";
import { HotkeysDialog } from "@/components/sitemaps/hotkeys-dialog";
import { BulkCreateDialog } from "@/components/sitemaps/bulk-create-dialog";
import { CommandPalette } from "@/components/sitemaps/command-palette";
import { ImportDialog } from "@/components/sitemaps/import-dialog";
import { ScanDialog } from "@/components/sitemaps/scan-dialog";
import { GenerateDialog } from "@/components/sitemaps/generate-dialog";
import { PageGenerateDialog } from "@/components/sitemaps/page-generate-dialog";
import { EditorHeader } from "@/components/sitemaps/editor-header";
import { GenerationProgressPanel } from "@/components/sitemaps/generation-progress-panel";
import { createNodesFromPaths } from "@/lib/sitemap-utils";

const nodeTypes: NodeTypes = {
    sitemapNode: SitemapNodeCard,
};

function SitemapEditorFlow() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const siteId = Number(searchParams.get("id"));
    const sitemapId = Number(searchParams.get("sitemapId"));
    const reactFlowWrapper = useRef<HTMLDivElement>(null);
    const searchInputRef = useRef<HTMLInputElement>(null);

    const editorData = useSitemapEditorData({ siteId, sitemapId });
    const {
        site,
        sitemap,
        sitemapNodes,
        nodes,
        edges,
        setNodes,
        setEdges,
        onNodesChange,
        onEdgesChange,
        isLoading,
        hasUnsavedChanges,
        setHasUnsavedChanges,
        history,
        loadData,
        handleSavePositions,
        execute,
    } = editorData;

    const nodeOps = useSitemapNodeOperations({
        sitemapId,
        sitemapNodes,
        nodes,
        execute,
        loadData,
        refreshHistory: history.refreshState,
    });

    const wpOps = useSitemapWPOperations({
        siteId,
        sitemapId,
        execute,
        loadData,
    });

    const canvas = useSitemapCanvas({
        sitemapNodes,
        nodes,
        edges,
        setNodes,
        setEdges,
        execute,
        loadData,
        refreshHistory: history.refreshState,
        getAllDescendantIds: nodeOps.getAllDescendantIds,
        setParentNodeId: nodeOps.setParentNodeId,
        setCreateDialogOpen: nodeOps.setCreateDialogOpen,
        setSelectedNode: nodeOps.setSelectedNode,
        setEditDialogOpen: nodeOps.setEditDialogOpen,
        setHasUnsavedChanges,
    });

    const [bulkCreateDialogOpen, setBulkCreateDialogOpen] = useState(false);
    const [importDialogOpen, setImportDialogOpen] = useState(false);
    const [scanDialogOpen, setScanDialogOpen] = useState(false);
    const [generateDialogOpen, setGenerateDialogOpen] = useState(false);
    const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
    const [hotkeysDialogOpen, setHotkeysDialogOpen] = useState(false);
    const [showUnsavedDialog, setShowUnsavedDialog] = useState(false);
    const [pendingNavigation, setPendingNavigation] = useState<string | null>(null);

    const handleUndo = useCallback(async () => {
        const success = await history.undo();
        if (success) {
            await loadData(true);
        }
    }, [history, loadData]);

    const handleRedo = useCallback(async () => {
        const success = await history.redo();
        if (success) {
            await loadData(true);
        }
    }, [history, loadData]);

    const handleNavigateBack = useCallback(() => {
        if (wpOps.activeGenerationTask &&
            (wpOps.activeGenerationTask.status === "running" || wpOps.activeGenerationTask.status === "paused")) {
            return;
        }
        const backUrl = `/sites/sitemaps?id=${siteId}`;
        if (hasUnsavedChanges) {
            setPendingNavigation(backUrl);
            setShowUnsavedDialog(true);
        } else {
            router.push(backUrl);
        }
    }, [hasUnsavedChanges, router, siteId, wpOps.activeGenerationTask]);

    const confirmNavigation = useCallback(() => {
        setShowUnsavedDialog(false);
        if (pendingNavigation) {
            router.push(pendingNavigation);
        }
    }, [pendingNavigation, router]);

    const handleBulkCreate = useCallback(async (paths: string[]) => {
        if (!sitemapId) return;

        const createNodeFn = async (input: CreateNodeInput) => {
            const result = await sitemapService.createNode(input);
            return result;
        };

        try {
            for await (const progress of createNodesFromPaths(
                paths,
                sitemapId,
                sitemapNodes,
                createNodeFn
            )) {
            }
            await loadData();
        } catch (error) {
            console.error("Failed to create nodes:", error);
        }
    }, [sitemapId, sitemapNodes, loadData]);

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === "Tab" && !e.repeat) {
                const target = e.target as HTMLElement;
                if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable) {
                    return;
                }
                e.preventDefault();
                setCommandPaletteOpen(true);
            }
        };

        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, []);

    const hotkeys = useMemo<HotkeyConfig[]>(() => [
        {
            key: "s",
            ctrl: true,
            description: "Save layout",
            category: "General",
            action: () => {
                if (hasUnsavedChanges) handleSavePositions();
            },
        },
        {
            key: "l",
            ctrl: true,
            description: "Auto layout",
            category: "Layout",
            action: canvas.handleAutoLayout,
        },
        {
            key: "n",
            ctrl: true,
            description: "Add new node",
            category: "Nodes",
            action: () => nodeOps.handleAddNode(),
        },
        {
            key: "Delete",
            description: "Delete selected nodes",
            category: "Nodes",
            action: nodeOps.handleDeleteSelectedNodes,
        },
        {
            key: "Backspace",
            description: "Delete selected nodes",
            category: "Nodes",
            action: nodeOps.handleDeleteSelectedNodes,
        },
        {
            key: "Escape",
            description: "Deselect all",
            category: "Selection",
            action: () => {
                setNodes((nds) => nds.map((n) => ({ ...n, selected: false })));
            },
        },
        {
            key: "f",
            shift: true,
            description: "Focus search",
            category: "Navigation",
            action: () => {
                searchInputRef.current?.focus();
            },
        },
        {
            key: "b",
            ctrl: true,
            description: "Bulk create nodes",
            category: "Nodes",
            action: () => setBulkCreateDialogOpen(true),
        },
        {
            key: "i",
            ctrl: true,
            description: "Import from file",
            category: "Nodes",
            action: () => setImportDialogOpen(true),
        },
        {
            key: "k",
            ctrl: true,
            description: "Scan from WordPress",
            category: "Nodes",
            action: () => setScanDialogOpen(true),
        },
        {
            key: "z",
            ctrl: true,
            description: "Undo",
            category: "History",
            action: handleUndo,
        },
        {
            key: "z",
            ctrl: true,
            shift: true,
            description: "Redo",
            category: "History",
            action: handleRedo,
        },
        {
            key: "g",
            ctrl: true,
            description: "Generate page content",
            category: "Content",
            action: () => wpOps.setPageGenerateDialogOpen(true),
        },
        {
            key: "Tab",
            description: "Command palette",
            category: "General",
            action: () => {},
        },
    ], [hasUnsavedChanges, handleSavePositions, canvas.handleAutoLayout, nodeOps, setNodes, handleUndo, handleRedo, wpOps]);

    useHotkeys(hotkeys);

    if (isLoading && !sitemap) {
        return (
            <div className="h-screen flex items-center justify-center">
                <div className="text-muted-foreground">Loading sitemap...</div>
            </div>
        );
    }

    return (
        <div className="h-full flex flex-col overflow-hidden">
            <EditorHeader
                site={site}
                sitemap={sitemap}
                hasUnsavedChanges={hasUnsavedChanges}
                canUndo={history.canUndo}
                canRedo={history.canRedo}
                activeGenerationTask={wpOps.activeGenerationTask}
                hotkeys={hotkeys}
                onNavigateBack={handleNavigateBack}
                onUndo={handleUndo}
                onRedo={handleRedo}
                onAutoLayout={canvas.handleAutoLayout}
                onAddNode={() => nodeOps.handleAddNode()}
                onBulkCreate={() => setBulkCreateDialogOpen(true)}
                onImport={() => setImportDialogOpen(true)}
                onScan={() => setScanDialogOpen(true)}
                onGenerateStructure={() => setGenerateDialogOpen(true)}
                onGeneratePages={() => wpOps.setPageGenerateDialogOpen(true)}
                onSave={handleSavePositions}
            />

            {wpOps.activeGenerationTask && (
                <GenerationProgressPanel
                    task={wpOps.activeGenerationTask}
                    onPause={wpOps.handlePauseGeneration}
                    onResume={wpOps.handleResumeGeneration}
                    onCancel={wpOps.handleCancelGeneration}
                />
            )}

            <ResizablePanelGroup direction="horizontal" className="flex-1 min-h-0">
                <ResizablePanel defaultSize={20} minSize={15} maxSize={35} className="overflow-hidden">
                    <SitemapSidebar
                        nodes={sitemapNodes}
                        selectedNodeIds={canvas.sidebarSelectedNodeIds}
                        onNodeSelect={(node) => {
                            nodeOps.setSelectedNode(node);
                            nodeOps.setEditDialogOpen(true);
                        }}
                        onNodesSelect={canvas.handleSidebarNodesSelect}
                        onAddChild={nodeOps.handleAddChild}
                        searchInputRef={searchInputRef}
                    />
                </ResizablePanel>

                <ResizableHandle withHandle />

                <ResizablePanel defaultSize={80}>
                    <div className="h-full" ref={reactFlowWrapper}>
                        <ReactFlow
                            nodes={nodes}
                            edges={edges}
                            onNodesChange={onNodesChange}
                            onEdgesChange={onEdgesChange}
                            onConnectStart={canvas.onConnectStart}
                            onConnect={canvas.onConnect}
                            onConnectEnd={canvas.onConnectEnd}
                            onNodeClick={canvas.onNodeClick}
                            onNodeDoubleClick={canvas.onNodeDoubleClick}
                            onNodeContextMenu={canvas.onNodeContextMenu}
                            onPaneContextMenu={canvas.onPaneContextMenu}
                            onEdgeContextMenu={canvas.onEdgeContextMenu}
                            onPaneClick={() => {
                                canvas.closeContextMenu();
                                canvas.closeEdgeContextMenu();
                            }}
                            onSelectionChange={canvas.handleSelectionChange}
                            nodeTypes={nodeTypes}
                            fitView
                            fitViewOptions={{ padding: 0.3, maxZoom: 0.8 }}
                            snapToGrid
                            snapGrid={[15, 15]}
                            minZoom={0.1}
                            maxZoom={1.5}
                            defaultViewport={{ x: 0, y: 0, zoom: 0.7 }}
                            selectionMode={SelectionMode.Partial}
                            selectionOnDrag={false}
                            panOnDrag
                        >
                            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
                            <Panel position="bottom-right">
                                <CanvasControls />
                            </Panel>
                        </ReactFlow>
                    </div>
                </ResizablePanel>
            </ResizablePanelGroup>

            <CanvasContextMenu
                selectedNode={canvas.contextMenuNode}
                selectedNodes={nodeOps.getSelectedSitemapNodes()}
                position={canvas.contextMenuPosition}
                siteUrl={site?.url}
                onClose={canvas.closeContextMenu}
                onAddNode={nodeOps.handleAddNode}
                onAddOrphanNode={nodeOps.handleAddOrphanNode}
                onEditNode={nodeOps.handleEditNode}
                onDeleteNode={nodeOps.handleDeleteNode}
                onAddChildNode={nodeOps.handleAddChild}
                onSyncFromWP={wpOps.handleSyncFromWP}
                onUpdateToWP={wpOps.handleUpdateToWP}
                onGenerateContent={wpOps.handleGenerateContent}
                onPublish={wpOps.handlePublish}
                onUnpublish={wpOps.handleUnpublish}
            />
            <EdgeContextMenu
                position={canvas.edgeContextMenuPosition}
                onClose={canvas.closeEdgeContextMenu}
                onDeleteEdge={canvas.handleDeleteEdge}
            />

            {nodeOps.selectedNode && (
                <NodeEditDialog
                    open={nodeOps.editDialogOpen}
                    onOpenChange={nodeOps.setEditDialogOpen}
                    node={nodeOps.selectedNode}
                    onUpdate={nodeOps.handleUpdateNode}
                    onDelete={() => nodeOps.handleDeleteNode(nodeOps.selectedNode!.id)}
                    onAddChild={() => nodeOps.handleAddChild(nodeOps.selectedNode!.id)}
                />
            )}

            <NodeEditDialog
                open={nodeOps.createDialogOpen}
                onOpenChange={nodeOps.setCreateDialogOpen}
                sitemapId={sitemapId}
                parentId={nodeOps.parentNodeId}
                onCreate={nodeOps.handleCreateNode}
            />

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

            <BulkCreateDialog
                open={bulkCreateDialogOpen}
                onOpenChange={setBulkCreateDialogOpen}
                onSubmit={handleBulkCreate}
            />

            <ImportDialog
                open={importDialogOpen}
                onOpenChange={setImportDialogOpen}
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            <ScanDialog
                open={scanDialogOpen}
                onOpenChange={setScanDialogOpen}
                mode="add"
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            <GenerateDialog
                open={generateDialogOpen}
                onOpenChange={setGenerateDialogOpen}
                mode="add"
                sitemapId={sitemapId}
                onSuccess={() => loadData()}
            />

            <PageGenerateDialog
                open={wpOps.pageGenerateDialogOpen}
                onOpenChange={(open) => {
                    wpOps.setPageGenerateDialogOpen(open);
                    if (!open) {
                        wpOps.setContextMenuSelectedNodes([]);
                    }
                }}
                sitemapId={sitemapId}
                selectedNodes={
                    wpOps.contextMenuSelectedNodes.length > 0
                        ? wpOps.contextMenuSelectedNodes
                        : nodeOps.getSelectedSitemapNodes()
                }
                allNodes={sitemapNodes}
                onSuccess={() => loadData()}
                onTaskStarted={wpOps.setActiveGenerationTask}
                activeTask={wpOps.activeGenerationTask}
            />

            <CommandPalette
                open={commandPaletteOpen}
                onOpenChange={setCommandPaletteOpen}
                hasUnsavedChanges={hasUnsavedChanges}
                hasSelectedNodes={nodes.some((n) => n.selected)}
                onSave={handleSavePositions}
                onAutoLayout={canvas.handleAutoLayout}
                onAddNode={() => nodeOps.handleAddNode()}
                onBulkCreate={() => setBulkCreateDialogOpen(true)}
                onImport={() => setImportDialogOpen(true)}
                onScan={() => setScanDialogOpen(true)}
                onGeneratePages={() => wpOps.setPageGenerateDialogOpen(true)}
                onFocusSearch={() => searchInputRef.current?.focus()}
                onDeleteSelected={nodeOps.handleDeleteSelectedNodes}
                onShowHotkeys={() => setHotkeysDialogOpen(true)}
            />

            <HotkeysDialog
                hotkeys={hotkeys}
                open={hotkeysDialogOpen}
                onOpenChange={setHotkeysDialogOpen}
            />
        </div>
    );
}

function SitemapEditorContent() {
    return (
        <ReactFlowProvider>
            <SitemapEditorFlow />
        </ReactFlowProvider>
    );
}

export default function SitemapEditorPage() {
    return (
        <Suspense
            fallback={
                <div className="h-screen flex items-center justify-center">
                    <div className="text-muted-foreground">Loading...</div>
                </div>
            }
        >
            <SitemapEditorContent />
        </Suspense>
    );
}
