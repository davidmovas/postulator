"use client";

import React, { useEffect, useState } from "react";
import { AIProvider, deleteAIProvider, listAIProviders, setAIProviderStatus } from "@/services/aiProvider";
import { useErrorHandling } from "@/lib/error-handling";
import { AIProvidersGrid } from "@/components/ai/AIProvidersGrid";
import { CreateEditAIProviderModal } from "@/components/ai/CreateEditAIProviderModal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

export default function AIProvidersPage() {
  const { withErrorHandling, showSuccess } = useErrorHandling();

  const [providers, setProviders] = useState<AIProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<AIProvider | null>(null);

  const [deleteTarget, setDeleteTarget] = useState<AIProvider | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const res = await listAIProviders();
      setProviders(res);
    } finally {
      setLoading(false);
    }
  };

  const refresh = async () => {
    setRefreshing(true);
    try {
      await withErrorHandling(async () => {
        const res = await listAIProviders();
        setProviders(res);
      }, { successMessage: "Providers updated", showSuccess: true });
    } finally {
      setRefreshing(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handleCreate = () => { setEditing(null); setEditOpen(true); };
  const handleEdit = (p: AIProvider) => { setEditing(p); setEditOpen(true); };

  const requestDelete = (p: AIProvider) => setDeleteTarget(p);

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await withErrorHandling(async () => {
        await deleteAIProvider(deleteTarget.id);
        await load();
        setDeleteTarget(null);
      }, { successMessage: "Provider deleted", showSuccess: true });
    } finally {
      setDeleting(false);
    }
  };

  const toggleActive = async (p: AIProvider) => {
    await withErrorHandling(async () => {
      await setAIProviderStatus(p.id, !p.isActive);
      await load();
      showSuccess(p.isActive ? "Provider deactivated" : "Provider activated");
    }, { showSuccess: false });
  };

  return (
    <div className="p-4 md:p-6 lg:p-8 space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">AI Providers</h1>
        <p className="mt-2 text-muted-foreground">Manage your AI provider clients: configure provider, model, API key, and activate or deactivate.</p>
      </div>

      <AIProvidersGrid
        providers={providers}
        loading={loading}
        onRefresh={refresh}
        refreshing={refreshing}
        onCreate={handleCreate}
        onEdit={handleEdit}
        onRequestDelete={requestDelete}
        onToggleActive={toggleActive}
      />

      <CreateEditAIProviderModal
        open={editOpen}
        onOpenChange={setEditOpen}
        provider={editing}
        onSaved={async () => { await load(); }}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}
        title="Delete provider?"
        description={deleteTarget ? (<span>Are you sure you want to delete <b>{deleteTarget.name}</b>? This action cannot be undone.</span>) : undefined}
        confirmText="Delete"
        variant="destructive"
        loading={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
