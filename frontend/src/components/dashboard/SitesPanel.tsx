"use client";
import * as React from "react";
import SitesTable from "@/components/tables/sites-table";
import type { Site } from "@/types/site";
import { useToast } from "@/components/ui/use-toast";

export function SitesPanel() {
  const [page, setPage] = React.useState<number>(1);
  const pageSize = 100;
  const [sites, setSites] = React.useState<Site[]>([]);
  const [total, setTotal] = React.useState<number>(0);
  const [loading, setLoading] = React.useState<boolean>(false);
  const [error, setError] = React.useState<string | undefined>(undefined);
  const { toast } = useToast();

  async function load() {
    try {
      setLoading(true);
      setError(undefined);
      const { items, total } = await import("@/services/sites").then(m => m.getSites(page, pageSize));
      setSites(items);
      setTotal(total);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load sites");
      setSites([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }

  React.useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  const refresh = () => void load();

  return (
    <div className="p-4 md:p-6 lg:p-8">
      <div className="mb-3">
        <h1 className="text-xl font-semibold">Sites</h1>
        <p className="text-sm text-muted-foreground">Manage your WordPress sites.</p>
      </div>
      {error && <div className="mb-3 text-sm text-destructive">{error}</div>}
      <div className="mb-4 grid grid-cols-2 md:grid-cols-4 gap-2 text-sm">
        <div className="rounded-md border p-3"><div className="text-muted-foreground">Total</div><div className="text-lg font-semibold">{total}</div></div>
        <div className="rounded-md border p-3"><div className="text-muted-foreground">Active</div><div className="text-lg font-semibold">{sites.filter(s=>s.is_active).length}</div></div>
        <div className="rounded-md border p-3"><div className="text-muted-foreground">Connected</div><div className="text-lg font-semibold">{sites.filter(s=>s.status==="connected").length}</div></div>
        <div className="rounded-md border p-3"><div className="text-muted-foreground">Pending</div><div className="text-lg font-semibold">{sites.filter(s=>s.status==="pending").length}</div></div>
      </div>
      <SitesTable
        sites={sites}
        page={page}
        pageSize={pageSize}
        total={total}
        onPageChange={setPage}
        onRefresh={refresh}
        onCreate={async (v) => {
          const svc = await import("@/services/sites");
          await svc.createSite({ name: v.name, url: v.url, username: v.username, password: v.password, is_active: v.is_active, strategy: v.strategy });
        }}
        onUpdate={async (id, v) => {
          const svc = await import("@/services/sites");
          await svc.updateSite(id, { name: v.name, url: v.url, username: v.username, password: v.password, is_active: v.is_active, strategy: v.strategy });
        }}
        onDelete={async (id) => {
          const svc = await import("@/services/sites");
          await svc.deleteSite(id);
        }}
        onToggleActive={async (id, active) => {
          const svc = await import("@/services/sites");
          await svc.setSiteActive(id, active);
        }}
        onBulkToggle={async (ids, active) => {
          const svc = await import("@/services/sites");
          await Promise.all(ids.map((id) => svc.setSiteActive(id, active)));
        }}
        onBulkDelete={async (ids) => {
          const svc = await import("@/services/sites");
          await Promise.all(ids.map((id) => svc.deleteSite(id)));
        }}
        onTestConnection={async (id) => {
          const svc = await import("@/services/sites");
          try {
            const res = await svc.testConnection(id);
            toast({
              title: res.success ? "Connection successful" : `Connection ${res.status}`,
              description: res.details ? `${res.message}\n${res.details}` : res.message,
              variant: res.success ? "success" : "destructive",
            });
          } catch (e) {
            const msg = e instanceof Error ? e.message : "Failed to test connection";
            toast({ title: "Test failed", description: msg, variant: "destructive" });
          }
        }}
        onMutateSites={(updater) => setSites((prev) => updater(prev))}
      />
      {loading && <div className="mt-3 text-sm text-muted-foreground">Loading...</div>}
    </div>
  );
}

export default SitesPanel;
