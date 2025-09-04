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

export type SitesTableProps = {
  sites: Site[];
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onRefresh: () => void;
  onMutateSites?: (updater: (prev: Site[]) => Site[]) => void; // optional state lifter
};

export default function SitesTable({ sites, page, pageSize, total, onPageChange, onRefresh, onMutateSites }: SitesTableProps) {
  const [selected, setSelected] = React.useState<Set<number>>(new Set());
  const [query, setQuery] = React.useState<string>("");
  const [addOpen, setAddOpen] = React.useState<boolean>(false);
  const [editOpen, setEditOpen] = React.useState<boolean>(false);
  const [editing, setEditing] = React.useState<Site | null>(null);

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

  const handleAdd = (values: SiteFormValues) => {
    const id = Math.max(0, ...sites.map((s) => s.id)) + 1;
    const newSite: Site = {
      id,
      name: values.name,
      url: values.url,
      username: values.username,
      password: values.password,
      api_key: values.api_key,
      is_active: values.is_active,
      status: values.is_active ? "pending" : "disabled",
    };
    apply((prev) => [newSite, ...prev]);
    setAddOpen(false);
  };

  const handleEdit = (values: SiteFormValues) => {
    if (!editing) return;
    apply((prev) =>
      prev.map((s) =>
        s.id === editing.id
          ? { ...s, name: values.name, url: values.url, username: values.username, password: values.password, api_key: values.api_key, is_active: values.is_active, status: values.is_active ? s.status === "disabled" ? "pending" : s.status : "disabled" }
          : s
      )
    );
    setEditOpen(false);
    setEditing(null);
  };

  const handleDelete = (id: number) => {
    apply((prev) => prev.filter((s) => s.id !== id));
  };

  const handleBulkDelete = () => {
    const ids = Array.from(selected);
    apply((prev) => prev.filter((s) => !ids.includes(s.id)));
    setSelected(new Set());
  };

  const handleBulkToggle = (active: boolean) => {
    const ids = new Set(selected);
    apply((prev) => prev.map((s) => (ids.has(s.id) ? { ...s, is_active: active, status: active ? (s.status === "disabled" ? "pending" : s.status) : "disabled" } : s)));
    setSelected(new Set());
  };

  const handleToggleActive = (id: number) => {
    apply((prev) =>
      prev.map((s) => (s.id === id ? { ...s, is_active: !s.is_active, status: !s.is_active ? (s.status === "disabled" ? "pending" : s.status) : "disabled" } : s))
    );
  };

  const handleTestConnection = (id: number) => {
    // Placeholder: wire to Wails TestSiteConnection in the future
    // TestSiteConnection({ site_id: id })
    onRefresh();
  };

  const currentPageItems = React.useMemo(() => {
    // For demo: paginate filtered array locally
    const start = (page - 1) * pageSize;
    return filtered.slice(start, start + pageSize);
  }, [filtered, page, pageSize]);

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
                <Button size="sm" variant="destructive" onClick={handleBulkDelete}>
                  <RiDeleteBinLine size={16} /> Delete
                </Button>
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
                  <TableCell className="text-muted-foreground text-sm">{s.last_check_at ? new Date(s.last_check_at).toLocaleString() : "—"}</TableCell>
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
                  <TableCell colSpan={6} className="text-center text-sm text-muted-foreground py-8">No sites found.</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between">
          <div className="text-xs text-muted-foreground">Page {page} of {pageCount}</div>
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
        api_key: editing.api_key ?? "",
        is_active: editing.is_active,
      } : undefined} onSubmit={handleEdit} />
    </TooltipProvider>
  );
}
