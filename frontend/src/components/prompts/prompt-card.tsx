"use client";

import { Prompt } from "@/models/prompts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem, DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Edit, MoreVertical, Trash2, MessageSquare, User } from "lucide-react";

interface PromptCardProps {
    prompt: Prompt;
    onEdit: (prompt: Prompt) => void;
    onDelete: (prompt: Prompt) => void;
}

export function PromptCard({ prompt, onEdit, onDelete }: PromptCardProps) {
    return (
        <Card className="relative overflow-hidden hover:shadow-lg transition-shadow duration-200 flex flex-col h-full min-h-[320px]">
            <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                    <CardTitle className="text-lg font-bold line-clamp-2">
                        {prompt.name}
                    </CardTitle>
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

                            <DropdownMenuSeparator />

                            <DropdownMenuItem
                                onClick={() => onDelete(prompt)}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>
            </CardHeader>

            <CardContent className="flex-1 space-y-4">
                {/* System Prompt */}
                <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                        <MessageSquare className="h-4 w-4" />
                        System Prompt
                    </div>
                    <div className="text-sm bg-muted/50 rounded-md p-2 max-h-40 overflow-auto whitespace-pre-wrap break-words">
                        {prompt.systemPrompt}
                    </div>
                </div>

                {/* User Prompt */}
                <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                        <User className="h-4 w-4" />
                        User Prompt
                    </div>
                    <div className="text-sm bg-muted/50 rounded-md p-2 max-h-40 overflow-auto whitespace-pre-wrap">
                        {prompt.userPrompt}
                    </div>
                </div>

                {/* Placeholders */}
                {prompt.placeholders && prompt.placeholders.length > 0 && (
                    <div className="space-y-2">
                        <div className="text-sm font-medium text-muted-foreground">
                            Placeholders
                        </div>
                        <div className="flex flex-wrap gap-1">
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
                    </div>
                )}
            </CardContent>
        </Card>
    );
}