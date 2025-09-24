"use client";
import * as React from "react";
import type { Prompt } from "@/types/prompt";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

const PLACEHOLDERS = [
  "{{title}}",
  "{{keywords}}",
  "{{category}}",
  "{{tags}}",
  "{{site.name}}",
  "{{site.url}}",
  "{{site.strategy}}",
];

function useInsertAtCursor() {
  const ref = React.useRef<HTMLTextAreaElement | null>(null);
  const setRef = (el: HTMLTextAreaElement | null) => (ref.current = el);
  function insertToken(token: string) {
    const el = ref.current;
    if (!el) return;
    const start = el.selectionStart ?? el.value.length;
    const end = el.selectionEnd ?? el.value.length;
    const before = el.value.slice(0, start);
    const after = el.value.slice(end);
    const next = `${before}${token}${after}`;
    const pos = start + token.length;
    el.value = next;
    // trigger input event for React state sync if needed by onChange
    el.dispatchEvent(new Event("input", { bubbles: true }));
    // restore caret
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(pos, pos);
    });
  }
  return { setRef, insertToken };
}

function PromptFormDialog({
  open,
  onOpenChange,
  onSubmit,
  initial,
  mode = "create",
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSubmit: (values: { name: string; is_active: boolean; is_default: boolean; system: string; user: string }) => Promise<void> | void;
  initial?: Partial<{ name: string; is_active: boolean; is_default: boolean; system: string; user: string }>;
  mode?: "create" | "edit";
}) {
  const [name, setName] = React.useState("");
  const [isActive, setIsActive] = React.useState(true);
  const [isDefault, setIsDefault] = React.useState(false);
  const [system, setSystem] = React.useState("");
  const [user, setUser] = React.useState("");
  const [saving, setSaving] = React.useState(false);
  const [search, setSearch] = React.useState("");
  const sysIns = useInsertAtCursor();
  const usrIns = useInsertAtCursor();

  React.useEffect(() => {
    if (open) {
      setName(initial?.name ?? "");
      setIsActive(initial?.is_active ?? true);
      setIsDefault(initial?.is_default ?? false);
      setSystem(initial?.system ?? "");
      setUser(initial?.user ?? "");
      setSearch("");
      setSaving(false);
    }
  }, [open, initial]);

  const filtered = PLACEHOLDERS.filter(p => p.toLowerCase().includes(search.toLowerCase()));
  const valid = name.trim().length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-150">
        <DialogHeader>
          <DialogTitle>{mode === "create" ? "Create Prompt" : "Edit Prompt"}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-3">
          <div className="grid gap-1">
            <label className="text-sm font-medium">Name</label>
            <Input value={name} onChange={(e)=>setName(e.target.value)} placeholder="e.g. Default SEO Prompt" />
          </div>
          <div className="flex items-center gap-6">
            <label className="flex items-center gap-2 text-sm"><Switch checked={isActive} onCheckedChange={setIsActive} /> Active</label>
            <label className="flex items-center gap-2 text-sm"><Switch checked={isDefault} onCheckedChange={setIsDefault} /> Make default</label>
          </div>
          <div className="grid gap-1">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">System</label>
              <Popover>
                <PopoverTrigger asChild>
                  <Button size="sm" variant="outline">Insert placeholder</Button>
                </PopoverTrigger>
                <PopoverContent>
                  <Input placeholder="Search placeholders" value={search} onChange={(e)=>setSearch(e.target.value)} className="h-8 mb-2" />
                  <div className="max-h-48 overflow-auto space-y-1">
                    {filtered.map(ph => (
                      <button key={ph} className="w-full text-left text-sm px-2 py-1 rounded hover:bg-accent" onClick={(e)=>{e.preventDefault(); sysIns.insertToken(ph);}}>{ph}</button>
                    ))}
                    {filtered.length === 0 && <div className="text-xs text-muted-foreground">No placeholders</div>}
                  </div>
                </PopoverContent>
              </Popover>
            </div>{/*
            <Textarea ref={sysIns.setRef} value={system} onChange={(e)=>setSystem(e.target.value)} rows={6} placeholder="System prompt text with {{placeholders}}" />
          */}</div>
          <div className="grid gap-1">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">User</label>
              <Popover>
                <PopoverTrigger asChild>
                  <Button size="sm" variant="outline">Insert placeholder</Button>
                </PopoverTrigger>
                <PopoverContent>
                  <Input placeholder="Search placeholders" value={search} onChange={(e)=>setSearch(e.target.value)} className="h-8 mb-2" />
                  <div className="max-h-48 overflow-auto space-y-1">
                    {filtered.map(ph => (
                      <button key={ph} className="w-full text-left text-sm px-2 py-1 rounded hover:bg-accent" onClick={(e)=>{e.preventDefault(); usrIns.insertToken(ph);}}>{ph}</button>
                    ))}
                    {filtered.length === 0 && <div className="text-xs text-muted-foreground">No placeholders</div>}
                  </div>
                </PopoverContent>
              </Popover>
            </div>{/*
            <Textarea ref={usrIns.setRef} value={user} onChange={(e)=>setUser(e.target.value)} rows={12} placeholder="User prompt text with {{placeholders}}" />
          */}</div>
        </div>
        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={saving}>Cancel</Button>
          <Button disabled={!valid || saving} onClick={async ()=>{
            try {
              setSaving(true);
              await onSubmit({ name, is_active: isActive, is_default: isDefault, system, user });
              onOpenChange(false);
            } catch (e) {
              toast.error(e instanceof Error ? e.message : String(e));
            } finally {
              setSaving(false);
            }
          }}>{mode === "create" ? "Create" : "Save"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AssignSitesDialog({
  open,
  onOpenChange,
  prompt,
}: { open: boolean; onOpenChange: (v: boolean)=>void; prompt: Prompt | null }) {
  const [sites, setSites] = React.useState<{ id: number; name: string; url: string }[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [assigning, setAssigning] = React.useState<number | null>(null);
  const [isActive, setIsActive] = React.useState(true);
  const [q, setQ] = React.useState("");
  const [assigned, setAssigned] = React.useState<Set<number>>(new Set());

  // Debounced search loader: only fetch when user typed >= 2 chars
  React.useEffect(() => {
    if (!open) return;
    const qq = q.trim();
    if (qq.length < 2) { setSites([]); return; }
    const handle = setTimeout(async () => {
      try {
        setLoading(true);
        const sitesSvc = await import("@/services/sites");
        const { items } = await sitesSvc.getSites(1, 1000);
        const mapped = items.map(s => ({ id: s.id, name: s.name, url: s.url }));
        const filtered = mapped.filter(s => s.name.toLowerCase().includes(qq.toLowerCase()));
        setSites(filtered);
      } catch (e) {
        toast.error(e instanceof Error ? e.message : String(e));
      } finally {
        setLoading(false);
      }
    }, 250);
    return () => clearTimeout(handle);
  }, [open, q]);

  // Load already assigned sites for this prompt on open
  React.useEffect(() => {
    let cancelled = false;
    async function loadAssigned() {
      if (!open || !prompt) { setAssigned(new Set()); return; }
      try {
        /*const sp = await import("@/services/sitePrompts");
        const res = await sp.getPromptSites(prompt.id, 1, 10000);
        const list: any[] = (res as any)?.site_prompts ?? [];
        const ids = new Set<number>(list.map((x: any) => x.site_id));
        if (!cancelled) setAssigned(ids);*/
      } catch {
        if (!cancelled) setAssigned(new Set());
      }
    }
    void loadAssigned();
    return () => { cancelled = true; };
  }, [open, prompt]);

  async function assignToSite(siteId: number) {
    if (!prompt) return;
    try {
      setAssigning(siteId);
      const sp = await import("@/services/sitePrompts");
      await sp.deleteSitePromptBySite(siteId).catch(()=>{});
      await sp.createSitePrompt(siteId, prompt.id);
      setAssigned((prev) => {
        const next = new Set(prev);
        next.add(siteId);
        return next;
      });
      toast.success(`Assigned to ${siteId}`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setAssigning(null);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-150">
        <DialogHeader>
          <DialogTitle>Assign “{prompt?.name ?? ""}” to sites</DialogTitle>
        </DialogHeader>
        <div className="flex items-center justify-between mb-2">
          <div className="text-sm text-muted-foreground">One prompt per site. Assigning will overwrite current prompt.</div>
          <label className="flex items-center gap-2 text-sm"><Switch checked={isActive} onCheckedChange={setIsActive} /> Active</label>
        </div>
        <div className="flex items-center gap-2 mb-2">
          <Input value={q} onChange={(e)=>setQ(e.target.value)} placeholder="Search sites by name… (min 2 chars)" className="h-8" />
        </div>
        {q.trim().length < 2 ? (
          <div className="text-sm text-muted-foreground">Type at least 2 characters to search.</div>
        ) : loading ? (
          <div className="text-sm text-muted-foreground">Searching…</div>
        ) : (
          <div className="max-h-[50vh] overflow-auto divide-y">
            {sites.map((s) => (
              <div key={s.id} className="flex items-center justify-between py-2">
                <div>
                  <div className="text-sm font-medium">{s.name}</div>
                  <div className="text-xs text-muted-foreground">{s.url}</div>
                </div>
                {(() => { const isAssigned = assigned.has(s.id); return (
                  <Button size="sm" onClick={() => void assignToSite(s.id)} disabled={assigning === s.id || isAssigned} variant={isAssigned ? "secondary" : "default"}>
                    {assigning === s.id ? "Assigning…" : isAssigned ? "Assigned" : "Assign"}
                  </Button>
                ); })()}
              </div>
            ))}
            {sites.length === 0 && <div className="text-sm text-muted-foreground p-2">No matching sites</div>}
          </div>
        )
        }
        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ConfirmDeletePromptDialog({ open, onOpenChange, prompt, onDeleted }: { open: boolean; onOpenChange: (v: boolean)=>void; prompt: Prompt | null; onDeleted: ()=>void }) {
  const [busy, setBusy] = React.useState(false);
  const [usedCount, setUsedCount] = React.useState<number | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    let cancelled = false;
    async function loadUsage() {
      if (!open || !prompt) { setUsedCount(null); setError(null); return; }
      setError(null);
      setUsedCount(null);
      try {
        const svc = await import("@/services/prompts");
        const count = await svc.countSitesUsingPrompt(prompt.id);
        if (!cancelled) setUsedCount(count);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    }
    void loadUsage();
    return () => { cancelled = true; };
  }, [open, prompt]);

  async function handleDelete() {
    if (!prompt) return;
    try {
      setBusy(true);
      const svc = await import("@/services/prompts");
      await svc.deletePrompt(prompt.id);
      toast.success("Prompt deleted");
      onOpenChange(false);
      onDeleted();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const blocked = (usedCount ?? 0) > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Delete prompt</DialogTitle>
          <DialogDescription>
            This action cannot be undone. This will permanently delete the prompt and remove its data from our servers.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <div>
            <span className="text-muted-foreground">Name:</span> <span className="font-medium">{prompt?.name}</span>
          </div>
          <div className="text-muted-foreground">ID #{prompt?.id}</div>
          {usedCount === null && !error && (
            <div className="text-xs text-muted-foreground">Checking usage…</div>
          )}
          {typeof usedCount === "number" && (
            <div className={cn("text-xs", blocked ? "text-destructive" : "text-muted-foreground")}>Used by {usedCount} site{usedCount === 1 ? "" : "s"}{blocked ? ": deletion is blocked." : ""}</div>
          )}
          {error && <div className="text-xs text-destructive">{error}</div>}
          {blocked && (
            <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">
              You cannot delete a prompt that is assigned to sites. Unassign it from all sites first.
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)} disabled={busy}>Cancel</Button>
          <Button variant="destructive" onClick={() => void handleDelete()} disabled={busy || blocked}>Delete</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default function PromptsPanel() {
  const page = 1;
  const pageSize = 1000; // no pagination needed
  const [items, setItems] = React.useState<Prompt[]>([]);
  const [total, setTotal] = React.useState<number>(0);
  const [loading, setLoading] = React.useState<boolean>(false);
  const [error, setError] = React.useState<string | undefined>();
  const [q, setQ] = React.useState("");
  const [onlyDefaultOnTop, setOnlyDefaultOnTop] = React.useState<boolean>(true);
  const [activeFilter, setActiveFilter] = React.useState<"all" | "active" | "inactive">("all");

  const [openCreate, setOpenCreate] = React.useState(false);
  const [assignPrompt, setAssignPrompt] = React.useState<Prompt | null>(null);
  const [editPrompt, setEditPrompt] = React.useState<Prompt | null>(null);
  const [deletePrompt, setDeletePrompt] = React.useState<Prompt | null>(null);

  async function load() {
    try {
      setLoading(true);
      setError(undefined);
      const svc = await import("@/services/prompts");
      const { items, total } = await svc.getPrompts(page, pageSize);
      setItems(items);
      setTotal(total);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load prompts");
      setItems([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  React.useEffect(() => { void load(); /* eslint-disable-next-line */ }, []);

  const filtered = React.useMemo(() => {
    const qq = q.trim().toLowerCase();
    const arr = items.filter((p) => {
      const hay = `${p.name} ${p.system} ${p.user}`.toLowerCase();
      const matchQ = qq ? hay.includes(qq) : true;
      const matchActive = activeFilter === "all" ? true : activeFilter === "active" ? p.is_active : !p.is_active;
      return matchQ && matchActive;
    });
    // Sort: Default → Active → Updated desc
    arr.sort((a, b) => {
      if (onlyDefaultOnTop) {
        if (a.is_default && !b.is_default) return -1;
        if (!a.is_default && b.is_default) return 1;
      }
      if (a.is_active !== b.is_active) return a.is_active ? -1 : 1;
      const au = a.updated_at ?? a.created_at ?? "";
      const bu = b.updated_at ?? b.created_at ?? "";
      return au > bu ? -1 : au < bu ? 1 : 0;
    });
    return arr;
  }, [items, q, activeFilter, onlyDefaultOnTop]);

  const excerpt = (s: string, max = 160) => (s.length > max ? s.slice(0, max) + "…" : s);

  return (
    <div className="p-4 md:p-6 lg:p-8">
      <div className="mb-3">
        <h1 className="text-xl font-semibold">Prompts</h1>
        <p className="text-sm text-muted-foreground">Manage AI prompts. Default is always on top. Search by name or content.</p>
      </div>
      {error && <div className="mb-3 text-sm text-destructive">{error}</div>}

      {/* Toolbar */}
      <div className="mb-4 flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-2">
          <Input placeholder="Search name or content…" value={q} onChange={(e) => setQ(e.target.value)} className="w-72" />
          <select
            aria-label="Active filter"
            className="h-9 rounded-md border bg-background px-2 text-sm"
            value={activeFilter}
            onChange={(e) => setActiveFilter(e.target.value as 'all' | 'active' | 'inactive')}
          >
            <option value="all">All</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
          <label className="inline-flex items-center gap-2 text-sm">
            <input type="checkbox" checked={onlyDefaultOnTop} onChange={(e) => setOnlyDefaultOnTop(e.target.checked)} />
            Only default/on top
          </label>
        </div>
        <div className="flex items-center gap-2">
          <Button onClick={() => setOpenCreate(true)}>Create</Button>
          <Button variant="secondary" onClick={() => void load()}>Refresh</Button>
        </div>
      </div>

      {/* Grid */}
      <div className="grid gap-3 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {filtered.map((p) => (
          <Card key={p.id} className={cn("overflow-hidden flex flex-col h-full", p.is_default ? "ring-1 ring-primary/30" : undefined)}>
            <CardHeader>
              <div className="flex items-start justify-between gap-2">
                <CardTitle className="text-base font-semibold leading-tight">
                  <span>{p.name}</span>
                </CardTitle>
                <div className="flex flex-wrap gap-1.5 justify-end">
                  {p.is_default && <Badge>Default</Badge>}
                  <Badge variant={p.is_active ? "secondary" : "outline"}>{p.is_active ? "Active" : "Inactive"}</Badge>
                </div>
              </div>
              <div className="text-[11px] text-muted-foreground mt-1">
                <span>Created {p.created_at?.slice(0, 19).replace("T", " ") ?? ""}</span>
                {p.updated_at && <span> • Updated {p.updated_at.slice(0, 19).replace("T", " ")}</span>}
              </div>
            </CardHeader>
            <CardContent className="flex-1">
              <div className="mb-2">
                <div className="text-xs text-muted-foreground mb-0.5">System</div>
                <div className="text-sm line-clamp-3 whitespace-pre-wrap">{excerpt(p.system)}</div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground mb-0.5">User</div>
                <div className="text-sm line-clamp-3 whitespace-pre-wrap">{excerpt(p.user)}</div>
              </div>
            </CardContent>
            <CardFooter className="flex items-center justify-between gap-2">
              <div className="text-xs text-muted-foreground">ID #{p.id}</div>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button size="sm" variant="outline">Actions</Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem disabled={p.is_default} onClick={async () => {
                    try {
                      const svc = await import("@/services/prompts");
                      await svc.makeDefaultPrompt(p.id);
                      toast.success("Made default");
                      void load();
                    } catch (e) {
                      toast.error(e instanceof Error ? e.message : String(e));
                    }
                  }}>Make default</DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setAssignPrompt(p)}>Assign to Sites</DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setEditPrompt(p)}>Edit</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={() => setDeletePrompt(p)}>Delete</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </CardFooter>
          </Card>
        ))}
      </div>

      <div className="mt-4 text-sm text-muted-foreground">Total: {total}</div>

      {loading && <div className="mt-3 text-sm text-muted-foreground">Loading…</div>}
      {!loading && filtered.length === 0 && <div className="mt-3 text-sm text-muted-foreground">No prompts found.</div>}

      {/* Dialog mounts */}
      <PromptFormDialog
        open={openCreate}
        onOpenChange={setOpenCreate}
        mode="create"
        onSubmit={async (v) => {
          const svc = await import("@/services/prompts");
          await svc.createPrompt(v);
          toast.success("Prompt created");
          void load();
        }}
      />
      <PromptFormDialog
        open={!!editPrompt}
        onOpenChange={(v)=>{ if(!v) setEditPrompt(null); }}
        mode="edit"
        initial={editPrompt ?? undefined}
        onSubmit={async (v) => {
          if (!editPrompt) return;
          const svc = await import("@/services/prompts");
          await svc.updatePrompt({ id: editPrompt.id, name: v.name, system: v.system, user: v.user, is_default: v.is_default, is_active: v.is_active });
          toast.success("Prompt updated");
          void load();
        }}
      />
      <AssignSitesDialog open={!!assignPrompt} onOpenChange={(v)=>{ if(!v) setAssignPrompt(null); }} prompt={assignPrompt} />
      <ConfirmDeletePromptDialog
        open={!!deletePrompt}
        onOpenChange={(v)=>{ if(!v) setDeletePrompt(null); }}
        prompt={deletePrompt}
        onDeleted={() => void load()}
      />
    </div>
  );
}
