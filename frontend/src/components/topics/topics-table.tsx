"use client";
import * as React from "react";
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { Pagination, PaginationContent, PaginationItem } from "@/components/ui/pagination";
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog";
import { TooltipProvider } from "@/components/ui/tooltip";
import { RiAddLine, RiDeleteBinLine, RiEdit2Line, RiMoreLine, RiRefreshLine, RiSearch2Line } from "@remixicon/react";
import type { Topic } from "@/types/topic";
import type { Site } from "@/types/site";
import { useToast } from "@/components/ui/use-toast";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import FileUpload from "@/components/ui/file-upload";

export type UpsertTopicValues = {
  title: string;
  keywords?: string;
  category?: string;
  tags?: string;
};

export type TopicsTableProps = {
  topics: Topic[];
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onRefresh: () => void;
  onMutateTopics?: (updater: (prev: Topic[]) => Topic[]) => void;
  onCreate?: (values: UpsertTopicValues) => Promise<void> | void;
  onUpdate?: (id: number, values: UpsertTopicValues) => Promise<void> | void;
  onDelete?: (id: number) => Promise<void> | void;
  onToggleActive?: (id: number, active: boolean) => Promise<void> | void;
  onBulkToggle?: (ids: number[], active: boolean) => Promise<void> | void;
  onBulkDelete?: (ids: number[]) => Promise<void> | void;
};

