"use client";

import { useReactFlow } from "@xyflow/react";
import { Button } from "@/components/ui/button";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { ZoomIn, ZoomOut, Maximize2 } from "lucide-react";
import { useEffect, useState, useCallback } from "react";

export function CanvasControls() {
    const { zoomIn, zoomOut, fitView, getZoom } = useReactFlow();
    const [zoom, setZoom] = useState(1);

    const handleZoomIn = useCallback(() => {
        zoomIn({ duration: 200 });
    }, [zoomIn]);

    const handleZoomOut = useCallback(() => {
        zoomOut({ duration: 200 });
    }, [zoomOut]);

    const handleFitView = useCallback(() => {
        fitView({ duration: 300, padding: 0.3 });
    }, [fitView]);

    // Update zoom level periodically
    useEffect(() => {
        const interval = setInterval(() => {
            setZoom(getZoom());
        }, 100);
        return () => clearInterval(interval);
    }, [getZoom]);

    // Keyboard shortcuts
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // Check if user is typing in an input
            if (
                e.target instanceof HTMLInputElement ||
                e.target instanceof HTMLTextAreaElement
            ) {
                return;
            }

            // Ctrl/Cmd + Plus = Zoom In
            if ((e.ctrlKey || e.metaKey) && (e.key === "=" || e.key === "+")) {
                e.preventDefault();
                handleZoomIn();
            }
            // Ctrl/Cmd + Minus = Zoom Out
            if ((e.ctrlKey || e.metaKey) && e.key === "-") {
                e.preventDefault();
                handleZoomOut();
            }
            // Ctrl/Cmd + 0 = Fit View
            if ((e.ctrlKey || e.metaKey) && e.key === "0") {
                e.preventDefault();
                handleFitView();
            }
        };

        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, [handleZoomIn, handleZoomOut, handleFitView]);

    return (
        <TooltipProvider delayDuration={300}>
            <div className="flex items-center gap-1 bg-background/95 backdrop-blur-sm border rounded-lg p-1 shadow-lg">
                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={handleZoomOut}
                        >
                            <ZoomOut className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent side="top">
                        <p>Zoom Out (Ctrl -)</p>
                    </TooltipContent>
                </Tooltip>

                <div className="text-xs text-muted-foreground px-2 min-w-[45px] text-center font-mono">
                    {Math.round(zoom * 100)}%
                </div>

                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={handleZoomIn}
                        >
                            <ZoomIn className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent side="top">
                        <p>Zoom In (Ctrl +)</p>
                    </TooltipContent>
                </Tooltip>

                <div className="h-4 w-px bg-border mx-1" />

                <Tooltip>
                    <TooltipTrigger asChild>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={handleFitView}
                        >
                            <Maximize2 className="h-4 w-4" />
                        </Button>
                    </TooltipTrigger>
                    <TooltipContent side="top">
                        <p>Fit View (Ctrl 0)</p>
                    </TooltipContent>
                </Tooltip>
            </div>
        </TooltipProvider>
    );
}
