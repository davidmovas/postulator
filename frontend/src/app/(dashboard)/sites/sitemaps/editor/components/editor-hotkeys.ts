import { RefObject } from "react";
import { Node } from "@xyflow/react";
import { HotkeyConfig } from "@/hooks/use-hotkeys";

interface HotkeyHandlers {
    hasUnsavedChanges: boolean;
    handleSavePositions: () => Promise<void>;
    handleAutoLayout: () => void;
    handleAddNode: () => void;
    handleDeleteSelectedNodes: () => void;
    setNodes: (updater: (nodes: Node[]) => Node[]) => void;
    handleUndo: () => void;
    handleRedo: () => void;
    setBulkCreateDialogOpen: (open: boolean) => void;
    setImportDialogOpen: (open: boolean) => void;
    setScanDialogOpen: (open: boolean) => void;
    setPageGenerateDialogOpen: (open: boolean) => void;
    searchInputRef: RefObject<HTMLInputElement | null>;
}

export function createHotkeyConfig(handlers: HotkeyHandlers): HotkeyConfig[] {
    return [
        {
            key: "s",
            ctrl: true,
            description: "Save layout",
            category: "General",
            action: () => {
                if (handlers.hasUnsavedChanges) handlers.handleSavePositions();
            },
        },
        {
            key: "l",
            ctrl: true,
            description: "Auto layout",
            category: "Layout",
            action: handlers.handleAutoLayout,
        },
        {
            key: "n",
            ctrl: true,
            description: "Add new node",
            category: "Nodes",
            action: () => handlers.handleAddNode(),
        },
        {
            key: "Delete",
            description: "Delete selected nodes",
            category: "Nodes",
            action: handlers.handleDeleteSelectedNodes,
        },
        {
            key: "Backspace",
            description: "Delete selected nodes",
            category: "Nodes",
            action: handlers.handleDeleteSelectedNodes,
        },
        {
            key: "Escape",
            description: "Deselect all",
            category: "Selection",
            action: () => {
                handlers.setNodes((nds) => nds.map((n) => ({ ...n, selected: false })));
            },
        },
        {
            key: "f",
            shift: true,
            description: "Focus search",
            category: "Navigation",
            action: () => {
                handlers.searchInputRef.current?.focus();
            },
        },
        {
            key: "b",
            ctrl: true,
            description: "Bulk create nodes",
            category: "Nodes",
            action: () => handlers.setBulkCreateDialogOpen(true),
        },
        {
            key: "i",
            ctrl: true,
            description: "Import from file",
            category: "Nodes",
            action: () => handlers.setImportDialogOpen(true),
        },
        {
            key: "k",
            ctrl: true,
            description: "Scan from WordPress",
            category: "Nodes",
            action: () => handlers.setScanDialogOpen(true),
        },
        {
            key: "z",
            ctrl: true,
            description: "Undo",
            category: "History",
            action: handlers.handleUndo,
        },
        {
            key: "z",
            ctrl: true,
            shift: true,
            description: "Redo",
            category: "History",
            action: handlers.handleRedo,
        },
        {
            key: "g",
            ctrl: true,
            description: "Generate page content",
            category: "Content",
            action: () => handlers.setPageGenerateDialogOpen(true),
        },
        {
            key: "Tab",
            description: "Command palette",
            category: "General",
            action: () => {},
        },
    ];
}
