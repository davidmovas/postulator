"use client";
import * as React from "react";
import type { Site } from "@/types/site";
import { SiteStatusBadge } from "@/components/badges/site-status-badge";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { RiRefreshLine, RiSearchLine } from "@remixicon/react";
import { ClientTime } from "@/components/common/ClientTime";

export interface SitesTableProps {
  sites: Site[];
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onRefresh?: () => void;
  className?: string;
}

export function SitesTable({ sites, page, pageSize, total, onPageChange, onRefresh, className }: SitesTableProps) {
  const [query, setQuery] = React.useState<string>("");

  // debounce search client-side (placeholder; in real app trigger backend)
  const [q, setQ] = React.useState<string>("");
  React.useEffect(() => {
    const t = setTimeout(() => setQ(query.trim()), 300);
    return () => clearTimeout(t);
  }, [query]);

  const filtered = React.useMemo(() => {
    if (!q) return sites;
    const lower = q.toLowerCase();
    return sites.filter(s => s.name.toLowerCase().includes(lower) || s.url.toLowerCase().includes(lower));
  }, [q, sites]);

  const pages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      <div className="flex items-center gap-2">
        <div className="relative w-72">
          <RiSearchLine className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-muted-foreground/50" size={18} />
          <Input
            placeholder="Search sites..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="pl-8"
          />
        </div>
        <Button variant="outline" size="sm" onClick={onRefresh} title="Refresh">
          <RiRefreshLine size={16} />
        </Button>
      </div>

      <div className="w-full overflow-auto rounded-md border">
        <table className="w-full text-sm">
          <thead className="bg-muted/30">
            <tr className="text-left">
              <th className="px-3 py-2 font-medium">Name</th>
              <th className="px-3 py-2 font-medium">URL</th>
              <th className="px-3 py-2 font-medium">Status</th>
              <th className="px-3 py-2 font-medium">Last Check</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((s) => (
              <tr key={s.id} className="border-t">
                <td className="px-3 py-2">{s.name}</td>
                <td className="px-3 py-2 text-muted-foreground"><a href={s.url} className="underline-offset-4 hover:underline" target="_blank" rel="noreferrer">{s.url}</a></td>
                <td className="px-3 py-2"><SiteStatusBadge status={s.status} /></td>
                <td className="px-3 py-2 text-muted-foreground">{s.last_check_at ? (
                  <>
                    <time suppressHydrationWarning dateTime={s.last_check_at} className="sr-only">{s.last_check_at}</time>
                    <ClientTime iso={s.last_check_at} />
                  </>
                ) : "—"}</td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-10 text-center text-muted-foreground">No sites found</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground">
          Page {page} of {pages} • {total} total
        </span>
        <div className="flex items-center gap-1">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>Prev</Button>
          <Button variant="outline" size="sm" disabled={page >= pages} onClick={() => onPageChange(page + 1)}>Next</Button>
        </div>
      </div>
    </div>
  );
}

export default SitesTable;
