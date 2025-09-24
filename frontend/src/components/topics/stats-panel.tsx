"use client";
import * as React from "react";

export default function StatsPanel() {
    return (
        <div>CCC</div>
    );

  /*const { toast } = useToast();

  // Data sources
  const [sites, setSites] = React.useState<Site[]>([]);
  const [siteTopics, setSiteTopics] = React.useState<any[]>([]); // SiteTopicLink-like
  const [topicsById, setTopicsById] = React.useState<Map<number, Topic>>(new Map());

  // Selections & filters
  const [selectedSiteId, setSelectedSiteId] = React.useState<number | null>(null);
  const [selectedTopicId, setSelectedTopicId] = React.useState<number | "all">("all");
  const [fromDate, setFromDate] = React.useState<string>(""); // yyyy-mm-dd
  const [toDate, setToDate] = React.useState<string>("");

  // Stats/availability/preview
  const [stats, setStats] = React.useState<any | null>(null);
  const [availability, setAvailability] = React.useState<any | null>(null);
  const [preview, setPreview] = React.useState<any | null>(null);

  // History
  const [history, setHistory] = React.useState<any[]>([]);
  const [page, setPage] = React.useState<number>(1);
  const pageSize = 100;
  const [total, setTotal] = React.useState<number>(0);

  // UI state
  const [loadingSites, setLoadingSites] = React.useState(false);
  const [loadingAll, setLoadingAll] = React.useState(false);

  const currentSite = React.useMemo(() => sites.find(s => s.id === selectedSiteId) ?? null, [sites, selectedSiteId]);

  React.useEffect(() => {
    async function loadSites() {
      try {
        setLoadingSites(true);
        const svc = await import("@/services/sites");
        const { items } = await svc.getSites(1, 1000);
        setSites(items);
      } catch (e) {
        toast({ title: "Failed to load sites", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
      } finally {
        setLoadingSites(false);
      }
    }
    void loadSites();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Load topics dictionary (for titles if needed)
  React.useEffect(() => {
    async function loadTopicsDict() {
      try {
        const svc = await import("@/services/topics");
        const { items } = await svc.getTopics(1, 10000);
        setTopicsById(new Map(items.map((t: Topic) => [t.id, t])));
      } catch {
        // Non-critical
      }
    }
    void loadTopicsDict();
  }, []);

  const formatStrategy = (s: Site["strategy"]) => s.replaceAll("_", " ").toUpperCase();

  const formatDateTimeEU = (iso?: string): string => {
    if (!iso) return "—";
    try {
      const d = new Date(iso);
      return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(d);
    } catch {
      return iso as string;
    }
  };

  const refreshForSite = React.useCallback(async (siteId: number) => {
    setLoadingAll(true);
    try {
      const statsSvc = await import("@/services/stats");
      const linksSvc = await import("@/services/siteTopics");

      // stats & availability
      const [st, linksRes] = await Promise.all([
        statsSvc.getTopicStats(siteId).catch(() => null),
        linksSvc.getSiteTopics(siteId, 1, 10000),
      ]);
      setStats(st);
      setSiteTopics(linksRes.items ?? []);

      // Check availability based on current site strategy (read-only)
      const site = sites.find(s => s.id === siteId);
      if (site) {
        try {
          const avail = await statsSvc.checkStrategyAvailability(siteId, site.strategy);
          setAvailability(avail);
        } catch {
          setAvailability(null);
        }
        // Do NOT auto-select next topic during refresh/site load to avoid side effects on RR position.
        // Keep existing preview as-is; user can trigger preview explicitly via the button.
      }

      // Load first page of history for the site (or topic if selected)
      await loadHistory(siteId, selectedTopicId, 1, fromDate, toDate);
    } catch (e) {
      toast({ title: "Load failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
    } finally {
      setLoadingAll(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sites, selectedTopicId, fromDate, toDate]);

  async function loadHistory(siteId: number, topicId: number | "all", pageNum: number, from: string, to: string) {
    try {
      const statsSvc = await import("@/services/stats");
      const useTopic = topicId !== "all" && topicId != null;
      const res = useTopic
        ? await statsSvc.getTopicUsageHistory(siteId, Number(topicId), pageNum, pageSize)
        : await statsSvc.getSiteUsageHistory(siteId, pageNum, pageSize);

      const items = (res.usage_history ?? []) as any[];
      const pagination = res.pagination ?? { page: pageNum, total: items.length };

      // Client-side date range filter if provided
      const filtered = items.filter((row: any) => {
        const usedAt: string | undefined = (row.used_at as string) || (row.timestamp as string) || undefined;
        if (!from && !to) return true;
        if (!usedAt) return false;
        const d = new Date(usedAt);
        if (from) {
          const fromD = new Date(from + "T00:00:00");
          if (d < fromD) return false;
        }
        if (to) {
          const toD = new Date(to + "T23:59:59");
          if (d > toD) return false;
        }
        return true;
      });

      setHistory(filtered);
      setPage(pagination.page ?? pageNum);
      setTotal(pagination.total ?? filtered.length);
    } catch (e) {
      toast({ title: "Failed to load history", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
      setHistory([]);
      setTotal(0);
    }
  }

  // Compute used/unused from siteTopics when stats may not include it
  const usedCount = React.useMemo(() => siteTopics.filter((l: any) => (l.usage_count ?? 0) > 0).length, [siteTopics]);
  const totalCount = React.useMemo(() => (stats?.total_topics ?? siteTopics.length ?? 0) as number, [stats, siteTopics.length]);
  const activeCount = React.useMemo(() => (stats?.active_topics ?? siteTopics.filter((l: any) => l.is_active).length ?? 0) as number, [stats, siteTopics]);
  const unusedCount = Math.max(0, totalCount - usedCount);

  // Strategy flags and global counts
  const strategyStr = (currentSite?.strategy as unknown as string) ?? "";
  const isRR = strategyStr === "round_robin";
  const isUnique = strategyStr === "unique";
  const isRandomAll = strategyStr === "random_all";
  const isRandom = strategyStr === "random" || isRandomAll;
  const activeAll = React.useMemo(() => Array.from(topicsById.values()).filter((t) => t.is_active).length, [topicsById]);

  // Derived Remaining for availability block
  const remainingForDisplay = React.useMemo(() => {
    if (!currentSite) return 0;
    if (isRR) return Math.max(0, (stats?.total_topics ?? totalCount ?? 0) - (stats?.round_robin_position ?? 0));
    if (isUnique) return stats?.unique_topics_left ?? 0;
    if (isRandomAll) return activeAll;
    if (isRandom) return activeCount;
    return (availability?.remaining_count as number) ?? 0;
  }, [currentSite, isRR, isUnique, isRandomAll, isRandom, stats, totalCount, activeAll, activeCount, availability]);

  // Top used topics (from siteTopics usage_count)
  const topUsed = React.useMemo(() => {
    const list = siteTopics.slice().sort((a: any, b: any) => (b.usage_count ?? 0) - (a.usage_count ?? 0));
    return list.slice(0, 10);
  }, [siteTopics]);

  // Handlers
  const onChangeSite = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const raw = e.target.value;
    // When selection is cleared, prevent any requests and reset state
    if (raw === "") {
      setSelectedSiteId(null);
      setSelectedTopicId("all");
      setStats(null);
      setAvailability(null);
      setPreview(null);
      setSiteTopics([]);
      setHistory([]);
      setTotal(0);
      setPage(1);
      return;
    }

    const id = Number(raw);
    if (Number.isFinite(id) && id > 0) {
      setSelectedSiteId(id);
      setSelectedTopicId("all");
      // Clear preview when switching sites; user can explicitly preview if needed
      setPreview(null);
      await refreshForSite(id);
    } else {
      setSelectedSiteId(null);
    }
  };

  const refresh = async () => {
    if (currentSite) await refreshForSite(currentSite.id);
  };

  const onApplyFilters = async () => {
    if (currentSite) await loadHistory(currentSite.id, selectedTopicId, 1, fromDate, toDate);
  };

  const onChangeTopic = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value === "all" ? "all" : Number(e.target.value);
    setSelectedTopicId(val);
    if (currentSite) await loadHistory(currentSite.id, val, 1, fromDate, toDate);
  };

  const gotoPage = async (next: number) => {
    if (!currentSite) return;
    setPage(next);
    await loadHistory(currentSite.id, selectedTopicId, next, fromDate, toDate);
  };

  return (
    <div className="flex flex-col gap-4">
      {/!* Site selector *!/}
      <div className="rounded-md border p-3 flex items-center justify-between gap-2">
        <div className="flex flex-col sm:flex-row sm:items-center gap-2">
          <div className="text-sm">
            <div className="text-muted-foreground">Site</div>
            <select className="h-9 border rounded-md px-2 bg-background min-w-[240px]" value={selectedSiteId ?? ""} onChange={onChangeSite} aria-label="Select site">
              <option value="">Select site…</option>
              {sites.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
          </div>
          {currentSite && (
            <div className="text-sm">
              <div className="text-muted-foreground">Strategy</div>
              <div className="font-medium">{formatStrategy(currentSite.strategy)}</div>
            </div>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="secondary" onClick={refresh}>Refresh</Button>
        </div>
      </div>

      {/!* Metrics & availability *!/}
      {currentSite && (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-2">
          <div className="rounded-md border p-3"><div className="text-xs text-muted-foreground">Total</div><div className="text-lg font-semibold">{isRandomAll ? activeAll : totalCount}</div></div>
          <div className="rounded-md border p-3"><div className="text-xs text-muted-foreground">Active</div><div className="text-lg font-semibold">{isRandomAll ? activeAll : activeCount}</div></div>
          {!isRR && (
            <>
              <div className="rounded-md border p-3"><div className="text-xs text-muted-foreground">Used</div><div className="text-lg font-semibold">{usedCount}</div></div>
              <div className="rounded-md border p-3"><div className="text-xs text-muted-foreground">Unused</div><div className="text-lg font-semibold">{unusedCount}</div></div>
            </>
          )}
          {isUnique && (
            <div className="rounded-md border p-3"><div className="text-xs text-muted-foreground">Unique Left</div><div className="text-lg font-semibold">{stats?.unique_topics_left ?? 0}</div></div>
          )}
          {isRR && (
            <div className="rounded-md border p-3"><div className="text-xs text-muted-foreground">Round Robin Position</div><div className="text-lg font-semibold">{stats?.round_robin_position ?? 0}</div></div>
          )}
        </div>
      )}

      {currentSite && (
        <div className="rounded-md border p-3 flex flex-col md:flex-row gap-3 items-start md:items-center justify-between">
          <div className="text-sm">
            <div className="font-medium">Check availability</div>
            <div className="text-muted-foreground text-xs">Based on strategy {formatStrategy(currentSite.strategy)}</div>
            <div className="mt-1 text-sm flex gap-4">
              <div>Can continue: <span className={availability?.can_continue ? "text-green-600" : "text-red-600"}>{String(availability?.can_continue ?? false)}</span></div>
              <div>Remaining: <span className="font-medium">{remainingForDisplay}</span></div>
              <div>Active/Unused/Total: <span className="font-medium">{isRandomAll ? activeAll : activeCount}/{unusedCount}/{isRandomAll ? activeAll : totalCount}</span></div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" onClick={async () => {
              if (!currentSite) return;
              try {
                const statsSvc = await import("@/services/stats");
                const res = await statsSvc.selectNextTopicForSite(currentSite.id, currentSite.strategy);
                setPreview(res);
                const title = res?.topic?.title ?? res?.site_topic?.topic_title ?? (res?.topic?.id ? `#${res.topic.id}` : res?.site_topic?.topic_id ? `#${res.site_topic.topic_id}` : "");
                toast({ title: "Preview updated", description: title });
              } catch (e) {
                toast({ title: "Preview failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
              }
            }}>Preview Next Topic</Button>
            <Button size="sm" disabled={!availability?.can_continue} onClick={async () => {
              if (!currentSite) return;
              try {
                const statsSvc = await import("@/services/stats");
                const res = await statsSvc.selectNextTopicForSite(currentSite.id, currentSite.strategy);
                setPreview(res);
                const title = res?.topic?.title ?? res?.site_topic?.topic_title ?? (res?.topic?.id ? `#${res.topic.id}` : res?.site_topic?.topic_id ? `#${res.site_topic.topic_id}` : "");
                toast({ title: availability?.can_continue ? "Continue available" : "Cannot continue", description: title });
              } catch (e) {
                toast({ title: "Continue failed", description: e instanceof Error ? e.message : String(e), variant: "destructive" });
              }
            }}>Continue</Button>
          </div>
        </div>
      )}

      {/!* Preview card *!/}
      {currentSite && preview && (
        <div className="rounded-md border p-3 text-sm">
          <div className="text-muted-foreground">Next topic preview</div>
          <div className="font-medium">{preview?.topic?.title ?? preview?.site_topic?.topic_title ?? topicsById.get(preview?.site_topic?.topic_id ?? preview?.topic?.id)?.title ?? `#${(preview?.topic?.id ?? preview?.site_topic?.topic_id) ?? ""}`}</div>
        </div>
      )}

      {/!* History controls *!/}
      {currentSite && (
        <div className="rounded-md border p-3 flex flex-col md:flex-row md:items-end gap-3">
          <div className="text-sm">
            <div className="text-muted-foreground">Topic</div>
            <select className="h-9 border rounded-md px-2 bg-background min-w-[220px]" value={selectedTopicId} onChange={onChangeTopic} aria-label="Filter by topic">
              <option value="all">All topics</option>
              {siteTopics.map((l: any) => (
                <option key={l.topic_id} value={l.topic_id}>{topicsById.get(l.topic_id)?.title ?? l.topic_title ?? `#${l.topic_id}`}</option>
              ))}
            </select>
          </div>
          <div className="text-sm">
            <div className="text-muted-foreground">From</div>
            <Input type="date" className="h-9" value={fromDate} onChange={(e) => setFromDate(e.target.value)} />
          </div>
          <div className="text-sm">
            <div className="text-muted-foreground">To</div>
            <Input type="date" className="h-9" value={toDate} onChange={(e) => setToDate(e.target.value)} />
          </div>
          <div className="flex-1" />
          <div className="flex items-center gap-2">
            <Button size="sm" variant="secondary" onClick={onApplyFilters}>Apply</Button>
          </div>
        </div>
      )}

      {/!* History table *!/}
      {currentSite && (
        <div className="rounded-md border">
          <div className="p-3 border-b flex items-center justify-between">
            <div className="text-sm font-medium">Topic usage history</div>
            <div className="text-xs text-muted-foreground">{history.length} item(s){total ? ` of ${total}` : ""}</div>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[220px]">Used at</TableHead>
                <TableHead>Topic</TableHead>
                <TableHead className="w-[140px]">Strategy</TableHead>
                <TableHead className="w-[140px]">Article ID</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {history.map((row: any, idx: number) => (
                <TableRow key={idx}>
                  <TableCell className="text-sm text-muted-foreground">{formatDateTimeEU((row.used_at as string) ?? (row.timestamp as string))}</TableCell>
                  <TableCell className="font-medium truncate" title={(row.topic_title as string) ?? topicsById.get(row.topic_id)?.title}>{(row.topic_title as string) ?? topicsById.get(row.topic_id)?.title ?? `#${row.topic_id}`}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{String(row.strategy ?? currentSite?.strategy ?? "").toString().replaceAll("_", " ").toUpperCase()}</TableCell>
                  <TableCell className="text-sm">{row.article_id ?? "—"}</TableCell>
                </TableRow>
              ))}
              {history.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-sm text-muted-foreground py-8">No history found.</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          <div className="p-3 flex items-center justify-between">
            <div className="text-xs text-muted-foreground">Page {page}</div>
            <Pagination>
              <PaginationContent>
                <PaginationItem>
                  <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => void gotoPage(page - 1)}>Prev</Button>
                </PaginationItem>
                <PaginationItem>
                  <Button size="sm" variant="outline" disabled={history.length < pageSize} onClick={() => void gotoPage(page + 1)}>Next</Button>
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          </div>
        </div>
      )}

      {/!* Top used topics *!/}
      {currentSite && (
        <div className="rounded-md border">
          <div className="p-3 border-b flex items-center justify-between">
            <div className="text-sm font-medium">Top used topics</div>
            <div className="text-xs text-muted-foreground">Top {Math.min(10, topUsed.length)}</div>
          </div>
          <div className="max-h-[360px] overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Topic</TableHead>
                  <TableHead className="w-[140px] text-right">Used</TableHead>
                  <TableHead className="w-[200px]">Last used</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {topUsed.map((l: any) => (
                  <TableRow key={l.id}>
                    <TableCell className="truncate" title={topicsById.get(l.topic_id)?.title ?? l.topic_title}>{topicsById.get(l.topic_id)?.title ?? l.topic_title ?? `#${l.topic_id}`}</TableCell>
                    <TableCell className="text-right">{l.usage_count ?? 0}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{formatDateTimeEU(l.last_used_at as string)}</TableCell>
                  </TableRow>
                ))}
                {topUsed.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} className="text-center text-sm text-muted-foreground py-8">No usage data.</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      )}

      {(loadingSites || loadingAll) && <div className="text-sm text-muted-foreground">Loading…</div>}
    </div>
  );*/
}
