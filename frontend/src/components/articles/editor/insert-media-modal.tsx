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
import { ImageIcon } from "lucide-react";

interface InsertMediaModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onInsert: (url: string, alt?: string) => void;
    title?: string;
    placeholder?: string;
}

export function InsertMediaModal({
    open,
    onOpenChange,
    onInsert,
    title = "Insert Image",
    placeholder = "https://example.com/image.jpg",
}: InsertMediaModalProps) {
    const [url, setUrl] = useState("");
    const [alt, setAlt] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [previewError, setPreviewError] = useState(false);

    const handleClose = useCallback(() => {
        setUrl("");
        setAlt("");
        setError(null);
        setPreviewError(false);
        onOpenChange(false);
    }, [onOpenChange]);

    const handleInsert = useCallback(() => {
        if (!url.trim()) {
            setError("Please enter an image URL");
            return;
        }

        try {
            new URL(url);
        } catch {
            setError("Please enter a valid URL");
            return;
        }

        onInsert(url.trim(), alt.trim() || undefined);
        handleClose();
    }, [url, alt, onInsert, handleClose]);

    const isValidUrl = url.trim().length > 0 && !error;

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <ImageIcon className="h-5 w-5" />
                        {title}
                    </DialogTitle>
                    <DialogDescription>
                        Enter the URL of the image you want to insert
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    {/* URL Input */}
                    <div className="space-y-2">
                        <Label htmlFor="media-url">Image URL</Label>
                        <Input
                            id="media-url"
                            value={url}
                            onChange={(e) => {
                                setUrl(e.target.value);
                                setError(null);
                                setPreviewError(false);
                            }}
                            placeholder={placeholder}
                            className={error ? "border-destructive" : ""}
                        />
                        {error && (
                            <p className="text-xs text-destructive">{error}</p>
                        )}
                    </div>

                    {/* Alt Text Input */}
                    <div className="space-y-2">
                        <Label htmlFor="media-alt">Alt Text (optional)</Label>
                        <Input
                            id="media-alt"
                            value={alt}
                            onChange={(e) => setAlt(e.target.value)}
                            placeholder="Describe the image for accessibility"
                        />
                        <p className="text-xs text-muted-foreground">
                            Alternative text helps with SEO and accessibility
                        </p>
                    </div>

                    {/* Preview */}
                    {isValidUrl && (
                        <div className="space-y-2">
                            <Label>Preview</Label>
                            <div className="border rounded-lg overflow-hidden bg-muted/30 p-2">
                                {previewError ? (
                                    <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
                                        Unable to load image preview
                                    </div>
                                ) : (
                                    <img
                                        src={url}
                                        alt={alt || "Preview"}
                                        className="max-h-48 w-auto mx-auto rounded"
                                        onError={() => setPreviewError(true)}
                                    />
                                )}
                            </div>
                        </div>
                    )}
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
