"use client";

import { useEffect, useMemo, useState, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatsFilters } from "./stats-filters";
import { useApiCall } from "@/hooks/use-api-call";
import { healthcheckService } from "@/services/healthcheck";
import { HealthCheckHistory } from "@/models/healthcheck";
import { toGoDateFormat } from "@/lib/time";
import {
  ChartContainer,
  ChartConfig,
  ChartTooltip,
} from "@/components/ui/chart";
import {
  ScatterChart,
  Scatter,
  XAxis,
  YAxis,
  ZAxis,
  CartesianGrid,
  ResponsiveContainer,
} from "recharts";
import { format, parseISO, isValid as isValidDate } from "date-fns";
import { RiArrowDownSLine, RiArrowRightSLine, RiBarChart2Line, RiTable2 } from "@remixicon/react";
import { DataTable } from "@/components/table/data-table";
import { ColumnDef } from "@tanstack/react-table";

interface HealthHistoryProps {
  siteId: number;
}

// Custom tooltip: show only Code and Error in a neat layout
function HealthTooltip({ active, payload }: any) {
  if (!active || !payload || !payload.length) return null;
  const p = payload[0]?.payload as any;
  const code = p?.code ?? p?.y ?? "—";
  const error = p?.error || "";
  // Prefer the point's own x timestamp; fallback to first raw item date
  const rawDate = typeof p?.x === 'number' ? new Date(p.x) : (p?.raws?.[0]?.checkedAt ? parseISO(p.raws[0].checkedAt) : null);
  const title = rawDate && isValidDate(rawDate) ? format(rawDate, "PPpp") : "";
  return (
    <div className="rounded-md border bg-background p-2 text-xs shadow-md max-w-[320px]">
      {title && <div className="mb-1 font-medium text-foreground">{title}</div>}
      <div className="grid grid-cols-[72px_1fr] gap-x-2 gap-y-1">
        <div className="text-muted-foreground">Code</div>
        <div className="font-medium">{code}</div>
        {error && (
          <>
            <div className="text-muted-foreground">Error</div>
            <div className="max-w-[240px] truncate" title={error}>{error}</div>
          </>
        )}
      </div>
    </div>
  );
}

