import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useRun, useRunItems, useRunProgress } from "../../data/hooks/runs.js";
import { ChevronRightIcon, EmptyState, HistoryIcon, SkeletonRows } from "../../ui/index.js";
import { itemView, statsView } from "./authority.js";
import { useNow } from "./clock.js";
import { RunEventFeed } from "./events.js";
import { ItemStatusTabs, RunItemTable } from "./items.js";
import { retryNotices, stepTimeline } from "./log-view.js";
import { pathOf, usePageIndex } from "./page-index.js";
import { itemSearchOf, readItemStatus } from "./params.js";
import { recipeSteps } from "./recipe.js";
import { ReviewDrawer } from "./review.js";
import { StartRunDialog } from "./start.js";
import { RunSummary } from "./summary.js";

const tickMs = 1000;

export function RunDetailScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const runId = params.runId ?? "";
    const itemId = params.itemId ?? null;
    const status = readItemStatus(searchParams);
    const search = itemSearchOf(status);

    const row = useRun(runId === "" ? null : runId);
    const progress = useRunProgress(runId);
    const listed = useRunItems(runId, status === "" ? undefined : status);
    const index = usePageIndex(siteId);
    const [rerunning, setRerunning] = useState<string | null>(null);

    const items = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const retries = useMemo(() => retryNotices(progress.events.events), [progress.events]);
    const steps = useMemo(() => recipeSteps(progress.run), [progress.run]);
    const stats = statsView(progress.stats);
    const now = useNow(tickMs, !progress.terminal);

    const selected = itemId === null ? undefined : items.find((candidate) => candidate.id === itemId);
    const { hasNextPage, isFetchingNextPage, fetchNextPage } = listed;

    useEffect(() => {
        if (itemId !== null && selected === undefined && hasNextPage && !isFetchingNextPage) {
            void fetchNextPage();
        }
    }, [itemId, selected, hasNextPage, isFetchingNextPage, fetchNextPage]);

    const paths = useMemo(() => {
        const table = new Map<string, string>();
        for (const entry of items) {
            table.set(entry.id, pathOf(index, entry.targetId));
        }
        return table;
    }, [items, index]);

    const timeline = useMemo(
        () => (itemId === null ? [] : stepTimeline(progress.events.events, itemId)),
        [progress.events, itemId],
    );

    const view = selected === undefined ? null : itemView(selected, progress.progress.get(selected.id), progress.terminal);

    if (progress.run === undefined) {
        return row.isPending ? (
            <div className="flex h-full min-h-0 flex-col p-3">
                <SkeletonRows rows={8} label={copy.runs.loading} />
            </div>
        ) : (
            <div className="flex h-full items-start justify-center p-6">
                <EmptyState icon={HistoryIcon} title={copy.runs.notFound} body={copy.empty.runs} />
            </div>
        );
    }

    const open = (nextItemId: string): void => {
        void navigate(`/s/${siteId}/runs/${runId}/items/${nextItemId}${search}`);
    };

    return (
        <div className="flex h-full min-h-0 flex-col">
            <nav className="flex h-6 shrink-0 items-center gap-1 border-b border-hairline px-3 text-2xs text-ink-faint">
                <Link to={`/s/${siteId}/runs`} className="hover:text-ink">
                    {copy.runs.backToRuns}
                </Link>
                <ChevronRightIcon size={12} />
                <span className="truncate font-mono">{runId}</span>
            </nav>

            <RunSummary
                run={progress.run}
                stats={stats}
                events={progress.events}
                gap={progress.gap}
                terminal={progress.terminal}
                steps={steps}
            />

            <div className="flex min-h-0 flex-1">
                <div className="flex min-w-0 flex-1 flex-col">
                    <div className="flex h-7 shrink-0 items-center justify-between gap-2 border-b border-hairline px-3">
                        <ItemStatusTabs
                            value={status}
                            onChange={(next) => {
                                setSearchParams(next === "" ? new URLSearchParams() : new URLSearchParams({ item: next }), {
                                    replace: true,
                                });
                            }}
                        />
                        <span className="shrink-0 font-mono text-2xs text-ink-faint">
                            {copy.runs.detail.targets(progress.run.targets?.length ?? 0)}
                        </span>
                    </div>
                    {listed.isPending ? (
                        <div className="min-h-0 flex-1 overflow-auto p-3">
                            <SkeletonRows rows={10} label={copy.runs.loadingItems} />
                        </div>
                    ) : (
                        <RunItemTable
                            runId={runId}
                            items={items}
                            progress={progress.progress}
                            retries={retries}
                            terminal={progress.terminal}
                            steps={steps}
                            index={index}
                            selectedId={itemId}
                            now={now}
                            narrowed={status !== ""}
                            hasMore={listed.hasNextPage}
                            loadingMore={listed.isFetchingNextPage}
                            onLoadMore={() => {
                                void listed.fetchNextPage();
                            }}
                            onOpen={open}
                        />
                    )}
                </div>
                <RunEventFeed
                    events={progress.events}
                    gap={progress.gap}
                    itemId={itemId}
                    paths={paths}
                />
            </div>

            {itemId === null ? null : (
                <ReviewDrawer
                    view={view}
                    steps={steps}
                    timeline={timeline}
                    retry={retries.get(itemId)}
                    path={selected === undefined ? itemId : pathOf(index, selected.targetId)}
                    now={now}
                    missing={selected === undefined && !listed.hasNextPage}
                    narrowed={status !== ""}
                    onClearFilter={() => {
                        setSearchParams(new URLSearchParams(), { replace: true });
                    }}
                    onClose={() => {
                        void navigate(`/s/${siteId}/runs/${runId}${search}`);
                    }}
                    onRerun={(pageId) => {
                        setRerunning(pageId);
                    }}
                />
            )}

            <StartRunDialog
                open={rerunning !== null}
                onOpenChange={(next) => {
                    if (!next) {
                        setRerunning(null);
                    }
                }}
                siteId={siteId}
                preselect={rerunning === null ? undefined : [rerunning]}
                onStarted={(nextRunId) => {
                    setRerunning(null);
                    void navigate(`/s/${siteId}/runs/${nextRunId}`);
                }}
            />
        </div>
    );
}