export default function TopicsTable({ topics, page, pageSize, total, onPageChange, onRefresh, onMutateTopics, onCreate, onUpdate, onDelete, onToggleActive, onBulkToggle, onBulkDelete }: TopicsTableProps) {
  const [selected, setSelected] = React.useState<Set<number>>(new Set());
  const [query, setQuery] = React.useState<string>("");
  const [addOpen, setAddOpen] = React.useState<boolean>(false);
  const [editOpen, setEditOpen] = React.useState<boolean>(false);
  const [editing, setEditing] = React.useState<Topic | null>(null);
  const { toast } = useToast();

  // Import dialog state
  const [importOpen, setImportOpen] = React.useState<boolean>(false);
  const [sites, setSites] = React.useState<Site[]>([]);
  const [importSiteId, setImportSiteId] = React.useState<number | "">("");
  const [importFile, setImportFile] = React.useState<File | null>(null);
  const [importing, setImporting] = React.useState<boolean>(false);
  const [importError, setImportError] = React.useState<string>("");

  React.useEffect(() => {
    if (!importOpen) return;
    // Load sites on open
    (async () => {
      try {
        const svc = await import("@/services/sites");
        const { items } = await svc.getSites(1, 1000);
        setSites(items);
      } catch (e) {
        // silent error in dialog
      }
    })();
  }, [importOpen]);

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    let items = topics;
    if (q) {
      items = items.filter((t) =>
        (t.title ?? "").toLowerCase().includes(q) ||
        (t.keywords ?? "").toLowerCase().includes(q) ||
        (t.tags ?? "").toLowerCase().includes(q)
      );
    }
    // Active filtering disabled - Topic type doesn't have is_active property
    // if (filterActive !== "all") {
    //   const wantActive = filterActive === "active";
    //   items = items.filter((t) => t.is_active === wantActive);
    // }
    return items;
  }, [topics, query]);

  const pageCount = Math.max(1, Math.ceil(total / pageSize));


  const toggleSelectAll = (checked: boolean) => {
    if (checked) setSelected(new Set(filtered.map((t) => t.id)));
    else setSelected(new Set());
  };

  const toggleRow = (id: number, checked: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  const handleAdd = async (values: UpsertTopicValues) => {
    if (onCreate) {
      try {
        await onCreate(values);
        setAddOpen(false);
        onRefresh();
        toast({ title: "Topic created", description: `${values.title} has been created.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to create topic";
        toast({ title: "Create failed", description: msg, variant: "destructive" });
      }
    }
  };

  const handleEdit = async (values: UpsertTopicValues) => {
    if (!editing) return;
    if (onUpdate) {
      try {
        await onUpdate(editing.id, values);
        setEditOpen(false);
        setEditing(null);
        onRefresh();
        toast({ title: "Topic updated", description: `${values.title} has been updated.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to update topic";
        toast({ title: "Update failed", description: msg, variant: "destructive" });
      }
    }
  };

  const handleDelete = async (id: number) => {
    if (onDelete) {
      try {
        await onDelete(id);
        onRefresh();
        toast({ title: "Topic deleted", description: `Topic has been deleted.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to delete topic";
        toast({ title: "Delete failed", description: msg, variant: "destructive" });
      }
    }
  };

  const handleBulkDelete = async () => {
    const ids = Array.from(selected);
    if (onBulkDelete) {
      try {
        await onBulkDelete(ids);
        setSelected(new Set());
        onRefresh();
        toast({ title: "Deleted", description: `Deleted ${ids.length} topic(s).` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to delete selected topics";
        toast({ title: "Bulk delete failed", description: msg, variant: "destructive" });
        return;
      }
    }
  };

  const handleBulkToggle = async (active: boolean) => {
    const ids = Array.from(selected);
    if (onBulkToggle) {
      try {
        await onBulkToggle(ids, active);
        setSelected(new Set());
        onRefresh();
        toast({ title: active ? "Enabled" : "Disabled", description: `${ids.length} topic(s) ${active ? "enabled" : "disabled"}.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to update selected topics";
        toast({ title: "Bulk toggle failed", description: msg, variant: "destructive" });
        return;
      }
    }
  };

  // handleToggleActive disabled - Topic type doesn't have is_active property
  // const handleToggleActive = async (id: number) => {
  //   if (onToggleActive) {
  //     const topic = topics.find((t) => t.id === id);
  //     if (topic) {
  //       try {
  //         await onToggleActive(id, !topic.is_active);
  //         onRefresh();
  //         toast({ title: !topic.is_active ? "Topic enabled" : "Topic disabled", description: topic.title });
  //         return;
  //       } catch (e) {
  //         const msg = e instanceof Error ? e.message : "Failed to toggle topic";
  //         toast({ title: "Toggle failed", description: msg, variant: "destructive" });
  //         return;
  //       }
  //     }
  //   }
  // };

  function formatDateTimeEU(iso?: string): string {
    if (!iso) return "—";
    try {
      const d = new Date(iso);
      return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(d);
    } catch {
      return iso;
    }
  }

  return (
    <TooltipProvider>
      <div className="flex flex-col gap-3">
        {/* Toolbar */}
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={() => setAddOpen(true)}>
              <RiAddLine size={16} />
              <span className="ml-1">Add Topic</span>
            </Button>
            <Button size="sm" variant="outline" onClick={() => setImportOpen(true)}>
              <RiAddLine size={16} />
              <span className="ml-1">Import</span>
            </Button>
            <Button size="sm" variant="secondary" onClick={() => onRefresh()}>
              <RiRefreshLine size={16} />
              <span className="ml-1">Refresh</span>
            </Button>
            {selected.size > 0 && (
              <div className="flex items-center gap-2">
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button size="sm" variant="destructive">
                      <RiDeleteBinLine size={16} /> Delete
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Delete {selected.size} selected topic(s)?</AlertDialogTitle>
                      <AlertDialogDescription>This action cannot be undone. This will permanently delete the selected topics.</AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction onClick={handleBulkDelete}>Delete</AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            )}
          </div>
          <div className="flex items-center gap-2">
            <div className="relative w-full sm:w-64">
              <RiSearch2Line className="absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
              <Input placeholder="Search title, keywords, tags..." className="pl-7" value={query} onChange={(e) => setQuery(e.target.value)} />
            </div>
          </div>
        </div>

        {/* Table */}
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[36px]">
                  <Checkbox checked={selected.size > 0 && selected.size === filtered.length} onCheckedChange={(c: boolean) => toggleSelectAll(Boolean(c))} aria-label="Select all" />
                </TableHead>
                <TableHead className="w-[40%]">Title</TableHead>
                <TableHead>Keywords</TableHead>
                <TableHead>Category</TableHead>
                <TableHead>Tags</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((t) => (
                <TableRow key={t.id} data-state={selected.has(t.id) ? "selected" : undefined}>
                  <TableCell>
                    <Checkbox checked={selected.has(t.id)} onCheckedChange={(c: boolean) => toggleRow(t.id, Boolean(c))} aria-label={`Select ${t.title}`} />
                  </TableCell>
                  <TableCell className="font-medium w-[40%]">{t.title}</TableCell>
                  <TableCell className="text-muted-foreground text-sm truncate max-w-[240px]" title={t.keywords}>{t.keywords || "—"}</TableCell>
                  <TableCell className="text-sm">{t.category || "—"}</TableCell>
                  <TableCell className="text-sm truncate max-w-[240px]" title={t.tags}>{t.tags || "—"}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">{formatDateTimeEU(t.created_at)}</TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" aria-label="Actions">
                          <RiMoreLine size={18} />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => { setEditing(t); setEditOpen(true); }}>
                          <RiEdit2Line size={16} className="mr-2" /> Edit
                        </DropdownMenuItem>
                        {/* Toggle disabled - handleToggleActive function commented out
                        <DropdownMenuItem onClick={() => handleToggleActive(t.id)}>
                          <RiToggleLine size={16} className="mr-2" /> {t.is_active ? "Disable" : "Enable"}
                        </DropdownMenuItem>
                        */}
                        <DropdownMenuSeparator />
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <button className="w-full text-left px-2 py-1.5 text-destructive flex items-center">
                              <RiDeleteBinLine size={16} className="mr-2" /> Delete
                            </button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Delete topic?</AlertDialogTitle>
                              <AlertDialogDescription>This action cannot be undone.</AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction onClick={() => handleDelete(t.id)}>Delete</AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
              {filtered.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-sm text-muted-foreground py-8">No topics found.</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between">
          <div className="text-xs text-muted-foreground whitespace-nowrap">Page {page} of {Math.max(1, pageCount)}</div>
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>Prev</Button>
              </PaginationItem>
              <PaginationItem>
                <Button size="sm" variant="outline" disabled={page >= pageCount} onClick={() => onPageChange(page + 1)}>Next</Button>
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      </div>

      {/* Add / Edit Dialogs */}
      <TopicForm open={addOpen} onOpenChange={setAddOpen} onSubmit={handleAdd} />
      <TopicForm open={editOpen} onOpenChange={(o) => { if (!o) setEditing(null); setEditOpen(o); }} initial={editing ? {
        title: editing.title,
        keywords: editing.keywords,
        category: editing.category,
        tags: editing.tags,
      } : undefined} onSubmit={handleEdit} />

      {/* Import Dialog */}
      <Dialog open={importOpen} onOpenChange={(o) => { setImportError(""); if (!o) { setImportFile(null); setImportSiteId(""); } setImportOpen(o); }}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>Import topics from file</DialogTitle>
          </DialogHeader>
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setImportError("");
              if (!importSiteId) { setImportError("Please select a site"); return; }
              if (!importFile) { setImportError("Please choose a file"); return; }
              setImporting(true);
              try {
                const ext = (importFile.name.split('.').pop() || '').toLowerCase();
                const textTypes = ["txt","csv","json"]; 
                let content = "";
                if (textTypes.includes(ext)) {
                  content = await importFile.text();
                } else {
                  const buf = await importFile.arrayBuffer();
                  // convert to base64
                  let binary = "";
                  const bytes = new Uint8Array(buf);
                  const chunk = 0x8000;
                  for (let i=0; i<bytes.length; i+=chunk) {
                    binary += String.fromCharCode.apply(null, Array.from(bytes.subarray(i, i+chunk)) as any);
                  }
                  content = btoa(binary);
                }
                const format = ext || "txt";
                const svc = await import("@/services/topics");
                await svc.importTopics(Number(importSiteId), content, format, false);
                toast({ title: "Import completed", description: `${importFile.name}` });
                setImportOpen(false);
                setImportFile(null);
                setImportSiteId("");
                onRefresh();
              } catch (e) {
                const msg = e instanceof Error ? e.message : "Failed to import";
                setImportError(msg);
              } finally {
                setImporting(false);
              }
            }}
            className="space-y-3"
          >
            <div className="grid gap-1.5">
              <Label htmlFor="site">Site</Label>
              <select id="site" className="h-9 border rounded-md px-2 bg-background" value={importSiteId} onChange={(e) => setImportSiteId(e.target.value ? Number(e.target.value) : "")} required>
                <option value="">Select site…</option>
                {sites.map((s) => (
                  <option key={s.id} value={s.id}>{s.name}</option>
                ))}
              </select>
            </div>
            <div className="grid gap-1.5">
              <Label>File</Label>
              <FileUpload accept="txt,csv,json,xls,xlsx" onFileSelected={setImportFile} />
            </div>
            {importError && <div className="text-sm text-destructive">{importError}</div>}
            <div className="flex justify-end gap-2 pt-2">
              <Button type="button" variant="outline" onClick={() => setImportOpen(false)}>Cancel</Button>
              <Button type="submit" disabled={importing || !importSiteId || !importFile}>{importing ? "Import..." : "Import"}</Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </TooltipProvider>
  );
}

// Inline form component using shadcn Dialog & basic inputs
function TopicForm({ open, onOpenChange, initial, onSubmit }: { open: boolean; onOpenChange: (o: boolean) => void; initial?: UpsertTopicValues; onSubmit: (values: UpsertTopicValues) => void | Promise<void>; }) {
  const [values, setValues] = React.useState<UpsertTopicValues>(initial ?? { title: "", keywords: "", category: "", tags: "" });
  React.useEffect(() => {
    setValues(initial ?? { title: "", keywords: "", category: "", tags: "" });
  }, [initial]);

  const [submitting, setSubmitting] = React.useState(false);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>{initial ? "Edit Topic" : "Add Topic"}</DialogTitle>
        </DialogHeader>
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            setSubmitting(true);
            try {
              await onSubmit(values);
            } finally {
              setSubmitting(false);
            }
          }}
          className="space-y-3"
        >
          <div className="grid gap-1.5">
            <Label htmlFor="title">Title</Label>
            <Input id="title" value={values.title} onChange={(e) => setValues((v) => ({ ...v, title: e.target.value }))} required />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="keywords">Keywords</Label>
            <Input id="keywords" value={values.keywords ?? ""} onChange={(e) => setValues((v) => ({ ...v, keywords: e.target.value }))} />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="category">Category</Label>
            <Input id="category" value={values.category ?? ""} onChange={(e) => setValues((v) => ({ ...v, category: e.target.value }))} />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="tags">Tags</Label>
            <Input id="tags" value={values.tags ?? ""} onChange={(e) => setValues((v) => ({ ...v, tags: e.target.value }))} />
          </div>
          {/* Active switch disabled - is_active property removed from UpsertTopicValues type
          <div className="flex items-center justify-between py-2">
            <Label htmlFor="is_active">Active</Label>
            <Switch id="is_active" checked={values.is_active} onCheckedChange={(c) => setValues((v) => ({ ...v, is_active: Boolean(c) }))} />
          </div>
          */}
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit" disabled={submitting}>{initial ? "Save" : "Create"}</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
