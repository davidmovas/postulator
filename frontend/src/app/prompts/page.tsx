"use client";

import React, { useEffect, useState } from "react";
import { listPrompts, deletePrompt, Prompt } from "@/services/prompt";
import { useErrorHandling } from "@/lib/error-handling";
import { PromptsGrid } from "@/components/prompts/PromptsGrid";
import { CreateEditPromptModal } from "@/components/prompts/CreateEditPromptModal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

export default function PromptsPage() {
  const { withErrorHandling } = useErrorHandling();

  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<Prompt | null>(null);

  const [deleteTarget, setDeleteTarget] = useState<Prompt | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const res = await listPrompts();
      setPrompts(res);
    } finally {
      setLoading(false);
    }
  };

  const refresh = async () => {
    setRefreshing(true);
    try {
      await withErrorHandling(async () => {
        const res = await listPrompts();
        setPrompts(res);
      }, { successMessage: "Prompts updated", showSuccess: true });
    } finally {
      setRefreshing(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleCreate = () => {
    setEditing(null);
    setEditOpen(true);
  };

  const handleEdit = (p: Prompt) => {
    setEditing(p);
    setEditOpen(true);
  };

  const requestDelete = (p: Prompt) => setDeleteTarget(p);

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await withErrorHandling(async () => {
        await deletePrompt(deleteTarget.id);
        await load();
        setDeleteTarget(null);
      }, { successMessage: "Prompt deleted", showSuccess: true });
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="p-4 md:p-6 lg:p-8 space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Prompts</h1>
        <p className="mt-2 text-muted-foreground">Create, edit, and manage AI prompts with dynamic placeholders.</p>
      </div>

      <PromptsGrid
        prompts={prompts}
        loading={loading}
        onRefresh={refresh}
        refreshing={refreshing}
        onCreate={handleCreate}
        onEdit={handleEdit}
        onRequestDelete={requestDelete}
      />

      <CreateEditPromptModal
        open={editOpen}
        onOpenChange={setEditOpen}
        prompt={editing}
        onSaved={async () => { await load(); }}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}
        title="Delete prompt?"
        description={deleteTarget ? (<span>Are you sure you want to delete <b>{deleteTarget.name}</b>? This action cannot be undone.</span>) : undefined}
        confirmText="Delete"
        variant="destructive"
        loading={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
