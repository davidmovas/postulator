"use client";

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { Keyboard } from "lucide-react";
import { HotkeyConfig, formatHotkey, groupHotkeysByCategory } from "@/hooks/use-hotkeys";

interface HotkeysDialogProps {
    hotkeys: HotkeyConfig[];
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
}

export function HotkeysDialog({ hotkeys, open, onOpenChange }: HotkeysDialogProps) {
    const groupedHotkeys = groupHotkeysByCategory(hotkeys);
    const isControlled = open !== undefined;

    const dialogContent = (
        <DialogContent className="max-w-md">
            <DialogHeader>
                <DialogTitle>Keyboard Shortcuts</DialogTitle>
                <DialogDescription>
                    Quick actions for the sitemap editor
                </DialogDescription>
            </DialogHeader>

            <div className="space-y-4 max-h-[60vh] overflow-y-auto">
                {Array.from(groupedHotkeys.entries()).map(([category, categoryHotkeys]) => (
                    <div key={category}>
                        <h4 className="text-sm font-medium text-muted-foreground mb-2">
                            {category}
                        </h4>
                        <div className="space-y-1">
                            {categoryHotkeys.map((hotkey) => (
                                <div
                                    key={`${hotkey.key}-${hotkey.ctrl}-${hotkey.shift}-${hotkey.alt}`}
                                    className="flex items-center justify-between py-1.5 px-2 rounded hover:bg-muted/50"
                                >
                                    <span className="text-sm">{hotkey.description}</span>
                                    <kbd className="px-2 py-1 text-xs font-mono bg-muted rounded border">
                                        {formatHotkey(hotkey)}
                                    </kbd>
                                </div>
                            ))}
                        </div>
                    </div>
                ))}
            </div>
        </DialogContent>
    );

    if (isControlled) {
        return (
            <Dialog open={open} onOpenChange={onOpenChange}>
                {dialogContent}
            </Dialog>
        );
    }

    return (
        <Dialog>
            <TooltipProvider>
                <Tooltip>
                    <TooltipTrigger asChild>
                        <DialogTrigger asChild>
                            <Button variant="outline" size="icon" className="h-8 w-8">
                                <Keyboard className="h-4 w-4" />
                            </Button>
                        </DialogTrigger>
                    </TooltipTrigger>
                    <TooltipContent>
                        <p>Keyboard Shortcuts</p>
                    </TooltipContent>
                </Tooltip>
            </TooltipProvider>
            {dialogContent}
        </Dialog>
    );
}
