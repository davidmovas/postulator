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
    Loader2,
    Wand2,
    Network,
    GitBranch,
    Link2,
    Unlink,
    CheckCheck,
    XCircle,
    Trash2,
} from "lucide-react";
import { cn } from "@/lib/utils";

export type EditorMode = "map" | "links";

interface EditorHeaderProps {
    site: Site | null;
    sitemap: Sitemap | null;
    hasUnsavedChanges: boolean;
    canUndo: boolean;
    canRedo: boolean;
    activeGenerationTask: GenerationTask | null;
    hotkeys: HotkeyConfig[];
    editorMode: EditorMode;
    onModeChange: (mode: EditorMode) => void;
    onNavigateBack: () => void;
    onUndo: () => void;
    onRedo: () => void;
    onAutoLayout: () => void;
    onSave: () => void;
    // Map mode specific
    onAddNode: () => void;
    onBulkCreate: () => void;
    onImport: () => void;
    onScan: () => void;
    onGenerateStructure: () => void;
    onGeneratePages: () => void;
    // Links mode specific (optional for now, will be added later)
    onSuggestLinks?: () => void;
    onApplyLinks?: () => void;
    onApproveAllLinks?: () => void;
    onRejectAllLinks?: () => void;
    onClearAILinks?: () => void;
    linkStats?: {
        total: number;
        planned: number;
        approved: number;
        applied: number;
    };
}

export function EditorHeader({
    site,
    sitemap,
    hasUnsavedChanges,
    canUndo,
    canRedo,
    activeGenerationTask,
    hotkeys,
    editorMode,
    onModeChange,
    onNavigateBack,
    onUndo,
    onRedo,
    onAutoLayout,
    onSave,
    // Map mode
    onAddNode,
    onBulkCreate,
    onImport,
    onScan,
    onGenerateStructure,
    onGeneratePages,
    // Links mode
    onSuggestLinks,
    onApplyLinks,
    onApproveAllLinks,
    onRejectAllLinks,
    onClearAILinks,
    linkStats,
}: EditorHeaderProps) {
    const isGenerating = activeGenerationTask &&
        (activeGenerationTask.status === "running" || activeGenerationTask.status === "paused");

    const isMapMode = editorMode === "map";
    const isLinksMode = editorMode === "links";

    return (
        <div className="border-b px-3 py-1.5 flex items-center justify-between bg-background shrink-0">
            {/* Left section - Navigation, title, mode switcher */}
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

                <Separator orientation="vertical" className="h-5 mx-1" />

                {/* Mode Switcher */}
                <div className="flex">
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant={isMapMode ? "default" : "outline"}
                                size="sm"
                                aria-label="Map mode"
                                className={cn(
                                    "h-7 px-2.5 rounded-r-none",
                                    isMapMode && "bg-primary text-primary-foreground"
                                )}
                                onClick={() => onModeChange("map")}
                            >
                                <Network className="h-4 w-4" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>
                            <p>Sitemap Mode <span className="text-muted-foreground ml-1">Edit structure</span></p>
                        </TooltipContent>
                    </Tooltip>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <Button
                                variant={isLinksMode ? "default" : "outline"}
                                size="sm"
                                aria-label="Links mode"
                                className={cn(
                                    "h-7 px-2.5 rounded-l-none border-l-0",
                                    isLinksMode && "bg-primary text-primary-foreground"
                                )}
                                onClick={() => onModeChange("links")}
                            >
                                <GitBranch className="h-4 w-4" />
                            </Button>
                        </TooltipTrigger>
                        <TooltipContent>
                            <p>Links Mode <span className="text-muted-foreground ml-1">Manage internal links</span></p>
                        </TooltipContent>
                    </Tooltip>
                </div>

                {/* Link Stats - only in links mode */}
                {isLinksMode && linkStats && linkStats.total > 0 && (
                    <>
                        <Separator orientation="vertical" className="h-5 mx-1" />
                        <div className="flex items-center gap-2 text-xs">
                            <span className="text-muted-foreground">Links:</span>
                            <span className="font-medium">{linkStats.total}</span>
                            {linkStats.planned > 0 && (
                                <span className="text-yellow-500">({linkStats.planned} pending)</span>
                            )}
                            {linkStats.approved > 0 && (
                                <span className="text-green-500">({linkStats.approved} approved)</span>
                            )}
                        </div>
                    </>
                )}
            </div>

            {/* Right section - Actions */}
            <div className="flex items-center gap-1">
                <ColorLegendPopover />
                <HotkeysDialog hotkeys={hotkeys} />

                <Separator orientation="vertical" className="h-5 mx-1" />

                {/* Common: Undo/Redo */}
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

                {/* Common: Auto Layout */}
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

                {/* ===== MAP MODE ACTIONS ===== */}
                {isMapMode && (
                    <>
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

                        {/* Content Generation Button */}
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
                    </>
                )}

                {/* ===== LINKS MODE ACTIONS ===== */}
                {isLinksMode && (
                    <>
                        {/* AI Suggest Links */}
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="outline"
                                    size="icon"
                                    className="h-7 w-7"
                                    onClick={onSuggestLinks}
                                    disabled={!onSuggestLinks}
                                >
                                    <Sparkles className="h-4 w-4" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>AI Suggest Links</p>
                            </TooltipContent>
                        </Tooltip>

                        <Separator orientation="vertical" className="h-5 mx-1" />

                        {/* Approve All */}
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="outline"
                                    size="icon"
                                    className="h-7 w-7"
                                    onClick={onApproveAllLinks}
                                    disabled={!onApproveAllLinks || !linkStats?.planned}
                                >
                                    <CheckCheck className="h-4 w-4 text-green-500" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>Approve All Planned</p>
                            </TooltipContent>
                        </Tooltip>

                        {/* Reject All */}
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="outline"
                                    size="icon"
                                    className="h-7 w-7"
                                    onClick={onRejectAllLinks}
                                    disabled={!onRejectAllLinks || !linkStats?.planned}
                                >
                                    <XCircle className="h-4 w-4 text-red-500" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>Reject All Planned</p>
                            </TooltipContent>
                        </Tooltip>

                        {/* Clear AI Links */}
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="outline"
                                    size="icon"
                                    className="h-7 w-7"
                                    onClick={onClearAILinks}
                                    disabled={!onClearAILinks || !linkStats?.total}
                                >
                                    <Trash2 className="h-4 w-4 text-orange-500" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>Clear All AI Suggestions</p>
                            </TooltipContent>
                        </Tooltip>

                        <Separator orientation="vertical" className="h-5 mx-1" />

                        {/* Apply Links */}
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <Button
                                    variant="outline"
                                    size="icon"
                                    className="h-7 w-7"
                                    onClick={onApplyLinks}
                                    disabled={!onApplyLinks || !linkStats?.approved}
                                >
                                    <Link2 className="h-4 w-4" />
                                </Button>
                            </TooltipTrigger>
                            <TooltipContent>
                                <p>Apply Links to WordPress</p>
                            </TooltipContent>
                        </Tooltip>

                        <Separator orientation="vertical" className="h-5 mx-1" />
                    </>
                )}

                {/* Common: Save */}
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