export function HealthHistory({ siteId }: HealthHistoryProps) {
  const { execute } = useApiCall();
  const [dateRange, setDateRange] = useState<{ from: Date; to: Date }>();
  const [history, setHistory] = useState<HealthCheckHistory[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(50);
  const [hasMore, setHasMore] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [chartOpen, setChartOpen] = useState(false);
  const [tableOpen, setTableOpen] = useState(false);

  // Initial range: last 7 days
  useEffect(() => {
    const to = new Date();
    const from = new Date();
    from.setDate(from.getDate() - 7);
    setDateRange({ from, to });
  }, []);

  const loadPage = useCallback(async (from: Date, to: Date, pageToLoad: number, reset = false) => {
    const fromStr = toGoDateFormat(from);
    const toStr = toGoDateFormat(to);
    setIsLoading(true);
    try {
      const resp = await execute(
        () => healthcheckService.getHistoryByPeriod(siteId, fromStr, toStr, pageToLoad, pageSize),
        {
          errorTitle: "Failed to load health history",
        }
      );
      const items = resp?.items || [];
      setHistory(prev => (reset ? items : [...prev, ...items]));
      setHasMore(Boolean(resp?.hasMore));
      setPage(pageToLoad + 1);
    } finally {
      setIsLoading(false);
    }
  }, [execute, siteId, pageSize]);

  useEffect(() => {
    if (dateRange?.from && dateRange?.to) {
      // reset to first page on range change
      setHistory([]);
      setPage(1);
      setHasMore(false);
      loadPage(dateRange.from, dateRange.to, 1, true);
    }
  }, [dateRange, loadPage]);

  const toStatusBand = (code?: number) => {
    if (!code || code < 100) return 0; // no response/unknown
    const band = Math.floor(code / 100) * 100;
    if (band >= 200 && band <= 599) return band;
    return 0;
  };

  // Aggregate by minute+status band to make dense data larger
  const chartData = useMemo(() => {
    type Point = { x: number; y: number; z: number; count: number; sumResp: number; raws: HealthCheckHistory[]; code?: number; error?: string };
    const healthyMap = new Map<string, Point>();
    const unhealthyMap = new Map<string, Point>();

    history.forEach((h) => {
      const t = parseISO(h.checkedAt);
      if (!isValidDate(t)) return;
      const tsMin = Math.floor(t.getTime() / 60000);
      const band = toStatusBand(h.statusCode);
      // Skip non-HTTP/no-response bands; show only 2xx and above
      if (band < 200) return;
      const isHealthy = (h.status || "").toLowerCase() === "healthy";
      const key = `${tsMin}_${band}`;
      const map = isHealthy ? healthyMap : unhealthyMap;
      const existing = map.get(key);
      if (existing) {
        existing.count += 1;
        existing.sumResp += Number(h.responseTimeMs || 0);
        if (!existing.error && h.errorMessage) existing.error = h.errorMessage;
        if (!existing.code && h.statusCode) existing.code = h.statusCode;
        existing.raws.push(h);
      } else {
        map.set(key, {
          x: tsMin * 60000,
          y: band,
          z: 0, // will compute later
          count: 1,
          sumResp: Number(h.responseTimeMs || 0),
          raws: [h],
          code: h.statusCode,
          error: h.errorMessage || undefined,
        });
      }
    });

    const sizeFromMs = (ms: number, isUnhealthy: boolean, count: number) => {
      const clamp = (v: number, min: number, max: number) => Math.max(min, Math.min(max, v));
      if (!ms || ms <= 0) {
        // fallback sizes
        const base = isUnhealthy ? 28 : 20;
        const growth = Math.log10(Math.max(1, count)) * 6;
        return clamp(base + growth, 16, 64);
      }
      const normalized = clamp(ms, 100, 4000); // 0.1s to 4s
      const ratio = (normalized - 100) / (4000 - 100);
      const base = 16 + ratio * 40; // 16..56
      const growth = Math.log10(Math.max(1, count)) * 6; // + density
      return clamp(base + growth, 16, 72);
    };

    const finalize = (m: Map<string, Point>, isUnhealthy: boolean) =>
      Array.from(m.values()).map((p) => ({
        ...p,
        z: sizeFromMs(p.sumResp / p.count, isUnhealthy, p.count),
      }));

    return { healthy: finalize(healthyMap, false), unhealthy: finalize(unhealthyMap, true) };
  }, [history]);

  const chartConfig = {
    healthy: { label: "Healthy", color: "hsl(145, 85%, 50%)" },
    unhealthy: { label: "Unhealthy", color: "hsl(4, 95%, 62%)" },
  } satisfies ChartConfig;

  const unhealthyRows = useMemo(() => {
    return history
      .filter((h) => (h.status || "").toLowerCase() !== "healthy")
      .sort((a, b) => new Date(b.checkedAt).getTime() - new Date(a.checkedAt).getTime());
  }, [history]);

  const hasHistory = history.length > 0;
  const hasUnhealthy = unhealthyRows.length > 0;

  const columns: ColumnDef<HealthCheckHistory>[] = useMemo(() => {
    return [
      {
        header: "Date",
        accessorKey: "checkedAt",
        cell: ({ row }) => {
          const dt = parseISO(row.original.checkedAt);
          return isValidDate(dt) ? format(dt, "PP") : "—";
        },
      },
      {
        header: "Time",
        accessorKey: "time",
        cell: ({ row }) => {
          const dt = parseISO(row.original.checkedAt);
          return isValidDate(dt) ? format(dt, "HH:mm") : "—";
        },
      },
      {
        header: "Status",
        accessorKey: "status",
        cell: ({ row }) => (
          <span className="font-medium text-destructive">{row.original.status}</span>
        ),
      },
      {
        header: "Code",
        accessorKey: "statusCode",
        cell: ({ row }) => row.original.statusCode || "—",
      },
      {
        header: "Message",
        accessorKey: "errorMessage",
        cell: ({ row }) => (
          <span className="block max-w-[520px] truncate" title={row.original.errorMessage || ""}>
            {row.original.errorMessage || "—"}
          </span>
        ),
      },
    ];
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Health</h2>
        <StatsFilters
          dateRange={dateRange}
          onDateRangeChange={(range) => {
            if (range?.from && range?.to) setDateRange(range);
          }}
        />
      </div>

      {hasHistory && (
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <RiBarChart2Line className="w-5 h-5 text-muted-foreground" />
                <div>
                  <CardTitle className="text-lg">Health Check Timeline</CardTitle>
                  <p className="text-sm text-muted-foreground">Lanes by status class; larger bubbles = more checks</p>
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setChartOpen((v) => !v)}
                className="h-8 w-8 p-0"
              >
                {chartOpen ? (
                  <RiArrowDownSLine className="w-4 h-4" />
                ) : (
                  <RiArrowRightSLine className="w-4 h-4" />
                )}
              </Button>
            </div>
          </CardHeader>
          {chartOpen && (
              <CardContent className="pt-0">
                  <ChartContainer config={chartConfig} className="h-[260px] w-full">
                      <ResponsiveContainer width="100%" height={260}>
                          <ScatterChart margin={{ left: 6, right: 6, top: 4, bottom: 4 }}>
                              <CartesianGrid vertical={false} />
                              <XAxis
                                  dataKey="x"
                                  type="number"
                                  domain={["auto", "auto"]}
                                  tickFormatter={(val) => {
                                      const d = new Date(Number(val));
                                      return isNaN(d.getTime()) ? "" : format(d, "MM-dd HH:mm");
                                  }}
                                  tickLine={false}
                                  axisLine={false}
                                  tickMargin={6}
                                  fontSize={11}
                              />
                              <YAxis
                                  dataKey="y"
                                  name="Status Class"
                                  domain={[200, 599]}
                                  ticks={[200, 300, 400, 500]}
                                  tickFormatter={(v) => `${Math.floor(Number(v) / 100)}xx`}
                                  tickLine={false}
                                  axisLine={false}
                                  tickMargin={6}
                                  fontSize={11}
                                  width={56}
                              />
                              <ZAxis dataKey="z" range={[24, 72]} />
                              <ChartTooltip content={<HealthTooltip />} />
                              <Scatter name="Healthy" data={chartData.healthy as any} fill="var(--color-healthy)" />
                              <Scatter name="Unhealthy" data={chartData.unhealthy as any} fill="var(--color-unhealthy)" />
                          </ScatterChart>
                      </ResponsiveContainer>
                  </ChartContainer>
              </CardContent>
          )}
        </Card>
      )}

      {hasUnhealthy && (
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <RiTable2 className="w-5 h-5 text-muted-foreground" />
                <div>
                  <CardTitle className="text-lg">Daily Breakdown (Unhealthy only)</CardTitle>
                  <p className="text-sm text-muted-foreground">Only issues are shown for the selected period</p>
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setTableOpen((v) => !v)}
                className="h-8 w-8 p-0"
              >
                {tableOpen ? (
                  <RiArrowDownSLine className="w-4 h-4" />
                ) : (
                  <RiArrowRightSLine className="w-4 h-4" />
                )}
              </Button>
            </div>
          </CardHeader>
          {tableOpen && (
            <CardContent className="pt-0">
              <DataTable
                columns={columns}
                data={unhealthyRows}
                searchKey="errorMessage"
                searchPlaceholder="Search errors..."
                isLoading={isLoading}
                emptyMessage="No unhealthy events in selected period."
                showPagination={true}
                defaultPageSize={25}
                enableViewOption={false}
              />
              {hasMore && (
                <div className="pt-4 flex justify-center">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => dateRange && loadPage(dateRange.from, dateRange.to, page, false)}
                    disabled={isLoading}
                  >
                    {isLoading ? "Loading..." : "Load more"}
                  </Button>
                </div>
              )}
            </CardContent>
          )}
        </Card>
      )}

      {!hasHistory && (
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-8 text-sm text-muted-foreground">
              No health data available for the selected period.
            </div>
            {hasMore && (
              <div className="pt-4 flex justify-center">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => dateRange && loadPage(dateRange.from, dateRange.to, page, false)}
                  disabled={isLoading}
                >
                  {isLoading ? "Loading..." : "Load more"}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

export default HealthHistory;
