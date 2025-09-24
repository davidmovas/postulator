"use client";
import * as React from "react";
import TopicsTable from "@/components/topics/topics-table";
import type { Topic } from "@/types/topic";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import AssignPanel from "@/components/topics/assign-panel";

export default function TopicsPanel() {
  const [page, setPage] = React.useState<number>(1);
  const pageSize = 100;
  const [topics, setTopics] = React.useState<Topic[]>([]);
  const [total, setTotal] = React.useState<number>(0);
  const [loading, setLoading] = React.useState<boolean>(false);
  const [error, setError] = React.useState<string | undefined>(undefined);

  async function load() {
    try {
      setLoading(true);
      setError(undefined);
      const svc = await import("@/services/topics");
      const { items, total } = await svc.getTopics(page, pageSize);
      setTopics(items);
      setTotal(total);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load topics");
      setTopics([]);
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
        <h1 className="text-xl font-semibold">Topics</h1>
        <p className="text-sm text-muted-foreground">Create, edit, and manage your content topics; assign them to sites; review stats & history.</p>
      </div>
      {error && <div className="mb-3 text-sm text-destructive">{error}</div>}

      <Tabs defaultValue="topics" className="w-full">
        <TabsList className="mb-3">
          <TabsTrigger value="topics">Topics</TabsTrigger>
          <TabsTrigger value="assign">Assign</TabsTrigger>
          <TabsTrigger value="stats">Stats & History</TabsTrigger>
        </TabsList>

        <TabsContent value="topics" className="mt-0">
          <TopicsTable
            topics={topics}
            page={page}
            pageSize={pageSize}
            total={total}
            onPageChange={setPage}
            onRefresh={refresh}
            onCreate={async (v) => {
              const svc = await import("@/services/topics");
              await svc.createTopic(v);
            }}
            onUpdate={async (id, v) => {
              const svc = await import("@/services/topics");
              await svc.updateTopic(id, v);
            }}
            onDelete={async (id) => {
              const svc = await import("@/services/topics");
              await svc.deleteTopic(id);
            }}
            // onToggleActive and onBulkToggle disabled - setTopicActive method not available
            // onToggleActive={async (id, active) => {
            //   const svc = await import("@/services/topics");
            //   await svc.setTopicActive(id, active);
            // }}
            // onBulkToggle={async (ids, active) => {
            //   const svc = await import("@/services/topics");
            //   await Promise.all(ids.map((id) => svc.setTopicActive(id, active)));
            // }}
            onBulkDelete={async (ids) => {
              const svc = await import("@/services/topics");
              await Promise.all(ids.map((id) => svc.deleteTopic(id)));
            }}
            onMutateTopics={(updater) => setTopics((prev) => updater(prev))}
          />
          {loading && <div className="mt-3 text-sm text-muted-foreground">Loading...</div>}
        </TabsContent>

        <TabsContent value="assign" className="mt-0">
          <AssignPanel />
        </TabsContent>

        <TabsContent value="stats" className="mt-0">
            {/*<StatsPanel />*/}
        </TabsContent>
      </Tabs>
    </div>
  );
}
