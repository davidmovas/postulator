"use client";
import * as React from "react";
import SitesTable from "@/components/tables/sites-table";
import type { Site } from "@/types/site";

export function SitesPanel() {
  const [page, setPage] = React.useState<number>(1);
  const pageSize = 10;

  // Placeholder data: replace with Wails call later
  // Avoid generating dates during initial render to prevent SSR/CSR mismatches.
  const [sites, setSites] = React.useState<Site[]>(() => [
    { id: 1, name: "Example Blog", url: "https://example.com", is_active: true, status: "connected" },
    { id: 2, name: "Dev Notes", url: "https://dev.local", is_active: false, status: "disabled" },
    { id: 3, name: "WP Site", url: "https://wp.site", is_active: true, status: "pending" },
  ]);

  // Set demo last_check_at after mount (client-only) to avoid SSR hydration mismatch.
  React.useEffect(() => {
    setSites((prev) => prev.map((s) => s.id === 1 ? { ...s, last_check_at: new Date().toISOString() } : s));
  }, []);

  const total = sites.length; // demo only

  return (
    <div className="p-4 md:p-6 lg:p-8">
      <div className="mb-3">
        <h3 className="text-xl font-semibold">Sites</h3>
        <p className="text-sm text-muted-foreground">Manage your WordPress sites.</p>
      </div>
      <div className="mb-4 grid grid-cols-2 md:grid-cols-4 gap-2 text-sm">
        <div className="rounded-md border p-3"><div className="text-muted-foreground">Total</div><div className="text-lg font-semibold">{sites.length}</div></div>
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
        onRefresh={() => setSites([...sites])}
        onMutateSites={(updater) => setSites((prev) => updater(prev))}
      />
    </div>
  );
}

export default SitesPanel;
