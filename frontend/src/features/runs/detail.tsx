import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useRun, useRunItems, useRunProgress } from "../../data/hooks/runs.js";
import {
    ChevronRightIcon,
    EmptyState,
    HistoryIcon,
    Screen,
    SkeletonRows,
    StatusBadge,
    Toolbar,
} from "../../ui/index.js";
import { itemView, runView, statsView } from "./authority.js";
import { useNow } from "./clock.js";
import { RunControls } from "./controls.js";
import { RunEventFeed } from "./events.js";
import { ItemStatusTabs, RunItemTable } from "./items.js";
import {
    kindLabel,
    pauseReasonShort,
    pauseReasonTone,
    statusIcon,
    statusLabel,
    statusTone,
} from "./labels.js";
import { retryNotices, stepTimeline } from "./log-view.js";
import { RunNotices } from "./notices.js";
import { pathOf, usePageIndex } from "./page-index.js";
import { itemSearchOf, readItemStatus } from "./params.js";
import { RunProgress } from "./progress.js";
import { recipeSteps } from "./recipe.js";
import { ReviewDrawer } from "./review.js";
import { StartRunDrawer } from "./start.js";

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

    const run = progress.run;

    if (run === undefined) {
        return (
            <Screen title={copy.runs.title}>
                {row.isPending ? (
                    <SkeletonRows rows={8} label={copy.runs.loading} />
                ) : (
                    <div className="flex h-full items-start justify-center">
                        <EmptyState icon={HistoryIcon} title={copy.runs.notFound} body={copy.empty.runs} />
                    </div>
                )}
            </Screen>
        );
    }

    const view = runView(run);
    const open = (nextItemId: string): void => {
        void navigate(`/s/${siteId}/runs/${runId}/items/${nextItemId}${search}`);
    };

    return (
        <Screen
            title={copy.runs.detail.header(kindLabel(run.kind))}
            badge={
                <span className="flex shrink-0 items-center gap-1.5">
                    <StatusBadge tone={statusTone(run.status)} icon={statusIcon(run.status)}>
                        {statusLabel(run.status)}
                    </StatusBadge>
                    {view.paused && run.pauseReason !== "" ? (
                        <StatusBadge tone={pauseReasonTone(run.pauseReason)} dot={false}>
                            {pauseReasonShort(run.pauseReason)}
                        </StatusBadge>
                    ) : null}
                </span>
            }
            variant="split"
            actions={<RunControls run={run} />}
            toolbar={
                <Toolbar label={copy.runs.detail.header(kindLabel(run.kind))}>
                    <Link
                        to={`/s/${siteId}/runs`}
                        className="flex shrink-0 items-center gap-0.5 text-2xs text-ink-faint hover:text-ink"
                    >
                        {copy.runs.backToRuns}
                        <ChevronRightIcon size={12} />
                    </Link>
                    <ItemStatusTabs
                        value={status}
                        onChange={(next) => {
                            setSearchParams(
                                next === "" ? new URLSearchParams() : new URLSearchParams({ item: next }),
                                { replace: true },
                            );
                        }}
                    />
                    <span className="ml-auto shrink-0 font-mono text-2xs text-ink-faint" title={run.id}>
                        {copy.runs.detail.targets(run.targets?.length ?? 0)}
                    </span>
                </Toolbar>
            }
            right={
                <RunEventFeed
                    events={progress.events}
                    gap={progress.gap}
                    itemId={itemId}
                    paths={paths}
                />
            }
        >
            <RunProgress run={run} view={view} stats={stats} terminal={progress.terminal} now={now} />
            <RunNotices run={run} view={view} events={progress.events} gap={progress.gap} />

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

            {itemId === null ? null : (
                <ReviewDrawer
                    view={
                        selected === undefined
                            ? null
                            : itemView(selected, progress.progress.get(selected.id), progress.terminal)
                    }
                    steps={steps}
                    timeline={timeline}
                    retry={retries.get(itemId)}
                    path={selected === undefined ? itemId : pathOf(index, selected.targetId)}
                    now={now}
                    missing={selected === undefined && !listed.hasNextPage}
                    narrowed={status !== ""}
                    page={selected === undefined ? undefined : index.byId.get(selected.targetId)}
                    siblings={items}
                    onMove={open}
                    onOpenPage={(pageId) => {
                        void navigate(`/s/${siteId}/pages/${pageId}`);
                    }}
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

            <StartRunDrawer
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
        </Screen>
    );
}
