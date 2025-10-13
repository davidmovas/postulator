"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { Database, Plus, RefreshCw } from "lucide-react";
import { AIProvider } from "@/services/aiProvider";
import { AIProviderCard } from "@/components/ai/AIProviderCard";

export interface AIProvidersGridProps {
  providers: AIProvider[];
  loading?: boolean;
  onRefresh: () => void | Promise<void>;
  refreshing?: boolean;
  onCreate: () => void;
  onEdit: (p: AIProvider) => void;
  onRequestDelete: (p: AIProvider) => void;
  onToggleActive: (p: AIProvider) => void | Promise<void>;
}

export function AIProvidersGrid({ providers, loading = false, onRefresh, refreshing = false, onCreate, onEdit, onRequestDelete, onToggleActive }: AIProvidersGridProps) {
  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">{providers.length} provider{providers.length === 1 ? "" : "s"}</div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => onRefresh()} disabled={!!refreshing}>
            <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <Button onClick={onCreate}>
            <Plus className="h-4 w-4 mr-2" />
            New Provider
          </Button>
        </div>
      </div>

      {/* Empty state */}
      {!loading && providers.length === 0 && (
        <div className="text-center py-12 border rounded-lg">
          <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground/50" />
          <h3 className="font-semibold mb-1">No AI providers yet</h3>
          <p className="text-sm text-muted-foreground mb-4">Create your first AI provider to get started.</p>
          <Button onClick={onCreate}>
            <Plus className="h-4 w-4 mr-2" />
            Create Provider
          </Button>
        </div>
      )}

      {/* Grid */}
      {providers.length > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {providers.map((p) => (
            <AIProviderCard key={p.id} provider={p} onEdit={onEdit} onRequestDelete={onRequestDelete} onToggleActive={onToggleActive} />
          ))}
        </div>
      )}
    </div>
  );
}

export default AIProvidersGrid;
