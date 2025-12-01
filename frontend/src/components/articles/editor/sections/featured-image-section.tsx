"use client";

import { useCallback, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ImageIcon, Upload, X, Link as LinkIcon, Sparkles } from "lucide-react";
import { ArticleFormData } from "@/hooks/use-article-form";
import { cn } from "@/lib/utils";
import {
    Tabs,
    TabsContent,
    TabsList,
    TabsTrigger,
} from "@/components/ui/tabs";

interface FeaturedImageSectionProps {
    formData: ArticleFormData;
    onUpdate: (updates: Partial<ArticleFormData>) => void;
    disabled?: boolean;
    onAiGenerate?: () => void;
    isAiLoading?: boolean;
}

export function FeaturedImageSection({
    formData,
    onUpdate,
    disabled = false,
    onAiGenerate,
    isAiLoading = false,
}: FeaturedImageSectionProps) {
    const [urlInput, setUrlInput] = useState(formData.featuredMediaUrl || "");
    const [urlError, setUrlError] = useState<string | null>(null);

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
            featuredMediaUrl: undefined,
            featuredMediaId: undefined,
        });
        setUrlInput("");
        setUrlError(null);
    }, [onUpdate]);

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
                                src={formData.featuredMediaUrl}
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
                        <p className="mt-2 text-xs text-muted-foreground truncate">
                            {formData.featuredMediaUrl}
                        </p>
                    </div>
                ) : (
                    /* Image Upload/URL Input */
                    <Tabs defaultValue="url" className="w-full">
                        <TabsList className="grid w-full grid-cols-2">
                            <TabsTrigger value="url">
                                <LinkIcon className="h-4 w-4 mr-2" />
                                URL
                            </TabsTrigger>
                            <TabsTrigger value="upload" disabled>
                                <Upload className="h-4 w-4 mr-2" />
                                Upload
                            </TabsTrigger>
                        </TabsList>

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

                        <TabsContent value="upload" className="space-y-4 mt-4">
                            <div className="aspect-video rounded-lg border-2 border-dashed border-muted-foreground/25 flex flex-col items-center justify-center gap-2 bg-muted/30 cursor-pointer hover:border-primary/50 transition-colors">
                                <Upload className="h-10 w-10 text-muted-foreground/50" />
                                <p className="text-sm text-muted-foreground">
                                    Click to upload or drag and drop
                                </p>
                                <p className="text-xs text-muted-foreground">
                                    PNG, JPG, GIF up to 10MB
                                </p>
                            </div>
                        </TabsContent>
                    </Tabs>
                )}
            </CardContent>
        </Card>
    );
}
