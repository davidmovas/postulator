"use client";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Clock, User, Globe, FileText, Send } from "lucide-react";
import { ArticleFormData } from "@/hooks/use-article-form";
import { formatSmartDate } from "@/lib/time";

interface Author {
    id: number;
    name: string;
    email?: string;
}

interface PublishingSectionProps {
    formData: ArticleFormData;
    onUpdate: (updates: Partial<ArticleFormData>) => void;
    authors?: Author[] | null;
    disabled?: boolean;
    isPublished?: boolean;
    wpPostUrl?: string;
    wpPostId?: number;
    createdAt?: string;
    updatedAt?: string;
    onPublish?: () => void;
    onUnpublish?: () => void;
    onSync?: () => void;
    isPublishing?: boolean;
}

export function PublishingSection({
    formData,
    onUpdate,
    authors,
    disabled = false,
    isPublished = false,
    wpPostUrl,
    wpPostId,
    createdAt,
    updatedAt,
    onPublish,
    onUnpublish,
    onSync,
    isPublishing = false,
}: PublishingSectionProps) {
    const authorsLoading = authors === null;

    return (
        <Card>
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <Send className="h-5 w-5" />
                    Publishing
                </CardTitle>
                <CardDescription>
                    Control how and when your article is published
                </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
                {/* Status */}
                <div className="space-y-2">
                    <Label>Status</Label>
                    <Select
                        value={formData.status}
                        onValueChange={(value) => onUpdate({ status: value as "draft" | "published" })}
                        disabled={disabled}
                    >
                        <SelectTrigger>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="draft">
                                <div className="flex items-center gap-2">
                                    <FileText className="h-4 w-4" />
                                    Draft
                                </div>
                            </SelectItem>
                            <SelectItem value="published">
                                <div className="flex items-center gap-2">
                                    <Globe className="h-4 w-4" />
                                    Published
                                </div>
                            </SelectItem>
                        </SelectContent>
                    </Select>
                </div>

                {/* Author */}
                {authors !== undefined && (
                    <div className="space-y-2">
                        <Label className="flex items-center gap-2">
                            <User className="h-4 w-4" />
                            Author
                        </Label>
                        <Select
                            value={formData.author?.toString()}
                            onValueChange={(value) => onUpdate({ author: parseInt(value) })}
                            disabled={disabled || authorsLoading}
                        >
                            <SelectTrigger>
                                <SelectValue placeholder="Select author" />
                            </SelectTrigger>
                            <SelectContent>
                                {(authors || []).map(author => (
                                    <SelectItem key={author.id} value={author.id.toString()}>
                                        {author.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                )}

                {/* WordPress Status */}
                {(isPublished || wpPostId) && (
                    <div className="p-4 rounded-lg border bg-muted/30 space-y-3">
                        <div className="flex items-center justify-between">
                            <Label className="flex items-center gap-2">
                                <Globe className="h-4 w-4" />
                                WordPress Status
                            </Label>
                            <Badge variant={isPublished ? "default" : "secondary"}>
                                {isPublished ? "Published" : "Not Published"}
                            </Badge>
                        </div>

                        {wpPostUrl && (
                            <div className="text-sm">
                                <a
                                    href={wpPostUrl}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 underline"
                                >
                                    View on WordPress
                                </a>
                            </div>
                        )}

                        <div className="flex gap-2">
                            {!isPublished && onPublish && (
                                <Button
                                    variant="default"
                                    size="sm"
                                    onClick={onPublish}
                                    disabled={disabled || isPublishing}
                                    className="flex-1"
                                >
                                    <Send className="h-4 w-4 mr-2" />
                                    Publish to WordPress
                                </Button>
                            )}
                            {isPublished && onSync && (
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={onSync}
                                    disabled={disabled || isPublishing}
                                    className="flex-1"
                                >
                                    Sync Changes
                                </Button>
                            )}
                            {isPublished && onUnpublish && (
                                <Button
                                    variant="destructive"
                                    size="sm"
                                    onClick={onUnpublish}
                                    disabled={disabled || isPublishing}
                                >
                                    Unpublish
                                </Button>
                            )}
                        </div>
                    </div>
                )}

                {/* Timestamps */}
                {(createdAt || updatedAt) && (
                    <div className="pt-4 border-t space-y-2 text-sm text-muted-foreground">
                        {createdAt && (
                            <div className="flex items-center gap-2">
                                <Clock className="h-4 w-4" />
                                Created: {formatSmartDate(createdAt)}
                            </div>
                        )}
                        {updatedAt && (
                            <div className="flex items-center gap-2">
                                <Clock className="h-4 w-4" />
                                Updated: {formatSmartDate(updatedAt)}
                            </div>
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}
