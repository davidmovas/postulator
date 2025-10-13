"use client";

import React from "react";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { MoreVertical, Pencil, Trash2 } from "lucide-react";
import { Prompt } from "@/services/prompt";

export interface PromptCardProps {
  prompt: Prompt;
  onEdit: (p: Prompt) => void;
  onRequestDelete: (p: Prompt) => void;
}

export function PromptCard({ prompt, onEdit, onRequestDelete }: PromptCardProps) {
  const formatDate = (dateString: string) => {
    if (!dateString) return "N/A";
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US", {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const sameDates = (prompt.createdAt && prompt.updatedAt) ? (prompt.createdAt === prompt.updatedAt) : false;

  return (
    <Card className="flex flex-col h-full bg-gradient-to-br from-secondary/30 via-muted/30 to-secondary/10 border border-border hover:shadow-lg transition-all">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-3">
          <CardTitle className="truncate" title={prompt.name}>{prompt.name}</CardTitle>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onEdit(prompt)}>
                <Pencil className="h-4 w-4 mr-2" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onRequestDelete(prompt)} className="text-destructive focus:text-destructive">
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 text-sm overflow-hidden">
        <div>
          <div className="text-xs font-medium text-muted-foreground mb-1">System Prompt</div>
          <pre className="whitespace-pre-wrap break-words rounded-md border bg-muted/40 p-2 max-h-40 overflow-auto text-[13px]">{prompt.systemPrompt || "—"}</pre>
        </div>
        <div>
          <div className="text-xs font-medium text-muted-foreground mb-1">User Prompt</div>
          <pre className="whitespace-pre-wrap break-words rounded-md border bg-muted/40 p-2 max-h-40 overflow-auto text-[13px]">{prompt.userPrompt || "—"}</pre>
        </div>
      </CardContent>
      <CardFooter className="pt-0 mt-auto">
        <div className="w-full flex flex-col gap-2">
          <div className="flex flex-wrap gap-1">
            {prompt.placeholders && prompt.placeholders.length > 0 && (
              prompt.placeholders.map((ph) => (
                <Badge key={ph} className="bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300 border-transparent">{`{{${ph}}}`}</Badge>
              ))
            )}
          </div>
          <div className="text-[11px] text-muted-foreground text-right leading-snug mt-1">
            {sameDates ? (
              <div>Created <span className="text-foreground/80">{formatDate(prompt.createdAt)}</span></div>
            ) : (
              <div>
                <span>Created <span className="text-foreground/80">{formatDate(prompt.createdAt)}</span></span>
                <span className="px-1">·</span>
                <span>Updated <span className="text-foreground/80">{formatDate(prompt.updatedAt)}</span></span>
              </div>
            )}
          </div>
        </div>
      </CardFooter>
    </Card>
  );
}

export default PromptCard;
