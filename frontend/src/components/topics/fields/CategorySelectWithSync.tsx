"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { RefreshCw } from "lucide-react";
import type { Category } from "@/services/site";
import { getSiteCategories, syncCategories } from "@/services/site";
import { useErrorHandling } from "@/lib/error-handling";

export interface CategorySelectWithSyncProps {
  siteId: number;
  selectedCategory?: Category | null;
  onChange: (value: Category | null) => void;
  label?: string;
  className?: string;
  selectId?: string;
}

export function CategorySelectWithSync({
  siteId,
  selectedCategory,
  onChange,
  label = "Category",
  className,
  selectId = "category",
}: CategorySelectWithSyncProps) {
  const { withErrorHandling } = useErrorHandling();

  const [categories, setCategories] = useState<Category[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isSyncing, setIsSyncing] = useState<boolean>(false);

  const hasCategories = useMemo(() => Array.isArray(categories) && categories.length > 0, [categories]);

  // Load categories on mount and when siteId changes
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      if (!siteId) return;
      setIsLoading(true);
      try {
        const list = await getSiteCategories(siteId);
        if (cancelled) return;
        setCategories(list);
        // If nothing selected yet, pick first or null and notify parent
        if (!selectedCategory) {
          const next = list.length > 0 ? list[0] : null;
          onChange(next);
        } else {
          // Keep selection if still present
          const keep = list.find((c) => c.id === selectedCategory.id) || null;
          if (keep?.id !== selectedCategory.id) {
            onChange(keep);
          }
        }
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [siteId]);

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value;
    if (val === "") {
      onChange(null);
      return;
    }
    const found = categories.find((c) => String(c.id) === val) || null;
    onChange(found);
  };

  const handleSync = async () => {
    if (!siteId) return;
    setIsSyncing(true);
    await withErrorHandling(async () => {
      await syncCategories(siteId);
      const list = await getSiteCategories(siteId);
      setCategories(list);
      // Try to keep the selection if possible
      if (selectedCategory) {
        const still = list.find((c) => c.id === selectedCategory.id) || null;
        onChange(still);
      } else {
        onChange(list.length > 0 ? list[0] : null);
      }
    }, { successMessage: "Categories synchronized", showSuccess: true });
    setIsSyncing(false);
  };

  const disabled = isLoading || isSyncing || !hasCategories;

  return (
    <div className={`flex items-end gap-3 ${className ?? ""}`}>
      <div className="flex-1 space-y-2">
        <Label htmlFor={selectId}>{label}</Label>
        <select
          id={selectId}
          value={selectedCategory ? String(selectedCategory.id) : ""}
          onChange={handleChange}
          disabled={disabled}
          className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {!hasCategories ? (
            <option value="">No categories available. Please Sync.</option>
          ) : (
            categories.map((c) => (
              <option key={c.id} value={String(c.id)}>
                {c.name}
              </option>
            ))
          )}
        </select>
      </div>
      <Button
        variant="secondary"
        onClick={handleSync}
        disabled={isSyncing}
      >
        <RefreshCw className="h-4 w-4 mr-2" />
        {isSyncing ? "Syncing..." : "Sync"}
      </Button>
    </div>
  );
}

export default CategorySelectWithSync;
