"use client";

import React from "react";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { CheckCircle2, MoreVertical, Pencil, Power, Trash2 } from "lucide-react";
import { AIProvider } from "@/services/aiProvider";

export interface AIProviderCardProps {
  provider: AIProvider;
  onEdit: (p: AIProvider) => void;
  onRequestDelete: (p: AIProvider) => void;
  onToggleActive: (p: AIProvider) => void | Promise<void>;
}

export function AIProviderCard({ provider, onEdit, onRequestDelete, onToggleActive }: AIProviderCardProps) {
  return (
    <Card className="flex flex-col h-full bg-gradient-to-br from-secondary/20 via-muted/30 to-secondary/5 border border-border hover:shadow-lg transition-all">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2 min-w-0">
            {provider.isActive ? (
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-emerald-500" title="Active" />
            ) : (
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-muted-foreground/40" title="Inactive" />
            )}
            <CardTitle className="truncate" title={provider.name}>{provider.name}</CardTitle>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onEdit(provider)}>
                <Pencil className="h-4 w-4 mr-2" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onToggleActive(provider)}>
                <Power className="h-4 w-4 mr-2" />
                {provider.isActive ? "Deactivate" : "Activate"}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onRequestDelete(provider)} className="text-destructive focus:text-destructive">
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 text-sm overflow-hidden">
        <div>
          <div className="text-xs font-medium text-muted-foreground mb-1">Model</div>
          <div className="rounded-md border bg-muted/40 p-2 text-[13px] break-words">
            {provider.model || "—"}
          </div>
        </div>
      </CardContent>
      <CardFooter className="pt-0 mt-auto">
        <div className="w-full flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <div className="flex items-center gap-2">
            <Badge className={provider.isActive ? "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300 border-transparent" : "bg-muted text-foreground/70 border-transparent"}>
              {provider.isActive ? "Active" : "Inactive"}
            </Badge>
          </div>
          <div className="hidden sm:flex items-center gap-1">
            <CheckCircle2 className="h-3.5 w-3.5 text-muted-foreground/70" />
            <span>{new Date(provider.updatedAt || provider.createdAt).toLocaleDateString("en-US", { month: "short", day: "2-digit" })}</span>
          </div>
        </div>
      </CardFooter>
    </Card>
  );
}

export default AIProviderCard;
