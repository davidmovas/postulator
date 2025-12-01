"use client";

import { useState, useCallback } from "react";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Link } from "lucide-react";

interface InsertLinkModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onInsert: (url: string, text?: string) => void;
}

export function InsertLinkModal({
    open,
    onOpenChange,
    onInsert,
}: InsertLinkModalProps) {
    const [url, setUrl] = useState("");
    const [text, setText] = useState("");
    const [error, setError] = useState<string | null>(null);

    const handleClose = useCallback(() => {
        setUrl("");
        setText("");
        setError(null);
        onOpenChange(false);
    }, [onOpenChange]);

    const handleInsert = useCallback(() => {
        if (!url.trim()) {
            setError("Please enter a URL");
            return;
        }

        // Add protocol if missing
        let finalUrl = url.trim();
        if (!finalUrl.startsWith("http://") && !finalUrl.startsWith("https://") && !finalUrl.startsWith("mailto:")) {
            finalUrl = "https://" + finalUrl;
        }

        try {
            new URL(finalUrl);
        } catch {
            setError("Please enter a valid URL");
            return;
        }

        onInsert(finalUrl, text.trim() || undefined);
        handleClose();
    }, [url, text, onInsert, handleClose]);

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[450px]">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Link className="h-5 w-5" />
                        Insert Link
                    </DialogTitle>
                    <DialogDescription>
                        Enter the URL and optional display text
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    {/* URL Input */}
                    <div className="space-y-2">
                        <Label htmlFor="link-url">URL</Label>
                        <Input
                            id="link-url"
                            value={url}
                            onChange={(e) => {
                                setUrl(e.target.value);
                                setError(null);
                            }}
                            placeholder="https://example.com"
                            className={error ? "border-destructive" : ""}
                        />
                        {error && (
                            <p className="text-xs text-destructive">{error}</p>
                        )}
                    </div>

                    {/* Display Text Input */}
                    <div className="space-y-2">
                        <Label htmlFor="link-text">Display Text (optional)</Label>
                        <Input
                            id="link-text"
                            value={text}
                            onChange={(e) => setText(e.target.value)}
                            placeholder="Click here"
                        />
                        <p className="text-xs text-muted-foreground">
                            Leave empty to use the selected text or URL as display text
                        </p>
                    </div>
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={handleClose}>
                        Cancel
                    </Button>
                    <Button onClick={handleInsert} disabled={!url.trim()}>
                        Insert
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
