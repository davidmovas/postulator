"use client";
import * as React from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/components/ui/use-toast";
import { Checkbox } from "@/components/ui/checkbox";
import type { Topic } from "@/types/topic";
import type { Site } from "@/types/site";
import type { SiteTopicLink } from "@/services/siteTopics";

export default function AssignPanel() {
  // Data state
  const [sites, setSites] = React.useState<Site[]>([]);
  const [topics, setTopics] = React.useState<Topic[]>([]);
  const [siteTopics, setSiteTopics] = React.useState<SiteTopicLink[]>([]);
  const [currentSiteId, setCurrentSiteId] = React.useState<number | null>(null);

  // UI state
  const [sitesQuery, setSitesQuery] = React.useState("");
  const [assignedQuery, setAssignedQuery] = React.useState("");
  const [topicsQuery, setTopicsQuery] = React.useState("");
  const [loading, setLoading] = React.useState(false);
  const [priorityDrafts, setPriorityDrafts] = React.useState<Record<number, number>>({});
  const [assignDefaults, setAssignDefaults] = React.useState<{ priority: number; active: boolean }>({ priority: 1, active: true });
  const { toast } = useToast();

  // Selection state
  const [selectedAssigned, setSelectedAssigned] = React.useState<Set<number>>(new Set()); // link IDs
  const [selectedFree, setSelectedFree] = React.useState<Set<number>>(new Set()); // topic IDs

  // Derived maps
  const topicsById = React.useMemo(() => new Map(topics.map(t => [t.id, t])), [topics]);
  const currentSite = React.useMemo(() => sites.find(s => s.id === currentSiteId) ?? null, [sites, currentSiteId]);

  // Stats per site for sites table
  const [siteStats, setSiteStats] = React.useState<Record<number, { total_topics: number; active_topics: number; unique_topics_left: number; round_robin_position: number }>>({});

  // Load base data (sites + topics)
  const loadBase = React.useCallback(async () => {
    try {
      setLoading(true);
      const sitesSvc = await import("@/services/sites");
      const topicsSvc = await import("@/services/topics");
      const [{ items: sItems }, { items: tItems }] = await Promise.all([
        sitesSvc.getSites(1, 100),
        topicsSvc.getTopics(1, 1000),
      ]);
      setSites(sItems);
      setTopics(tItems);
    } catch (e) {
      toast({ title: "Load failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
    } finally {
      setLoading(false);
    }
  }, [toast]);

  React.useEffect(() => {
    void loadBase();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Load stats when sites change
  React.useEffect(() => {
    async function loadAllStats() {
      try {
        const statsSvc = await import("@/services/stats");
        const entries = await Promise.all(sites.map(async (s) => {
          try {
            const st = await statsSvc.getTopicStats(s.id);
            return [s.id, {
              total_topics: st.total_topics ?? 0,
              active_topics: st.active_topics ?? 0,
              unique_topics_left: st.unique_topics_left ?? 0,
              round_robin_position: st.round_robin_position ?? 0,
            }] as const;
          } catch {
            return [s.id, { total_topics: 0, active_topics: 0, unique_topics_left: 0, round_robin_position: 0 }] as const;
          }
        }));
        const map: Record<number, { total_topics: number; active_topics: number; unique_topics_left: number; round_robin_position: number }> = {};
        for (const [id, val] of entries) map[id] = val;
        setSiteStats(map);
      } catch (e) {
        console.warn("Failed to load site stats", e);
      }
    }
    if (sites.length) void loadAllStats();
  }, [sites]);

  // Load site-topic links when current site changes
  const loadLinks = React.useCallback(async (siteId: number) => {
    try {
      const svc = await import("@/services/siteTopics");
      const { items } = await svc.getSiteTopics(siteId, 1, 10000);
      setSiteTopics(items);
      setPriorityDrafts(Object.fromEntries(items.map(it => [it.id, it.priority])));
      setSelectedAssigned(new Set());
      setSelectedFree(new Set());
    } catch (e) {
      toast({ title: "Failed to load assignments", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
    }
  }, [toast]);

  React.useEffect(() => {
    if (currentSiteId) void loadLinks(currentSiteId);
    else setSiteTopics([]);
  }, [currentSiteId, loadLinks]);

  const assignedTopicIds = React.useMemo(() => new Set(siteTopics.map(st => st.topic_id)), [siteTopics]);

  // Filters
  const filteredAssigned = React.useMemo(() => {
    const q = assignedQuery.trim().toLowerCase();
    if (!q) return siteTopics;
    return siteTopics.filter(link => {
      const t = topicsById.get(link.topic_id);
      return (
        (t?.title ?? link.topic_title ?? "").toLowerCase().includes(q) ||
        (t?.keywords ?? "").toLowerCase().includes(q) ||
        (t?.tags ?? "").toLowerCase().includes(q) ||
        (t?.category ?? "").toLowerCase().includes(q)
      );
    });
  }, [siteTopics, topicsById, assignedQuery]);

  const freeTopics = React.useMemo(() => {
    const q = topicsQuery.trim().toLowerCase();
    // Show only topics that are free (not assigned to this site) AND active
    const items = topics.filter(t => t.is_active && !assignedTopicIds.has(t.id));
    return q ? items.filter(t => (t.title ?? "").toLowerCase().includes(q) || (t.keywords ?? "").toLowerCase().includes(q) || (t.tags ?? "").toLowerCase().includes(q) || (t.category ?? "").toLowerCase().includes(q)) : items;
  }, [topics, assignedTopicIds, topicsQuery]);

  // Actions
  async function assignTopic(topicId: number) {
    if (!currentSite) return;
    try {
      const svc = await import("@/services/siteTopics");
      const link = await svc.createSiteTopic({ site_id: currentSite.id, topic_id: topicId, priority: assignDefaults.priority, is_active: assignDefaults.active });
      setSiteTopics(prev => [...prev, link]);
      toast({ title: "Assigned", description: topicsById.get(topicId)?.title ?? String(topicId) });
    } catch (e) {
      toast({ title: "Assign failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
    }
  }

  async function assignTopicsBulk(ids: number[]) {
    if (!currentSite || ids.length === 0) return;
    try {
      const svc = await import("@/services/siteTopics");
      const created: SiteTopicLink[] = [];
      for (const tid of ids) {
        try {
          const link = await svc.createSiteTopic({ site_id: currentSite.id, topic_id: tid, priority: assignDefaults.priority, is_active: assignDefaults.active });
          created.push(link);
        } catch {}
      }
      if (created.length > 0) setSiteTopics(prev => [...prev, ...created]);
      setSelectedFree(new Set());
    } finally {
      // no toasts requested for bulk assign success/failure beyond actions already handled
    }
  }

  async function unassignLink(linkId: number) {
    try {
      const svc = await import("@/services/siteTopics");
      await svc.deleteSiteTopic(linkId);
      setSiteTopics(prev => prev.filter(l => l.id !== linkId));
      // No toasts for unassign per requirement
    } catch (e) {
      // Suppress toasts for unassign errors as requested
      console.warn("Unassign failed", e);
    }
  }

  async function unassignLinksBulk(ids: number[]) {
    try {
      const svc = await import("@/services/siteTopics");
      await Promise.all(ids.map(id => svc.deleteSiteTopic(id).catch(() => {})));
      setSiteTopics(prev => prev.filter(l => !ids.includes(l.id)));
      setSelectedAssigned(new Set());
    } catch (e) {
      // Suppress toasts
      console.warn("Bulk unassign issues", e);
    }
  }

  async function toggleLinkActive(linkId: number) {
    const link = siteTopics.find(l => l.id === linkId);
    if (!link) return;
    try {
      const svc = await import("@/services/siteTopics");
      await svc.setSiteTopicActive(linkId, !link.is_active);
      setSiteTopics(prev => prev.map(l => l.id === linkId ? { ...l, is_active: !l.is_active } : l));
    } catch (e) {
      toast({ title: "Toggle failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
    }
  }

  async function toggleLinksBulk(ids: number[], active: boolean) {
    try {
      const svc = await import("@/services/siteTopics");
      await Promise.all(ids.map(id => {
        const link = siteTopics.find(l => l.id === id);
        if (!link || link.is_active === active) return Promise.resolve();
        return svc.setSiteTopicActive(id, active).catch(() => {});
      }));
      setSiteTopics(prev => prev.map(l => ids.includes(l.id) ? { ...l, is_active: active } : l));
      setSelectedAssigned(new Set());
    } catch (e) {
      toast({ title: "Bulk toggle failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
    }
  }

  async function savePriority(linkId: number) {
    const link = siteTopics.find(l => l.id === linkId);
    if (!link) return;
    const next = priorityDrafts[linkId] ?? link.priority;
    try {
      const svc = await import("@/services/siteTopics");
      await svc.updateSiteTopic({ id: link.id, site_id: link.site_id, topic_id: link.topic_id, priority: next, is_active: link.is_active });
      setSiteTopics(prev => prev.map(l => l.id === linkId ? { ...l, priority: next } : l));
      toast({ title: "Priority updated", description: String(next) });
    } catch (e) {
      toast({ title: "Save failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
    }
  }

  // Helpers
  const formatStatusDot = (active: boolean) => (
    <span className={`inline-block h-2.5 w-2.5 rounded-full ${active ? "bg-green-500" : "bg-red-500"}`} aria-hidden />
  );

  const formatStrategy = (s: Site["strategy"]) => s.replaceAll("_", " ").toUpperCase();

  return (
    <div className="flex flex-col gap-4">
      {!currentSite ? (
        <div className="rounded-md border">
          <div className="p-3 border-b flex items-center justify-between gap-2">
            <div>
              <h3 className="text-sm font-medium">Sites</h3>
              <p className="text-xs text-muted-foreground">Select a site to assign topics.</p>
              <div className="mt-2"><Input className="h-8" placeholder="Search sites…" value={sitesQuery} onChange={e => setSitesQuery(e.target.value)} /></div>
            </div>
            <div className="self-start">
              <Button size="sm" variant="secondary" onClick={() => void loadBase()}>Refresh</Button>
            </div>
          </div>
          <div className="grid grid-cols-12 gap-2 text-xs text-muted-foreground font-medium px-3 py-2 border-b">
            <div className="col-span-5">Name</div>
            <div className="col-span-2 text-center">Topics</div>
            <div className="col-span-1 text-center">Active</div>
            <div className="col-span-2">Status</div>
            <div className="col-span-2">Strategy</div>
          </div>
          <div className="divide-y">
            {sites.filter(s => (s.name.toLowerCase().includes(sitesQuery.toLowerCase()) || s.url.toLowerCase().includes(sitesQuery.toLowerCase()))).map(s => {
              const st = siteStats[s.id];
              let topicsCol = "";
              if (s.strategy === "unique") topicsCol = String(st?.unique_topics_left ?? 0);
              else if (s.strategy === "round_robin") topicsCol = `${st?.round_robin_position ?? 0}/${st?.total_topics ?? 0}`;
              else topicsCol = "R";
              return (
                <button key={s.id} className="w-full grid grid-cols-12 gap-2 px-3 py-2 text-left hover:bg-muted/50" onClick={() => setCurrentSiteId(s.id)}>
                  <div className="col-span-5 truncate font-medium" title={s.url}>{s.name}</div>
                  <div className="col-span-2 text-center text-xs text-muted-foreground">{topicsCol}</div>
                  <div className="col-span-1 flex items-center justify-center">{formatStatusDot(s.is_active)}</div>
                  <div className={`col-span-2 text-xs ${s.status === "connected" ? "text-green-600" : s.status === "error" ? "text-red-600" : "text-muted-foreground"}`}>{s.status}</div>
                  <div className="col-span-2 text-xs text-muted-foreground">{formatStrategy(s.strategy)}</div>
                </button>
              );
            })}
            {sites.length === 0 && <div className="p-4 text-sm text-muted-foreground">No sites found.</div>}
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {/* Site card */}
          <div className="rounded-md border p-3 flex items-center justify-between">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-semibold truncate">{currentSite.name}</span>
                {formatStatusDot(currentSite.is_active)}
                <span className={`text-xs ${currentSite.status === "connected" ? "text-green-600" : currentSite.status === "error" ? "text-red-600" : "text-muted-foreground"}`}>{currentSite.status}</span>
              </div>
              <a className="block text-xs text-blue-600 truncate" href={currentSite.url} target="_blank" rel="noreferrer">{currentSite.url}</a>
            </div>
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <div><span className="text-xs">Topics:</span> <span className="font-medium">{siteTopics.length}</span></div>
              <div><span className="text-xs">Strategy:</span> <span className="font-medium">{formatStrategy(currentSite.strategy)}</span></div>
            </div>
            <div className="flex items-center gap-2">
              <Button size="sm" variant="secondary" onClick={() => void loadLinks(currentSite.id)}>Refresh</Button>
              <Button size="sm" variant="outline" onClick={() => setCurrentSiteId(null)}>Back</Button>
            </div>
          </div>

          {/* Assigned topics table */}
          <div className="rounded-md border">
            <div className="p-3 border-b flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-medium">Assigned topics</h3>
                <div className="text-xs text-muted-foreground">{filteredAssigned.length}</div>
              </div>
              <div className="flex items-center gap-2">
                <Input className="h-8 w-56" placeholder="Search assigned…" value={assignedQuery} onChange={e => setAssignedQuery(e.target.value)} />
                {selectedAssigned.size > 0 && (
                  <div className="flex items-center gap-2">
                    <Button size="sm" variant="outline" onClick={() => void toggleLinksBulk(Array.from(selectedAssigned), true)}>Enable</Button>
                    <Button size="sm" variant="outline" onClick={() => void toggleLinksBulk(Array.from(selectedAssigned), false)}>Disable</Button>
                    <Button size="sm" variant="destructive" onClick={() => void unassignLinksBulk(Array.from(selectedAssigned))}>Unassign</Button>
                  </div>
                )}
              </div>
            </div>
            <div className="grid grid-cols-12 gap-2 text-xs text-muted-foreground font-medium px-3 py-2 border-b">
              <div className="col-span-1 flex items-center">
                <Checkbox
                  checked={selectedAssigned.size > 0 && selectedAssigned.size === filteredAssigned.length}
                  onCheckedChange={(c: boolean) => setSelectedAssigned(c ? new Set(filteredAssigned.map(l => l.id)) : new Set())}
                  aria-label="Select all assigned"
                />
              </div>
              <div className="col-span-3">Title</div>
              <div className="col-span-3">Keywords</div>
              <div className="col-span-2">Category</div>
              <div className="col-span-2">Tags</div>
              <div className="col-span-1 text-center">Priority</div>
            </div>
            <div className="max-h-[360px] overflow-auto divide-y">
              {filteredAssigned.map(link => {
                const t = topicsById.get(link.topic_id);
                return (
                  <div key={link.id} className="grid grid-cols-12 items-center gap-2 px-3 py-2" data-selected={selectedAssigned.has(link.id) ? "true" : undefined}>
                    <div className="col-span-1 flex items-center">
                      <Checkbox checked={selectedAssigned.has(link.id)} onCheckedChange={(c: boolean) => setSelectedAssigned(prev => { const n = new Set(prev); c ? n.add(link.id) : n.delete(link.id); return n; })} aria-label={`Select ${t?.title ?? link.topic_title ?? `#${link.topic_id}`}`} />
                    </div>
                    <div className="col-span-3 truncate" title={t?.title}><span className="font-medium">{t?.title ?? link.topic_title ?? `#${link.topic_id}`}</span></div>
                    <div className="col-span-3 text-xs text-muted-foreground truncate" title={t?.keywords}>{t?.keywords ?? "—"}</div>
                    <div className="col-span-2 text-xs text-muted-foreground truncate" title={t?.category}>{t?.category ?? "—"}</div>
                    <div className="col-span-2 text-xs text-muted-foreground truncate" title={t?.tags}>{t?.tags ?? "—"}</div>
                    <div className="col-span-1 flex items-center justify-center gap-2">
                      <Input className="h-7 w-16 text-center" type="number" min={0} step={1}
                        value={priorityDrafts[link.id] ?? link.priority}
                        onChange={e => setPriorityDrafts(prev => ({ ...prev, [link.id]: Number(e.target.value) }))}
                        onBlur={() => void savePriority(link.id)}
                      />
                      <Switch checked={link.is_active} onCheckedChange={() => void toggleLinkActive(link.id)} />
                    </div>
                  </div>
                );
              })}
              {filteredAssigned.length === 0 && <div className="p-6 text-sm text-muted-foreground text-center">No topics assigned</div>}
            </div>
          </div>

          {/* Free topics table */}
          <div className="rounded-md border">
            <div className="p-3 border-b flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-medium">Free topics</h3>
                <div className="text-xs text-muted-foreground">{freeTopics.length}</div>
              </div>
              <div className="flex items-center gap-2">
                <Input className="h-8 w-56" placeholder="Search free…" value={topicsQuery} onChange={e => setTopicsQuery(e.target.value)} />
                {selectedFree.size > 0 && (
                  <div className="flex items-center gap-2">
                    <Button size="sm" onClick={() => void assignTopicsBulk(Array.from(selectedFree))}>Assign selected</Button>
                  </div>
                )}
              </div>
            </div>
            <div className="grid grid-cols-12 gap-2 text-xs text-muted-foreground font-medium px-3 py-2 border-b">
              <div className="col-span-1 flex items-center">
                <Checkbox
                  checked={selectedFree.size > 0 && selectedFree.size === freeTopics.length}
                  onCheckedChange={(c: boolean) => setSelectedFree(c ? new Set(freeTopics.map(t => t.id)) : new Set())}
                  aria-label="Select all free"
                />
              </div>
              <div className="col-span-4">Title</div>
              <div className="col-span-3">Keywords</div>
              <div className="col-span-2">Category</div>
              <div className="col-span-1">Tags</div>
              <div className="col-span-1 text-right">Active</div>
            </div>
            <div className="max-h-[360px] overflow-auto divide-y">
              {freeTopics.map(t => (
                <div key={t.id} className="grid grid-cols-12 items-center gap-2 px-3 py-2" data-selected={selectedFree.has(t.id) ? "true" : undefined}>
                  <div className="col-span-1 flex items-center">
                    <Checkbox checked={selectedFree.has(t.id)} onCheckedChange={(c: boolean) => setSelectedFree(prev => { const n = new Set(prev); c ? n.add(t.id) : n.delete(t.id); return n; })} aria-label={`Select ${t.title}`} />
                  </div>
                  <div className="col-span-4 truncate" title={t.title}><span className="font-medium">{t.title}</span></div>
                  <div className="col-span-3 text-xs text-muted-foreground truncate" title={t.keywords}>{t.keywords ?? "—"}</div>
                  <div className="col-span-2 text-xs text-muted-foreground truncate" title={t.category}>{t.category ?? "—"}</div>
                  <div className="col-span-1 text-xs text-muted-foreground truncate" title={t.tags}>{t.tags ?? "—"}</div>
                  <div className="col-span-1 flex items-center justify-end gap-2">
                    <span className={`inline-block h-2.5 w-2.5 rounded-full ${t.is_active ? "bg-green-500" : "bg-red-500"}`} />
                    <Button size="sm" onClick={() => void assignTopic(t.id)}>Assign</Button>
                  </div>
                </div>
              ))}
              {freeTopics.length === 0 && <div className="p-6 text-sm text-muted-foreground text-center">No free topics</div>}
            </div>
            <div className="p-3 border-t grid grid-cols-2 gap-3">
              <div>
                <div className="text-xs text-muted-foreground mb-1">Default priority</div>
                <Input className="h-8 w-28" type="number" min={0} step={1} value={assignDefaults.priority} onChange={e => setAssignDefaults(prev => ({ ...prev, priority: Number(e.target.value) }))} />
              </div>
              <div className="flex items-center gap-2">
                <div className="text-xs text-muted-foreground">Active by default</div>
                <Switch checked={assignDefaults.active} onCheckedChange={c => setAssignDefaults(prev => ({ ...prev, active: Boolean(c) }))} />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
