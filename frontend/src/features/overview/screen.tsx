import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";

import { copy } from "../../copy/index.js";
import { useOpenExternal } from "../../data/hooks/browser.js";
import { useEdges } from "../../data/hooks/graph.js";
import { usePageTree } from "../../data/hooks/pages.js";
import { useSiteOverview } from "../../data/hooks/reports.js";
import { useRuns } from "../../data/hooks/runs.js";
import { useSite } from "../../data/hooks/sites.js";
import { usePluginState } from "../../data/hooks/sync.js";
import type { RunSort } from "../../data/sorts.js";
import { maxLimit } from "../../lib/paging.js";
import type { MenuEntry } from "../../ui/index.js";
import {
    Banner,
    Button,
    EmptyState,
    IconButton,
    Menu,
    MonitoringIcon,
    MoreHorizIcon,
    PlayArrowIcon,
    PublicIcon,
    Screen,
    Skeleton,
    SkeletonRows,
    TravelExploreIcon,
} from "../../ui/index.js";
import { SyncButton } from "../sync/index.js";
import { ReadinessChecklist } from "./checklist.js";
import { driftCount, edgeTile, entityTile, histogram, isEmptySite, pageTile, runSummary } from "./model/overview.js";
import { ObservationPanel } from "./observation.js";
import { CoveragePanel, DepthPanel, DriftPanel, RunPanel, TopEntities } from "./panels.js";
import { Tiles } from "./tiles.js";
import { useReadiness } from "./readiness.js";
import { SetupBanner } from "./setup-banner.js";

const newestFirst: RunSort = { field: "createdAt", desc: true };

function Loading(): ReactElement {
    return (
        <div className="flex min-h-0 flex-1 gap-4 p-4">
            <div className="flex min-w-0 flex-1 flex-col gap-3">
                <Skeleton height={56} />
                <div className="grid grid-cols-3 gap-3">
                    <Skeleton height={128} />
                    <Skeleton height={128} />
                    <Skeleton height={128} />
                </div>
                <Skeleton height={180} />
            </div>
            <div className="w-80 shrink-0">
                <SkeletonRows rows={10} height={22} label={copy.overview.loading} />
            </div>
        </div>
    );
}

export function OverviewScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const siteId = params.siteId ?? "";
    const site = useSite(siteId);
    const overview = useSiteOverview(siteId);
    const plugin = usePluginState(siteId);
    const runs = useRuns({ siteId }, newestFirst, 20);
    const proposed = useEdges({ siteId, status: "proposed" }, maxLimit);
    const tree = usePageTree(siteId);
    const readiness = useReadiness(siteId);
    const open = useOpenExternal();
    const [dismissed, setDismissed] = useState(false);

    const answered = overview.data;
    const runRows = useMemo(() => (runs.data?.pages ?? []).flatMap((page) => page.items), [runs.data]);
    const summary = useMemo(() => runSummary(runRows), [runRows]);
    const proposedPages = proposed.data?.pages ?? [];
    const proposedCount = proposedPages.reduce((carried, page) => carried + page.items.length, 0);
    const proposedCapped = proposedPages.at(-1)?.hasMore === true;
    const drifted = useMemo(() => driftCount(tree.data?.roots ?? []), [tree.data]);

    const baseUrl = site.data?.site.baseUrl ?? "";
    const overflow: MenuEntry[] = [
        {
            key: "open",
            label: copy.app.openExternal,
            icon: PublicIcon,
            disabled: baseUrl === "" || open.isPending,
            onSelect: () => {
                open.mutate({ url: baseUrl });
            },
        },
        {
            key: "reports",
            label: copy.overview.reports,
            icon: MonitoringIcon,
            onSelect: () => {
                void navigate(`/s/${siteId}/reports`);
            },
        },
    ];

    const startRun = (): void => {
        void navigate(`/s/${siteId}/runs?action=new`);
    };

    const rail = (
        <div className="flex flex-col gap-3 p-3">
            <ReadinessChecklist readiness={readiness} />
            <DriftPanel count={drifted} siteId={siteId} />
            {answered === undefined ? null : (
                <CoveragePanel
                    edges={edgeTile(answered.edges, proposedCount, proposedCapped)}
                    onRelink={startRun}
                />
            )}
        </div>
    );

    const setupNeeded = plugin.data?.plugin.installed === false && !dismissed;

    return (
        <Screen
            title={copy.overview.title}
            actions={
                <>
                    <SyncButton siteId={siteId} label={copy.overview.sync} />
                    <Button variant="primary" icon={PlayArrowIcon} onClick={startRun}>
                        {copy.overview.startRun}
                    </Button>
                    <Menu
                        label={copy.app.more}
                        align="end"
                        items={overflow}
                        trigger={<IconButton icon={MoreHorizIcon} label={copy.app.more} variant="ghost" size="sm" />}
                    />
                </>
            }
            variant="split"
            right={rail}
        >
            {overview.isPending ? (
                <Loading />
            ) : overview.isError || answered === undefined ? (
                <div className="p-4">
                    <Banner
                        tone="danger"
                        title={copy.errors.INTERNAL}
                        actions={
                            <Button
                                onClick={() => {
                                    void overview.refetch();
                                }}
                            >
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : (
                <div className="@container flex min-h-0 flex-1 flex-col gap-3 overflow-auto p-4">
                    {setupNeeded ? (
                        <SetupBanner
                            onDismiss={() => {
                                setDismissed(true);
                            }}
                        />
                    ) : null}
                    {isEmptySite(answered.pages, answered.entities) ? (
                        <EmptyState
                            icon={TravelExploreIcon}
                            title={copy.overview.empty}
                            body={copy.overview.emptyBody}
                            actions={<SyncButton siteId={siteId} label={copy.overview.sync} variant="primary" />}
                        />
                    ) : (
                        <>
                            <RunPanel summary={summary} siteId={siteId} onStart={startRun} />
                            <Tiles
                                entities={entityTile(answered.entities)}
                                pages={pageTile(answered.pages)}
                                edges={edgeTile(answered.edges, proposedCount, proposedCapped)}
                            />
                            <div className="grid grid-cols-1 gap-3 @xl:grid-cols-[1.25fr_1fr]">
                                <DepthPanel bars={histogram(answered.depth ?? [])} />
                                <TopEntities entries={answered.top ?? []} siteId={siteId} />
                            </div>
                            <ObservationPanel siteId={siteId} />
                        </>
                    )}
                </div>
            )}
        </Screen>
    );
}
