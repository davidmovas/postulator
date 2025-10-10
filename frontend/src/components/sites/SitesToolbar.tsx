"use client";

import React from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Activity, Plus, RefreshCw, Search } from "lucide-react";

export interface SitesToolbarProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  isLoading?: boolean;
  totalSites: number;
  onCreate: () => void;
  onRefresh: () => void | Promise<void>;
  isRefreshing?: boolean;
  onHealthCheckAll: () => void | Promise<void>;
}

export function SitesToolbar({
  searchQuery,
  onSearchChange,
  isLoading = false,
  totalSites,
  onCreate,
  onRefresh,
  isRefreshing = false,
  onHealthCheckAll,
}: SitesToolbarProps) {
  return (
    <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
      {/* Search */}
      <div className="relative w-full sm:w-96">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search by name, URL or username..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Actions */}
      <div className="flex gap-2 w-full sm:w-auto">
        <Button
          variant="outline"
          onClick={() => onHealthCheckAll()}
          disabled={isLoading || totalSites === 0}
          className="flex-1 sm:flex-initial"
        >
          <Activity className="h-4 w-4 mr-2" />
          Check All
        </Button>
        <Button
          variant="outline"
          onClick={() => onRefresh()}
          disabled={isRefreshing}
          className="flex-1 sm:flex-initial"
        >
          <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
        <Button onClick={onCreate} className="flex-1 sm:flex-initial">
          <Plus className="h-4 w-4 mr-2" />
          Add Site
        </Button>
      </div>
    </div>
  );
}

export default SitesToolbar;
