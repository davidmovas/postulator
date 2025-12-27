"use client";

import { Prompt, PROMPT_CATEGORIES, isV2Prompt } from "@/models/prompts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Edit, MoreVertical, Trash2, FileText, Settings2, Lock } from "lucide-react";

interface PromptCardProps {
    prompt: Prompt;
    onEdit: (prompt: Prompt) => void;
    onDelete: (prompt: Prompt) => void;
}

export function PromptCard({ prompt, onEdit, onDelete }: PromptCardProps) {
    const categoryLabel = PROMPT_CATEGORIES[prompt.category] || prompt.category;
    const isV2 = isV2Prompt(prompt);

    // Get enabled context field keys for v2 prompts
    const enabledFields = isV2 && prompt.contextConfig
        ? Object.entries(prompt.contextConfig)
            .filter(([_, value]) => value.enabled)
            .map(([key]) => key)
        : [];

    return (
        <Card className="relative overflow-hidden hover:shadow-lg transition-shadow duration-200 flex flex-col h-full min-h-[280px]">
            <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                    <div className="space-y-1.5 flex-1 min-w-0 pr-2">
                        <div className="flex items-center gap-2">
                            {prompt.isBuiltin && (
                                <Lock className="h-4 w-4 text-muted-foreground shrink-0" />
                            )}
                            <CardTitle className="text-lg font-bold line-clamp-2">
                                {prompt.name}
                            </CardTitle>
                        </div>
                        <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-xs">
                                {categoryLabel}
                            </Badge>
                            {prompt.isBuiltin && (
                                <Badge variant="secondary" className="text-xs">
                                    Builtin
                                </Badge>
                            )}
                        </div>
                    </div>
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                <MoreVertical className="h-4 w-4" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => onEdit(prompt)}>
                                <Edit className="h-4 w-4 mr-2" />
                                Edit
                            </DropdownMenuItem>

                            {!prompt.isBuiltin && (
                                <>
                                    <DropdownMenuSeparator />
                                    <DropdownMenuItem
                                        onClick={() => onDelete(prompt)}
                                        className="text-destructive focus:text-destructive"
                                    >
                                        <Trash2 className="h-4 w-4 mr-2" />
                                        Delete
                                    </DropdownMenuItem>
                                </>
                            )}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </CardHeader>

            <CardContent className="flex-1 flex flex-col gap-3">
                {/* Instructions (v2) or System Prompt (v1) */}
                <div className="flex-1 min-h-0">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground mb-2">
                        <FileText className="h-4 w-4" />
                        {isV2 ? "Instructions" : "System Prompt"}
                    </div>
                    <ScrollArea className="h-[120px] rounded-md border bg-muted/30">
                        <div className="p-3 text-sm whitespace-pre-wrap break-words font-mono">
                            {isV2 ? prompt.instructions : prompt.systemPrompt}
                        </div>
                    </ScrollArea>
                </div>

                {/* Context Fields (v2) or Placeholders (v1) */}
                <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                        <Settings2 className="h-4 w-4" />
                        {isV2 ? "Context Fields" : "Placeholders"}
                    </div>
                    {isV2 ? (
                        enabledFields.length > 0 ? (
                            <div className="flex flex-wrap gap-1.5">
                                {enabledFields.map((field) => (
                                    <Badge
                                        key={field}
                                        variant="secondary"
                                        className="text-xs font-normal"
                                    >
                                        {field}
                                    </Badge>
                                ))}
                            </div>
                        ) : (
                            <p className="text-xs text-muted-foreground italic">
                                No context fields enabled
                            </p>
                        )
                    ) : (
                        prompt.placeholders && prompt.placeholders.length > 0 ? (
                            <div className="flex flex-wrap gap-1.5">
                                {prompt.placeholders.map((placeholder, index) => (
                                    <Badge
                                        key={index}
                                        variant="secondary"
                                        className="text-xs font-mono"
                                    >
                                        {placeholder}
                                    </Badge>
                                ))}
                            </div>
                        ) : (
                            <p className="text-xs text-muted-foreground italic">
                                No placeholders
                            </p>
                        )
                    )}
                </div>
            </CardContent>
        </Card>
    );
}
