"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { Database, Plus, RefreshCw } from "lucide-react";
import { Prompt } from "@/services/prompt";
import { PromptCard } from "@/components/prompts/PromptCard";

export interface PromptsGridProps {
  prompts: Prompt[];
  loading?: boolean;
  onRefresh: () => void | Promise<void>;
  refreshing?: boolean;
  onCreate: () => void;
  onEdit: (p: Prompt) => void;
  onRequestDelete: (p: Prompt) => void;
}

export function PromptsGrid({ prompts, loading = false, onRefresh, refreshing = false, onCreate, onEdit, onRequestDelete }: PromptsGridProps) {
  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">{prompts.length} prompt{prompts.length === 1 ? "" : "s"}</div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => onRefresh()} disabled={!!refreshing}>
            <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button onClick={onCreate}>
            <Plus className="h-4 w-4 mr-2" />
            New Prompt
          </Button>
        </div>
      </div>

      {/* Empty state */}
      {!loading && prompts.length === 0 && (
        <div className="text-center py-12 border rounded-lg">
          <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground/50" />
          <h3 className="font-semibold mb-1">No prompts yet</h3>
          <p className="text-sm text-muted-foreground mb-4">Create your first prompt to get started.</p>
          <Button onClick={onCreate}>
            <Plus className="h-4 w-4 mr-2" />
            Create Prompt
          </Button>
        </div>
      )}

      {/* Grid */}
      {prompts.length > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {prompts.map((p) => (
            <PromptCard key={p.id} prompt={p} onEdit={onEdit} onRequestDelete={onRequestDelete} />
          ))}
        </div>
      )}
    </div>
  );
}

export default PromptsGrid;
