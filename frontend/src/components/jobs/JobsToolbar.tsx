"use client";

import React from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Plus, RefreshCw, Search } from "lucide-react";

export interface JobsToolbarProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  isLoading?: boolean;
  totalJobs: number;
  onCreate: () => void;
  onRefresh: () => void | Promise<void>;
  isRefreshing?: boolean;
}

export function JobsToolbar({
  searchQuery,
  onSearchChange,
  isLoading = false,
  totalJobs,
  onCreate,
  onRefresh,
  isRefreshing = false,
}: JobsToolbarProps) {
  return (
    <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
      {/* Search */}
      <div className="relative w-full sm:w-96">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search by name, status or model..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Actions */}
      <div className="flex gap-2 w-full sm:w-auto">
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
          New Job
        </Button>
      </div>
    </div>
  );
}

export default JobsToolbar;
