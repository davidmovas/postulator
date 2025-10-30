"use client";

import React, { useMemo, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Database, RefreshCw, MoreVertical, Upload, RefreshCw as RefreshIcon } from "lucide-react";
import { Site } from "@/services/site";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export interface SiteTopicStats {
  siteId: number;
  total: number;
  unused: number;
}

export interface TopicsSitesTableProps {
  sites: Site[];
  isLoading?: boolean;
  stats: Record<number, SiteTopicStats | undefined>;
  onManage: (siteId: number) => void;
  onImport: (siteId: number) => void;
  onSyncCategories: (siteId: number) => void | Promise<void>;
  onRefresh?: () => void | Promise<void>;
  onOpenImportDialog?: () => void;
  isRefreshing?: boolean;
}

type SortField = "name" | "url" | "total" | "unused";
type SortDirection = "asc" | "desc" | null;

export function TopicsSitesTable({ sites, isLoading = false, stats, onManage, onImport, onSyncCategories, onRefresh, onOpenImportDialog, isRefreshing }: TopicsSitesTableProps) {
  const rows = useMemo(() => {
    return sites.map((s) => ({
      site: s,
      stats: stats[s.id] || { siteId: s.id, total: 0, unused: 0 },
    }));
  }, [sites, stats]);

  const [search, setSearch] = useState("");
  const [sortField, setSortField] = useState<SortField | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);

  const filtered = useMemo(() => {
    const q = search.toLowerCase().trim();
    if (!q) return rows;
    return rows.filter(({ site }) =>
      site.name.toLowerCase().includes(q) || site.url.toLowerCase().includes(q)
    );
  }, [rows, search]);

  const sorted = useMemo(() => {
    if (!sortField || !sortDirection) return filtered;
    const copy = [...filtered];
    copy.sort((a, b) => {
      let av: string | number = "";
      let bv: string | number = "";
      if (sortField === "name") { av = a.site.name; bv = b.site.name; }
      else if (sortField === "url") { av = a.site.url; bv = b.site.url; }
      else if (sortField === "total") { av = a.stats.total; bv = b.stats.total; }
      else { av = a.stats.unused; bv = b.stats.unused; }

      if (typeof av === "string" && typeof bv === "string") {
        return sortDirection === "asc" ? av.localeCompare(bv) : bv.localeCompare(av);
      }
      if (typeof av === "number" && typeof bv === "number") {
        return sortDirection === "asc" ? av - bv : bv - av;
      }
      return 0;
    });
    return copy;
  }, [filtered, sortField, sortDirection]);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : prev === "desc" ? null : "asc"));
      if (sortDirection === null) setSortField(null);
    } else {
      setSortField(field);
      setSortDirection("asc");
    }
  };

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3">
        <div className="relative w-full sm:w-96">
          <Input
            placeholder="Search by site name or URL..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-3"
          />
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => onRefresh && onRefresh()}
            disabled={isLoading || !!isRefreshing}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>

            {/*
             <Button onClick={() => onOpenImportDialog && onOpenImportDialog()} className="bg-purple-600 hover:bg-purple-700 text-white">
                <Upload className="h-4 w-4 mr-2" />
                Import Topics
             </Button>
            */}
        </div>
      </div>

      <div className="w-full overflow-x-auto rounded-lg border">
        <Table className="min-w-[800px] text-sm">
          <TableHeader>
            <TableRow className="[&>th]:py-2">
              <TableHead>
                <button className="hover:text-foreground" onClick={() => toggleSort("name")}>Site</button>
              </TableHead>
              <TableHead>
                <button className="hover:text-foreground" onClick={() => toggleSort("url")}>URL</button>
              </TableHead>
              <TableHead>
                <button className="hover:text-foreground" onClick={() => toggleSort("total")}>Total Topics</button>
              </TableHead>
              <TableHead>
                <button className="hover:text-foreground" onClick={() => toggleSort("unused")}>Unused</button>
              </TableHead>
              <TableHead className="w-[70px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && sites.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-12">
                  <RefreshCw className="h-8 w-8 animate-spin mx-auto mb-2 text-muted-foreground" />
                  <p className="text-muted-foreground">Loading sites...</p>
                </TableCell>
              </TableRow>
            ) : sorted.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center py-12">
                  <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground/50" />
                  <h3 className="font-semibold mb-1">No sites found</h3>
                  <p className="text-sm text-muted-foreground mb-4">Add sites first to manage topics</p>
                </TableCell>
              </TableRow>
            ) : (
              sorted.map(({ site, stats }) => (
                <TableRow key={site.id} className="cursor-pointer [&>td]:py-2" onClick={() => onManage(site.id)}>
                  <TableCell className="font-medium">{site.name}</TableCell>
                  <TableCell className="text-muted-foreground">{site.url}</TableCell>
                  <TableCell>{stats.total}</TableCell>
                  <TableCell className={stats.unused === 0 ? "text-red-600" : ""}>{stats.unused}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon">
                          <MoreVertical className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => onImport(site.id)}>
                          <Upload className="h-4 w-4 mr-2 text-purple-600" />
                          Import Topics
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onSyncCategories(site.id)}>
                          <RefreshIcon className="h-4 w-4 mr-2" />
                          Sync Categories
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

export default TopicsSitesTable;
