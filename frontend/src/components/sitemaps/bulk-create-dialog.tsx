"use client";

import { useState, useEffect, useRef } from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { Keyboard } from "lucide-react";

interface BulkCreateDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSubmit: (paths: string[]) => void;
}

const PLACEHOLDER_EXAMPLE = `/services/web-development Web Development Services
/services/mobile-apps Mobile App Development
/services/consulting
/about-us/team Our Team
/about-us/careers
/blog
/contact Contact Us`;

export function BulkCreateDialog({
    open,
    onOpenChange,
    onSubmit,
}: BulkCreateDialogProps) {
    const [value, setValue] = useState("");
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    useEffect(() => {
        if (open) {
            setValue("");
            // Focus textarea when dialog opens
            setTimeout(() => textareaRef.current?.focus(), 50);
        }
    }, [open]);

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
            e.preventDefault();
            handleSubmit();
        }
    };

    const handleSubmit = () => {
        const lines = value
            .split("\n")
            .map((line) => line.trim())
            .filter((line) => line.length > 0);

        if (lines.length === 0) return;

        // Normalize paths - ensure they start with /
        const normalizedPaths = lines.map((line) => {
            // Check if line starts with / or a word (path segment)
            if (!line.startsWith("/")) {
                return "/" + line;
            }
            return line;
        });

        onSubmit(normalizedPaths);
        onOpenChange(false);
    };

    const lineCount = value.split("\n").filter((l) => l.trim()).length;

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-w-2xl gap-0 p-0 overflow-hidden">
                <DialogHeader className="px-4 py-3 border-b bg-muted/30">
                    <DialogTitle className="text-base">
                        Bulk Create Nodes
                    </DialogTitle>
                </DialogHeader>

                <div className="p-4 space-y-3">
                    <p className="text-sm text-muted-foreground">
                        Enter URL paths, one per line. Optionally add a title after the path separated by a space.
                    </p>

                    <Textarea
                        ref={textareaRef}
                        value={value}
                        onChange={(e) => setValue(e.target.value)}
                        onKeyDown={handleKeyDown}
                        placeholder={PLACEHOLDER_EXAMPLE}
                        rows={12}
                        className="font-mono text-sm resize-none"
                    />

                    <div className="flex items-center justify-between text-xs text-muted-foreground">
                        <span>
                            {lineCount} path{lineCount !== 1 ? "s" : ""} to create
                        </span>
                        <div className="flex items-center gap-1.5">
                            <Keyboard className="h-3 w-3" />
                            <span>Ctrl + Enter to create</span>
                        </div>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
