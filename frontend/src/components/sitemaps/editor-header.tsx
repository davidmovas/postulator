"use client";

import { Sitemap, GenerationTask } from "@/models/sitemaps";
import { Site } from "@/models/sites";
import { HotkeyConfig } from "@/hooks/use-hotkeys";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
    Tooltip,
    TooltipContent,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { ColorLegendPopover } from "@/components/sitemaps/color-legend-popover";
import { HotkeysDialog } from "@/components/sitemaps/hotkeys-dialog";
import {
    ArrowLeft,
    Plus,
    Save,
    LayoutGrid,
    ListPlus,
    Undo2,
    Redo2,
    Upload,
    ScanLine,
    Sparkles,
    FileText,
    Loader2,
    Wand2,
} from "lucide-react";
import { cn } from "@/lib/utils";

interface EditorHeaderProps {
    site: Site | null;
    sitemap: Sitemap | null;
    hasUnsavedChanges: boolean;
    canUndo: boolean;
    canRedo: boolean;
    activeGenerationTask: GenerationTask | null;
    hotkeys: HotkeyConfig[];
    onNavigateBack: () => void;
    onUndo: () => void;
    onRedo: () => void;
    onAutoLayout: () => void;
    onAddNode: () => void;
    onBulkCreate: () => void;
    onImport: () => void;
    onScan: () => void;
    onGenerateStructure: () => void;
    onGeneratePages: () => void;
    onSave: () => void;
}

export function EditorHeader({
    site,
    sitemap,
    hasUnsavedChanges,
    canUndo,
    canRedo,
    activeGenerationTask,
    hotkeys,
    onNavigateBack,
    onUndo,
    onRedo,
    onAutoLayout,
    onAddNode,
    onBulkCreate,
    onImport,
    onScan,
    onGenerateStructure,
    onGeneratePages,
    onSave,
}: EditorHeaderProps) {
    const isGenerating = activeGenerationTask &&
        (activeGenerationTask.status === "running" || activeGenerationTask.status === "paused");

    return (
        <div className="border-b px-3 py-1.5 flex items-center justify-between bg-background shrink-0">
            <div className="flex items-center gap-2">
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon"
                            className={cn(
                                "h-7 w-7",
                                isGenerating && "opacity-50 cursor-not-allowed"
                            )}
                            onClick={onNavigateBack}
                        >
                            <ArrowLeft className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        {isGenerating
                            ? <p className="text-yellow-500">Wait for generation to complete</p>
                            : <p>Back to Sitemaps</p>
                        }
                    </TooltipContent>
                </Tooltip>
                <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{sitemap?.name}</span>
                    <span className="text-xs text-muted-foreground">({site?.name})</span>
                </div>
            </div>
            <div className="flex items-center gap-1">
                <ColorLegendPopover />
                <HotkeysDialog hotkeys={hotkeys} />

                <Separator orientation="vertical" className="h-5 mx-1" />

                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onUndo}
                            disabled={!canUndo}
                        >
                            <Undo2 className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Undo <span className="text-muted-foreground ml-1">Ctrl+Z</span></p>
                    </TooltipContent>
                </Tooltip>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onRedo}
                            disabled={!canRedo}
                        >
                            <Redo2 className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Redo <span className="text-muted-foreground ml-1">Ctrl+Shift+Z</span></p>
                    </TooltipContent>
                </Tooltip>

                <Separator orientation="vertical" className="h-5 mx-1" />

                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onAutoLayout}
                        >
                            <LayoutGrid className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Auto Layout <span className="text-muted-foreground ml-1">Ctrl+L</span></p>
                    </TooltipContent>
                </Tooltip>

                <Separator orientation="vertical" className="h-5 mx-1" />

                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onAddNode}
                        >
                            <Plus className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Add Node <span className="text-muted-foreground ml-1">Ctrl+N</span></p>
                    </TooltipContent>
                </Tooltip>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onBulkCreate}
                        >
                            <ListPlus className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Bulk Create <span className="text-muted-foreground ml-1">Ctrl+B</span></p>
                    </TooltipContent>
                </Tooltip>

                <Separator orientation="vertical" className="h-5 mx-1" />

                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onImport}
                        >
                            <Upload className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Import from File <span className="text-muted-foreground ml-1">Ctrl+I</span></p>
                    </TooltipContent>
                </Tooltip>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onScan}
                        >
                            <ScanLine className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Scan from WordPress <span className="text-muted-foreground ml-1">Ctrl+K</span></p>
                    </TooltipContent>
                </Tooltip>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="outline"
                            size="icon"
                            className="h-7 w-7"
                            onClick={onGenerateStructure}
                        >
                            <Sparkles className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>AI Generate Structure</p>
                    </TooltipContent>
                </Tooltip>

                <Separator orientation="vertical" className="h-5 mx-1" />

                {/* Content Generation Button - Separated group */}
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant={isGenerating ? "default" : "outline"}
                            size="icon"
                            className={cn(
                                "h-7 w-7",
                                isGenerating && "bg-purple-600 hover:bg-purple-700 text-white"
                            )}
                            onClick={onGeneratePages}
                        >
                            {isGenerating ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                                <Wand2 className="h-4 w-4" />
                            )}
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        {isGenerating ? (
                            <p>Click to view progress</p>
                        ) : (
                            <p>Generate Page Content <span className="text-muted-foreground ml-1">Ctrl+G</span></p>
                        )}
                    </TooltipContent>
                </Tooltip>

                <Separator orientation="vertical" className="h-5 mx-1" />

                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant={hasUnsavedChanges ? "default" : "outline"}
                            size="sm"
                            className={cn("h-7", hasUnsavedChanges && "animate-pulse")}
                            onClick={onSave}
                        >
                            <Save className="h-4 w-4 mr-1" />
                            Save
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Save Layout <span className="text-muted-foreground ml-1">Ctrl+S</span></p>
                    </TooltipContent>
                </Tooltip>
            </div>
        </div>
    );
}
