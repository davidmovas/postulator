"use client";

import React, { useEffect, useMemo, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { CreateEditTopicModal } from "@/components/topics/CreateEditTopicModal";
import { ImportTopicsDialog } from "@/components/topics/ImportTopicsDialog";
import { TopicRowActions } from "@/components/topics/TopicRowActions";
import { getSite, Site, unassignTopicFromSite } from "@/services/site";
import { Topic, deleteTopic } from "@/services/topic";
import { getTopicsBySite } from "@/services/site";
import { countUnusedTopics } from "@/services/topic";
import { DEFAULT_TOPIC_STRATEGY, TOPIC_STRATEGIES } from "@/constants/topics";
import { useErrorHandling } from "@/lib/error-handling";
import { RefreshCw, ArrowLeft, Pencil, Trash2, Unlink2, Plus, Database, Upload } from "lucide-react";

export default function SiteTopicsPage() {
  const params = useParams();
  const router = useRouter();
  const { withErrorHandling } = useErrorHandling();

  const siteId = Number(params?.siteId);

  const [site, setSite] = useState<Site | null>(null);
  const [topics, setTopics] = useState<Topic[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [strategy, setStrategy] = useState<string>(DEFAULT_TOPIC_STRATEGY);
  const [unusedCount, setUnusedCount] = useState<number>(0);

  // Modals
  const [editOpen, setEditOpen] = useState(false);
  const [editingTopic, setEditingTopic] = useState<Topic | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmAction, setConfirmAction] = useState<null | { type: "delete" | "unassign"; ids: number[] }>(null);
  const [importOpen, setImportOpen] = useState(false);

  const load = async () => {
    setIsLoading(true);
    try {
      const s = await getSite(siteId);
      setSite(s);
      try {
        const t = await getTopicsBySite(siteId);
        setTopics(t);
      } catch {
        setTopics([]);
      }
      try {
        const uc = await countUnusedTopics(siteId);
        setUnusedCount(uc);
      } catch {
        setUnusedCount(0);
      }
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (!Number.isFinite(siteId)) return;
    load();
  }, [siteId]);

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    if (!q) return topics;
    return topics.filter((t) => t.title.toLowerCase().includes(q));
  }, [topics, search]);

  const allSelected = filtered.length > 0 && filtered.every((t) => selectedIds.has(t.id));
  const toggleSelectAll = () => {
    if (allSelected) {
      const newSet = new Set(selectedIds);
      filtered.forEach((t) => newSet.delete(t.id));
      setSelectedIds(newSet);
    } else {
      const newSet = new Set(selectedIds);
      filtered.forEach((t) => newSet.add(t.id));
      setSelectedIds(newSet);
    }
  };

  const toggleSelect = (id: number) => {
    const s = new Set(selectedIds);
    if (s.has(id)) s.delete(id); else s.add(id);
    setSelectedIds(s);
  };

  const openEdit = (t?: Topic) => {
    setEditingTopic(t ?? null);
    setEditOpen(true);
  };

  const requestDelete = (ids: number[]) => {
    setConfirmAction({ type: "delete", ids });
    setConfirmOpen(true);
  };

  const requestUnassign = (ids: number[]) => {
    setConfirmAction({ type: "unassign", ids });
    setConfirmOpen(true);
  };

  const performConfirm = async () => {
    if (!confirmAction) return;
    const { type, ids } = confirmAction;
    setConfirmOpen(false);
    await withErrorHandling(async () => {
      if (type === "delete") {
        for (const id of ids) {
          await deleteTopic(id);
        }
      } else if (type === "unassign") {
        for (const id of ids) {
          await unassignTopicFromSite(siteId, id);
        }
      }
      await load();
      setSelectedIds(new Set());
    }, { successMessage: type === "delete" ? "Topics deleted" : "Topics unassigned", showSuccess: true });
    setConfirmAction(null);
  };

  return (
    <div className="p-4 md:p-6 lg:p-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <button className="text-muted-foreground hover:text-foreground flex items-center gap-2" onClick={() => router.push("/topics") }>
            <ArrowLeft className="h-4 w-4" /> Back to sites
          </button>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight">{site ? site.name : "Site"} — Topics</h1>
          <div className="mt-2 rounded-lg border p-4 bg-card text-card-foreground">
            <div className="grid grid-cols-1 sm:grid-cols-3 divide-y sm:divide-y-0 sm:divide-x">
              <div className="p-3">
                <div className="text-xs text-muted-foreground">Strategy</div>
                <div className="text-lg font-semibold mt-1 capitalize">{strategy}</div>
              </div>
              <div className="p-3">
                <div className="text-xs text-muted-foreground">Total</div>
                <div className="text-lg font-semibold mt-1">{topics.length}</div>
              </div>
              <div className="p-3">
                <div className="text-xs text-muted-foreground">Unused</div>
                <div className="text-lg font-semibold mt-1">{unusedCount}</div>
              </div>
            </div>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => load()}><RefreshCw className="h-4 w-4 mr-2"/>Refresh</Button>
          <Button onClick={() => openEdit()}><Plus className="h-4 w-4 mr-2"/>Add Topic</Button>
          <Button onClick={() => setImportOpen(true)} className="bg-purple-600 hover:bg-purple-700 text-white"><Upload className="h-4 w-4 mr-2"/>Import Topics</Button>
        </div>
      </div>

      <div className="flex items-center justify-between gap-3">
        <Input placeholder="Search topics..." value={search} onChange={(e) => setSearch(e.target.value)} className="max-w-sm" />
        <div className="flex gap-2">
          <Button variant="outline" disabled={selectedIds.size === 0} onClick={() => requestUnassign(Array.from(selectedIds))}><Unlink2 className="h-4 w-4 mr-2"/>Unassign</Button>
          <Button variant="destructive" disabled={selectedIds.size === 0} onClick={() => requestDelete(Array.from(selectedIds))}><Trash2 className="h-4 w-4 mr-2"/>Delete</Button>
        </div>
      </div>

      <div className="border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[40px]"><input type="checkbox" checked={allSelected} onChange={toggleSelectAll} /></TableHead>
              <TableHead>Title</TableHead>
              <TableHead className="w-[160px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={3} className="text-center py-12">
                  <RefreshCw className="h-8 w-8 animate-spin mx-auto mb-2 text-muted-foreground" />
                  <p className="text-muted-foreground">Loading topics...</p>
                </TableCell>
              </TableRow>
            ) : filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={3} className="text-center py-12">
                  <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground/50" />
                  <h3 className="font-semibold mb-1">No topics</h3>
                  <p className="text-sm text-muted-foreground">Create or import topics for this site.</p>
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((t) => (
                <TableRow key={t.id}>
                  <TableCell><input type="checkbox" checked={selectedIds.has(t.id)} onChange={() => toggleSelect(t.id)} /></TableCell>
                  <TableCell className="font-medium">{t.title}</TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <TopicRowActions
                        topic={t}
                        onEdit={(topic) => openEdit(topic)}
                        onUnassign={(id) => requestUnassign([id])}
                        onDelete={(id) => requestDelete([id])}
                      />
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <CreateEditTopicModal
        open={editOpen}
        onOpenChange={setEditOpen}
        topic={editingTopic}
        siteId={siteId}
        onSaved={load}
      />

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={confirmAction?.type === "delete" ? "Delete topics?" : "Unassign topics?"}
        description={confirmAction?.type === "delete" ? "This will permanently remove the selected topics." : "This will unassign the selected topics from the site."}
        confirmText={confirmAction?.type === "delete" ? "Delete" : "Unassign"}
        variant={confirmAction?.type === "delete" ? "destructive" : "default"}
        onConfirm={performConfirm}
      />

      <ImportTopicsDialog
        open={importOpen}
        onOpenChange={setImportOpen}
        siteId={siteId}
        onImported={load}
      />
    </div>
  );
}
