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
import { RiAddLine, RiDeleteBinLine, RiEdit2Line, RiMoreLine, RiPlayLine, RiRefreshLine, RiSearch2Line, RiToggleLine } from "@remixicon/react";
import SiteStatusBadge from "@/components/ui/site-status-badge";
import { Site } from "@/types/site";
import { SiteForm, SiteFormValues } from "@/components/forms/site-form";
import { useToast } from "@/components/ui/use-toast";

export type SitesTableProps = {
  sites: Site[];
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onRefresh: () => void;
  onMutateSites?: (updater: (prev: Site[]) => Site[]) => void; // optional state lifter
  // Optional backend handlers; if provided, component will call them and then onRefresh()
  onCreate?: (values: SiteFormValues) => Promise<void> | void;
  onUpdate?: (id: number, values: SiteFormValues) => Promise<void> | void;
  onDelete?: (id: number) => Promise<void> | void;
  onToggleActive?: (id: number, active: boolean) => Promise<void> | void;
  onBulkToggle?: (ids: number[], active: boolean) => Promise<void> | void;
  onBulkDelete?: (ids: number[]) => Promise<void> | void;
  onTestConnection?: (id: number) => Promise<void> | void;
};

export default function SitesTable({ sites, page, pageSize, total, onPageChange, onRefresh, onMutateSites, onCreate, onUpdate, onDelete, onToggleActive, onBulkToggle, onBulkDelete, onTestConnection }: SitesTableProps) {
  const [selected, setSelected] = React.useState<Set<number>>(new Set());
  const [query, setQuery] = React.useState<string>("");
  const [addOpen, setAddOpen] = React.useState<boolean>(false);
  const [editOpen, setEditOpen] = React.useState<boolean>(false);
  const [editing, setEditing] = React.useState<Site | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [loading, setLoading] = React.useState<boolean>(false);
  const { toast } = useToast();

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return sites;
    return sites.filter((s) => s.name.toLowerCase().includes(q) || s.url.toLowerCase().includes(q));
  }, [query, sites]);

  const pageCount = Math.max(1, Math.ceil(total / pageSize));

  const apply = (fn: (prev: Site[]) => Site[]) => {
    if (onMutateSites) onMutateSites(fn);
  };

  const toggleSelectAll = (checked: boolean) => {
    if (checked) setSelected(new Set(filtered.map((s) => s.id)));
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

  const handleAdd = async (values: SiteFormValues) => {
    if (onCreate) {
      try {
        setLoading(true);
        setError(null);
        await onCreate(values);
        setAddOpen(false);
        onRefresh();
        toast({ title: "Site created", description: `${values.name} has been created successfully.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to create site";
        setError(msg);
        toast({ title: "Create failed", description: msg, variant: "destructive" });
        return;
      } finally {
        setLoading(false);
      }
    }
    const id = Math.max(0, ...sites.map((s) => s.id)) + 1;
    const newSite: Site = {
      id,
      name: values.name,
      url: values.url,
      username: values.username,
      password: values.password,
      is_active: values.is_active,
      status: values.is_active ? "pending" : "disabled",
      strategy: values.strategy,
    };
    apply((prev) => [newSite, ...prev]);
    setAddOpen(false);
  };

  const handleEdit = async (values: SiteFormValues) => {
    if (!editing) return;
    if (onUpdate) {
      try {
        setLoading(true);
        setError(null);
        await onUpdate(editing.id, values);
        setEditOpen(false);
        setEditing(null);
        onRefresh();
        toast({ title: "Site updated", description: `${values.name} has been updated.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to update site";
        setError(msg);
        toast({ title: "Update failed", description: msg, variant: "destructive" });
        return;
      } finally {
        setLoading(false);
      }
    }
    apply((prev) =>
      prev.map((s) =>
        s.id === editing.id
          ? { ...s, name: values.name, url: values.url, username: values.username, password: values.password, strategy: values.strategy, is_active: values.is_active, status: values.is_active ? s.status === "disabled" ? "pending" : s.status : "disabled" }
          : s
      )
    );
    setEditOpen(false);
    setEditing(null);
  };

  const handleDelete = async (id: number) => {
    if (onDelete) {
      try {
        setLoading(true);
        setError(null);
        await onDelete(id);
        onRefresh();
        toast({ title: "Site deleted", description: `Site has been deleted.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to delete site";
        setError(msg);
        toast({ title: "Delete failed", description: msg, variant: "destructive" });
        return;
      } finally {
        setLoading(false);
      }
    }
    apply((prev) => prev.filter((s) => s.id !== id));
  };

  const handleBulkDelete = async () => {
    const ids = Array.from(selected);
    if (onBulkDelete) {
      try {
        await onBulkDelete(ids);
        setSelected(new Set());
        onRefresh();
        toast({ title: "Deleted", description: `Deleted ${ids.length} site(s).` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to delete selected sites";
        toast({ title: "Bulk delete failed", description: msg, variant: "destructive" });
        return;
      }
    }
    apply((prev) => prev.filter((s) => !ids.includes(s.id)));
    setSelected(new Set());
  };

  const handleBulkToggle = async (active: boolean) => {
    const idsArr = Array.from(selected);
    if (onBulkToggle) {
      try {
        await onBulkToggle(idsArr, active);
        setSelected(new Set());
        onRefresh();
        toast({ title: active ? "Enabled" : "Disabled", description: `${idsArr.length} site(s) ${active ? "enabled" : "disabled"}.` });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to update selected sites";
        toast({ title: "Bulk toggle failed", description: msg, variant: "destructive" });
        return;
      }
    }
    const ids = new Set(selected);
    apply((prev) => prev.map((s) => (ids.has(s.id) ? { ...s, is_active: active, status: active ? (s.status === "disabled" ? "pending" : s.status) : "disabled" } : s)));
    setSelected(new Set());
  };

  const handleToggleActive = async (id: number) => {
    if (onToggleActive) {
      const site = sites.find((s) => s.id === id);
      if (site) {
        try {
          await onToggleActive(id, !site.is_active);
          onRefresh();
          toast({ title: !site.is_active ? "Site enabled" : "Site disabled", description: site.name });
          return;
        } catch (e) {
          const msg = e instanceof Error ? e.message : "Failed to toggle site";
          toast({ title: "Toggle failed", description: msg, variant: "destructive" });
          return;
        }
      }
    }
    apply((prev) =>
      prev.map((s) => (s.id === id ? { ...s, is_active: !s.is_active, status: !s.is_active ? (s.status === "disabled" ? "pending" : s.status) : "disabled" } : s))
    );
  };

  const handleTestConnection = async (id: number) => {
    if (onTestConnection) {
      try {
        await onTestConnection(id);
        onRefresh();
        toast({ title: "Connection test", description: "Test initiated. Check status shortly." });
        return;
      } catch (e) {
        const msg = e instanceof Error ? e.message : "Failed to test connection";
        toast({ title: "Test failed", description: msg, variant: "destructive" });
        return;
      }
    }
    onRefresh();
  };

  const currentPageItems = React.useMemo(() => {
    // Use server-side pagination: `sites` already represents the current page.
    // We only apply client-side filtering to the current page items.
    return filtered;
  }, [filtered]);

  function formatDateTimeEU(iso: string): string {
    try {
      const d = new Date(iso);
      // en-GB => DD/MM/YYYY and 24h time
      return new Intl.DateTimeFormat("en-GB", {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }).format(d);
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
              <span className="ml-1">Add Site</span>
            </Button>
            <Button size="sm" variant="secondary" onClick={() => onRefresh()}>
              <RiRefreshLine size={16} />
              <span className="ml-1">Refresh</span>
            </Button>
            {selected.size > 0 && (
              <div className="flex items-center gap-2">
                <Button size="sm" variant="outline" onClick={() => handleBulkToggle(true)}>
                  <RiToggleLine size={16} /> Enable
                </Button>
                <Button size="sm" variant="outline" onClick={() => handleBulkToggle(false)}>
                  <RiToggleLine size={16} /> Disable
                </Button>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button size="sm" variant="destructive">
                      <RiDeleteBinLine size={16} /> Delete
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Delete {selected.size} selected site(s)?</AlertDialogTitle>
                      <AlertDialogDescription>This action cannot be undone. This will permanently delete the selected sites.</AlertDialogDescription>
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
          <div className="relative w-full sm:w-64">
            <RiSearch2Line className="absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
            <Input
              placeholder="Search sites..."
              className="pl-7"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
        </div>

        {/* Table */}
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[36px]">
                  <Checkbox
                    checked={selected.size > 0 && selected.size === filtered.length}
                    onCheckedChange={(c: boolean) => toggleSelectAll(Boolean(c))}
                    aria-label="Select all"
                  />
                </TableHead>
                <TableHead>Name</TableHead>
                <TableHead>URL</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-center">Active</TableHead>
                <TableHead>Strategy</TableHead>
                <TableHead>Last check</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {currentPageItems.map((s) => (
                <TableRow key={s.id} data-state={selected.has(s.id) ? "selected" : undefined}>
                  <TableCell>
                    <Checkbox checked={selected.has(s.id)} onCheckedChange={(c: boolean) => toggleRow(s.id, Boolean(c))} aria-label={`Select ${s.name}`} />
                  </TableCell>
                  <TableCell className="font-medium">{s.name}</TableCell>
                  <TableCell>
                    <a href={s.url} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                      {s.url}
                    </a>
                  </TableCell>
                  <TableCell><SiteStatusBadge status={s.status} /></TableCell>
                  <TableCell className="align-middle">
                    <span
                      className={`block mx-auto h-2.5 w-2.5 rounded-full ${s.is_active ? "bg-green-500" : "bg-red-500"}`}
                      aria-label={s.is_active ? "Active" : "Inactive"}
                      title={s.is_active ? "Active" : "Inactive"}
                    />
                  </TableCell>
                  <TableCell className="text-xs uppercase text-muted-foreground">{s.strategy.replace("_", " ")}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">{s.last_check_at ? formatDateTimeEU(s.last_check_at) : "—"}</TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" aria-label="Actions">
                          <RiMoreLine size={18} />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => { setEditing(s); setEditOpen(true); }}>
                          <RiEdit2Line size={16} className="mr-2" /> Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleToggleActive(s.id)}>
                          <RiToggleLine size={16} className="mr-2" /> {s.is_active ? "Disable" : "Enable"}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => handleTestConnection(s.id)}>
                          <RiPlayLine size={16} className="mr-2" /> Test connection
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <AlertDialog>
                          <AlertDialogTrigger asChild>
                            <button className="w-full text-left px-2 py-1.5 text-destructive flex items-center">
                              <RiDeleteBinLine size={16} className="mr-2" /> Delete
                            </button>
                          </AlertDialogTrigger>
                          <AlertDialogContent>
                            <AlertDialogHeader>
                              <AlertDialogTitle>Delete site?</AlertDialogTitle>
                              <AlertDialogDescription>This action cannot be undone.</AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                              <AlertDialogCancel>Cancel</AlertDialogCancel>
                              <AlertDialogAction onClick={() => handleDelete(s.id)}>Delete</AlertDialogAction>
                            </AlertDialogFooter>
                          </AlertDialogContent>
                        </AlertDialog>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
              {currentPageItems.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-sm text-muted-foreground py-8">No sites found.</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between">
          <div className="text-xs text-muted-foreground whitespace-nowrap">Page {page} of {pageCount}</div>
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
                  Prev
                </Button>
              </PaginationItem>
              <PaginationItem>
                <Button size="sm" variant="outline" disabled={page >= pageCount} onClick={() => onPageChange(page + 1)}>
                  Next
                </Button>
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      </div>

      {/* Add / Edit Modals */}
      <SiteForm open={addOpen} onOpenChange={setAddOpen} onSubmit={handleAdd} />
      <SiteForm open={editOpen} onOpenChange={(o) => { if (!o) setEditing(null); setEditOpen(o); }} initial={editing ? {
        name: editing.name,
        url: editing.url,
        username: editing.username ?? "",
        password: editing.password ?? "",
        strategy: editing.strategy,
        is_active: editing.is_active,
      } : undefined} onSubmit={handleEdit} />
    </TooltipProvider>
  );
}
