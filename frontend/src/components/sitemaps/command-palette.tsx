"use client";

import {
    CommandDialog,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
    CommandSeparator,
    CommandShortcut,
} from "@/components/ui/command";
import {
    Save,
    Plus,
    LayoutGrid,
    Search,
    ListPlus,
    Keyboard,
    Trash2,
    Upload,
} from "lucide-react";

interface CommandPaletteProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    hasUnsavedChanges: boolean;
    hasSelectedNodes: boolean;
    onSave: () => void;
    onAutoLayout: () => void;
    onAddNode: () => void;
    onBulkCreate: () => void;
    onImport: () => void;
    onFocusSearch: () => void;
    onDeleteSelected: () => void;
    onShowHotkeys: () => void;
}

export function CommandPalette({
    open,
    onOpenChange,
    hasUnsavedChanges,
    hasSelectedNodes,
    onSave,
    onAutoLayout,
    onAddNode,
    onBulkCreate,
    onImport,
    onFocusSearch,
    onDeleteSelected,
    onShowHotkeys,
}: CommandPaletteProps) {
    const runCommand = (command: () => void) => {
        onOpenChange(false);
        command();
    };

    return (
        <CommandDialog open={open} onOpenChange={onOpenChange}>
            <CommandInput placeholder="Type a command or search..." />
            <CommandList>
                <CommandEmpty>No results found.</CommandEmpty>

                <CommandGroup heading="Actions">
                    <CommandItem onSelect={() => runCommand(onAddNode)}>
                        <Plus className="mr-2 h-4 w-4" />
                        <span>Add Node</span>
                        <CommandShortcut>Ctrl+N</CommandShortcut>
                    </CommandItem>
                    <CommandItem onSelect={() => runCommand(onBulkCreate)}>
                        <ListPlus className="mr-2 h-4 w-4" />
                        <span>Bulk Create Nodes</span>
                        <CommandShortcut>Ctrl+B</CommandShortcut>
                    </CommandItem>
                    <CommandItem onSelect={() => runCommand(onImport)}>
                        <Upload className="mr-2 h-4 w-4" />
                        <span>Import from File</span>
                        <CommandShortcut>Ctrl+I</CommandShortcut>
                    </CommandItem>
                    {hasSelectedNodes && (
                        <CommandItem onSelect={() => runCommand(onDeleteSelected)}>
                            <Trash2 className="mr-2 h-4 w-4" />
                            <span>Delete Selected</span>
                            <CommandShortcut>Del</CommandShortcut>
                        </CommandItem>
                    )}
                </CommandGroup>

                <CommandSeparator />

                <CommandGroup heading="Layout">
                    <CommandItem onSelect={() => runCommand(onAutoLayout)}>
                        <LayoutGrid className="mr-2 h-4 w-4" />
                        <span>Auto Layout</span>
                        <CommandShortcut>Ctrl+L</CommandShortcut>
                    </CommandItem>
                    <CommandItem
                        onSelect={() => runCommand(onSave)}
                        disabled={!hasUnsavedChanges}
                    >
                        <Save className="mr-2 h-4 w-4" />
                        <span>Save Layout</span>
                        <CommandShortcut>Ctrl+S</CommandShortcut>
                    </CommandItem>
                </CommandGroup>

                <CommandSeparator />

                <CommandGroup heading="Navigation">
                    <CommandItem onSelect={() => runCommand(onFocusSearch)}>
                        <Search className="mr-2 h-4 w-4" />
                        <span>Focus Search</span>
                        <CommandShortcut>Shift+F</CommandShortcut>
                    </CommandItem>
                    <CommandItem onSelect={() => runCommand(onShowHotkeys)}>
                        <Keyboard className="mr-2 h-4 w-4" />
                        <span>Show Keyboard Shortcuts</span>
                        <CommandShortcut>?</CommandShortcut>
                    </CommandItem>
                </CommandGroup>
            </CommandList>
        </CommandDialog>
    );
}
