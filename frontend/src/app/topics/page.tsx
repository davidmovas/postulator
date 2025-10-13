"use client";

import React, { useEffect, useMemo, useState } from "react";
import { listSites, Site, syncCategories } from "@/services/site";
import { getTopicsBySite } from "@/services/topic";
import { countUnusedTopics } from "@/services/topic";
import { useErrorHandling } from "@/lib/error-handling";
import { TopicsSitesTable } from "@/components/topics/TopicsSitesTable";
import { ImportTopicsDialog } from "@/components/topics/ImportTopicsDialog";
import { SiteTopicsManager } from "@/components/topics/SiteTopicsManager";

export default function TopicsPage() {
  const { withErrorHandling } = useErrorHandling();

  const [sites, setSites] = useState<Site[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [stats, setStats] = useState<Record<number, { siteId: number; total: number; unused: number }>>({});

  const [importForSiteId, setImportForSiteId] = useState<number | null>(null);
  const [selectedSiteId, setSelectedSiteId] = useState<number | null>(null);

  const loadSites = async () => {
    setIsLoading(true);
    try {
      const data = await listSites();
      setSites(data);
      return data;
    } finally {
      setIsLoading(false);
    }
  };

  const loadStatsForSites = async (items: Site[]) => {
    const entries = await Promise.all(
      items.map(async (s) => {
        try {
          // total topics assigned
          const t = await getTopicsBySite(s.id);
          // unused count via API
          const unused = await countUnusedTopics(s.id);
          return [s.id, { siteId: s.id, total: t.length, unused }] as const;
        } catch (e) {
          return [s.id, { siteId: s.id, total: 0, unused: 0 }] as const;
        }
      })
    );
    setStats(Object.fromEntries(entries));
  };

  useEffect(() => {
    (async () => {
      const data = await loadSites();
      await loadStatsForSites(data || []);
    })();
  }, []);

  const handleManage = (siteId: number) => {
    setSelectedSiteId(siteId);
  };

  const handleImport = (siteId: number) => {
    setImportForSiteId(siteId);
  };

  const handleSyncCategories = async (siteId: number) => {
    await withErrorHandling(async () => {
      await syncCategories(siteId);
      // No categories to display yet, but provide feedback
    }, { successMessage: "Categories synchronized", showSuccess: true });
  };

  return (
    <div className="p-4 md:p-6 lg:p-8 space-y-6">
      {selectedSiteId ? (
        <SiteTopicsManager siteId={selectedSiteId} onBack={() => setSelectedSiteId(null)} />
      ) : (
        <>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Topics</h1>
            <p className="mt-2 text-muted-foreground">Manage topics by site: import, review, and organize assignments.</p>
          </div>

          <TopicsSitesTable
            sites={sites}
            isLoading={isLoading}
            stats={stats}
            onManage={handleManage}
            onImport={handleImport}
            onSyncCategories={handleSyncCategories}
          />

          <ImportTopicsDialog
            open={importForSiteId !== null}
            onOpenChange={(o) => { if (!o) setImportForSiteId(null); }}
            siteId={importForSiteId}
            onImported={async () => {
              const data = await loadSites();
              await loadStatsForSites(data || []);
            }}
          />
        </>
      )}
    </div>
  );
}
