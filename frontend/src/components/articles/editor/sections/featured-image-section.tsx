"use client";

import { useCallback, useState, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ImageIcon, Upload, X, Link as LinkIcon, Sparkles, Loader2 } from "lucide-react";
import { ArticleFormData } from "@/hooks/use-article-form";
import { cn } from "@/lib/utils";
import {
    Tabs,
    TabsContent,
    TabsList,
    TabsTrigger,
} from "@/components/ui/tabs";
import { mediaService } from "@/services/media";
import { useToast } from "@/components/ui/use-toast";

interface FeaturedImageSectionProps {
    formData: ArticleFormData;
    onUpdate: (updates: Partial<ArticleFormData>) => void;
    disabled?: boolean;
    onAiGenerate?: () => void;
    isAiLoading?: boolean;
    siteId: number;
}

const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml', 'image/bmp'];

export function FeaturedImageSection({
    formData,
    onUpdate,
    disabled = false,
    onAiGenerate,
    isAiLoading = false,
    siteId,
}: FeaturedImageSectionProps) {
    const [urlInput, setUrlInput] = useState(formData.featuredMediaUrl || "");
    const [urlError, setUrlError] = useState<string | null>(null);
    const [isUploading, setIsUploading] = useState(false);
    const [uploadError, setUploadError] = useState<string | null>(null);
    const [dragActive, setDragActive] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);
    const { toast } = useToast();

    const handleUrlSubmit = useCallback(() => {
        if (!urlInput.trim()) {
            setUrlError("Please enter a URL");
            return;
        }

        // Basic URL validation
        try {
            new URL(urlInput);
            onUpdate({
                featuredMediaUrl: urlInput,
                featuredMediaId: undefined, // Clear media ID when using URL
            });
            setUrlError(null);
        } catch {
            setUrlError("Please enter a valid URL");
        }
    }, [urlInput, onUpdate]);

    const handleRemoveImage = useCallback(() => {
        onUpdate({
            featuredMediaUrl: null,
            featuredMediaId: null,
        });
        setUrlInput("");
        setUrlError(null);
        setUploadError(null);
    }, [onUpdate]);

    const validateFile = (file: File): string | null => {
        if (!ALLOWED_TYPES.includes(file.type)) {
            return "Invalid file type. Allowed: JPG, PNG, GIF, WebP, SVG, BMP";
        }
        if (file.size > MAX_FILE_SIZE) {
            return "File is too large. Maximum size is 10MB";
        }
        return null;
    };

    const uploadFile = async (file: File) => {
        const error = validateFile(file);
        if (error) {
            setUploadError(error);
            return;
        }

        setIsUploading(true);
        setUploadError(null);

        try {
            // Convert file to base64
            const base64 = await mediaService.fileToBase64(file);

            // Upload to WordPress
            const result = await mediaService.uploadMedia(
                siteId,
                file.name,
                base64,
                formData.title || "" // Use article title as alt text
            );

            // Update form with uploaded media
            onUpdate({
                featuredMediaId: result.id,
                featuredMediaUrl: result.sourceUrl,
            });

            toast({
                title: "Image uploaded",
                description: "Featured image has been uploaded to WordPress",
            });
        } catch (err) {
            const message = err instanceof Error ? err.message : "Failed to upload image";
            setUploadError(message);
            toast({
                variant: "destructive",
                title: "Upload failed",
                description: message,
            });
        } finally {
            setIsUploading(false);
        }
    };

    const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file) {
            uploadFile(file);
        }
        // Reset input so same file can be selected again
        if (fileInputRef.current) {
            fileInputRef.current.value = "";
        }
    };

    const handleDrag = (e: React.DragEvent) => {
        e.preventDefault();
        e.stopPropagation();
        if (e.type === "dragenter" || e.type === "dragover") {
            setDragActive(true);
        } else if (e.type === "dragleave") {
            setDragActive(false);
        }
    };

    const handleDrop = (e: React.DragEvent) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(false);

        const file = e.dataTransfer.files?.[0];
        if (file) {
            uploadFile(file);
        }
    };

    const handleUploadClick = () => {
        fileInputRef.current?.click();
    };

    const hasImage = !!formData.featuredMediaUrl;

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center justify-between">
                    <div>
                        <CardTitle className="flex items-center gap-2">
                            <ImageIcon className="h-5 w-5" />
                            Featured Image
                        </CardTitle>
                        <CardDescription>
                            Set the main image for your article
                        </CardDescription>
                    </div>
                    {onAiGenerate && (
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={onAiGenerate}
                            disabled={disabled || isAiLoading}
                        >
                            <Sparkles className="h-4 w-4 mr-2" />
                            AI Generate
                        </Button>
                    )}
                </div>
            </CardHeader>
            <CardContent className="space-y-4">
                {hasImage ? (
                    /* Image Preview */
                    <div className="relative group">
                        <div className="aspect-video rounded-lg overflow-hidden border bg-muted">
                            <img
                                src={formData.featuredMediaUrl || undefined}
                                alt="Featured image preview"
                                className="w-full h-full object-cover"
                                onError={(e) => {
                                    (e.target as HTMLImageElement).src = "";
                                    (e.target as HTMLImageElement).classList.add("hidden");
                                }}
                            />
                        </div>
                        <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2 rounded-lg">
                            <Button
                                variant="secondary"
                                size="sm"
                                onClick={handleRemoveImage}
                                disabled={disabled}
                            >
                                <X className="h-4 w-4 mr-1" />
                                Remove
                            </Button>
                        </div>
                        <div className="mt-2 flex items-center gap-2">
                            <p className="text-xs text-muted-foreground truncate flex-1">
                                {formData.featuredMediaUrl}
                            </p>
                            {formData.featuredMediaId && (
                                <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded">
                                    ID: {formData.featuredMediaId}
                                </span>
                            )}
                        </div>
                    </div>
                ) : (
                    /* Image Upload/URL Input */
                    <Tabs defaultValue="upload" className="w-full">
                        <TabsList className="grid w-full grid-cols-2">
                            <TabsTrigger value="upload">
                                <Upload className="h-4 w-4 mr-2" />
                                Upload
                            </TabsTrigger>
                            <TabsTrigger value="url">
                                <LinkIcon className="h-4 w-4 mr-2" />
                                URL
                            </TabsTrigger>
                        </TabsList>

                        <TabsContent value="upload" className="space-y-4 mt-4">
                            <input
                                ref={fileInputRef}
                                type="file"
                                accept={ALLOWED_TYPES.join(',')}
                                onChange={handleFileSelect}
                                className="hidden"
                                disabled={disabled || isUploading}
                            />
                            <div
                                onClick={handleUploadClick}
                                onDragEnter={handleDrag}
                                onDragLeave={handleDrag}
                                onDragOver={handleDrag}
                                onDrop={handleDrop}
                                className={cn(
                                    "aspect-video rounded-lg border-2 border-dashed flex flex-col items-center justify-center gap-2 cursor-pointer transition-colors",
                                    dragActive
                                        ? "border-primary bg-primary/10"
                                        : "border-muted-foreground/25 bg-muted/30 hover:border-primary/50",
                                    (disabled || isUploading) && "opacity-50 cursor-not-allowed"
                                )}
                            >
                                {isUploading ? (
                                    <>
                                        <Loader2 className="h-10 w-10 text-primary animate-spin" />
                                        <p className="text-sm text-muted-foreground">
                                            Uploading to WordPress...
                                        </p>
                                    </>
                                ) : (
                                    <>
                                        <Upload className="h-10 w-10 text-muted-foreground/50" />
                                        <p className="text-sm text-muted-foreground">
                                            Click to upload or drag and drop
                                        </p>
                                        <p className="text-xs text-muted-foreground">
                                            PNG, JPG, GIF, WebP up to 10MB
                                        </p>
                                    </>
                                )}
                            </div>
                            {uploadError && (
                                <p className="text-xs text-destructive">{uploadError}</p>
                            )}
                        </TabsContent>

                        <TabsContent value="url" className="space-y-4 mt-4">
                            <div className="space-y-2">
                                <Label htmlFor="image-url">Image URL</Label>
                                <div className="flex gap-2">
                                    <Input
                                        id="image-url"
                                        value={urlInput}
                                        onChange={(e) => {
                                            setUrlInput(e.target.value);
                                            setUrlError(null);
                                        }}
                                        disabled={disabled}
                                        placeholder="https://example.com/image.jpg"
                                        className={cn(urlError && "border-destructive")}
                                    />
                                    <Button
                                        type="button"
                                        onClick={handleUrlSubmit}
                                        disabled={disabled || !urlInput.trim()}
                                    >
                                        Set
                                    </Button>
                                </div>
                                {urlError && (
                                    <p className="text-xs text-destructive">{urlError}</p>
                                )}
                                <p className="text-xs text-muted-foreground">
                                    Enter the full URL of the image you want to use
                                </p>
                            </div>

                            {/* Image Preview Placeholder */}
                            <div className="aspect-video rounded-lg border-2 border-dashed border-muted-foreground/25 flex flex-col items-center justify-center gap-2 bg-muted/30">
                                <ImageIcon className="h-10 w-10 text-muted-foreground/50" />
                                <p className="text-sm text-muted-foreground">
                                    No image selected
                                </p>
                            </div>
                        </TabsContent>
                    </Tabs>
                )}
            </CardContent>
        </Card>
    );
}
