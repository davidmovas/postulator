import type { ReactElement } from "react";
import { useMemo } from "react";
import { Link, useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useGraph } from "../../data/hooks/graph.js";
import { usePageTree } from "../../data/hooks/pages.js";
import { useLinkAudit, useSiteOverview } from "../../data/hooks/reports.js";
import { useSite } from "../../data/hooks/sites.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import { Banner, Button, EmptyState, MonitoringIcon, SkeletonRows } from "../../ui/index.js";
import { auditCard, bars, coverageRows, driftRows, flattenTree, tiles } from "./model/site.js";
import { Histogram, LinkAuditPanel } from "./panels.js";
import { CoverageTable, DriftTable } from "./tables.js";
import { SiteTileRow } from "./tiles.js";

export interface SiteTabProps {
    siteId: string;
}

export function SiteTab({ siteId }: SiteTabProps): ReactElement {
    const navigate = useNavigate();
    const key = siteId === "" ? null : siteId;
    const overview = useSiteOverview(key);
    const audit = useLinkAudit(key);
    const tree = usePageTree(key);
    const graph = useGraph(key);
    const site = useSite(key);
    const sync = useSyncSite();

    const pages = useMemo(() => flattenTree(tree.data?.roots ?? null), [tree.data]);
    const drifted = useMemo(() => driftRows(pages), [pages]);
    const audits = useMemo(() => audit.data?.pages ?? [], [audit.data]);
    const coverage = useMemo(
        () => coverageRows(graph.data?.entities ?? [], pages, audits),
        [graph.data, pages, audits],
    );

    if (overview.isPending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={10} label={copy.reports.loading} />
            </div>
        );
    }

    const failure = overview.error === null ? null : react(overview.error);
    if (failure !== null && failure.kind !== "silent" && failure.kind !== "unlock") {
        return (
            <div className="p-4">
                <Banner
                    tone="danger"
                    title={failure.message}
                    actions={
                        <Button size="sm" variant="secondary" onClick={() => void overview.refetch()}>
                            {copy.app.retry}
                        </Button>
                    }
                />
            </div>
        );
    }

    const held = overview.data;
    if (held === undefined) {
        return (
            <div className="p-4">
                <SkeletonRows rows={10} label={copy.reports.loading} />
            </div>
        );
    }

    if (held.pages.total === 0 && held.entities.total === 0) {
        return (
            <div className="p-4">
                <EmptyState
                    icon={MonitoringIcon}
                    title={copy.reports.empty.site}
                    actions={
                        <>
                            <Button
                                variant="primary"
                                onClick={() => {
                                    void navigate(`/s/${siteId}/runs?action=new`);
                                }}
                            >
                                {copy.reports.empty.startRun}
                            </Button>
                            <Link
                                to={`/s/${siteId}/graph`}
                                className="flex h-7 items-center rounded-md border border-hairline px-2.5 text-sm text-ink hover:bg-raised"
                            >
                                {copy.reports.empty.graph}
                            </Link>
                        </>
                    }
                />
            </div>
        );
    }

    return (
        <div className="min-h-0 flex-1 space-y-4 overflow-auto p-4">
            <SiteTileRow tiles={tiles(held.entities, held.pages, held.depth)} drifted={drifted.length} />
            <div className="flex flex-wrap gap-4">
                <Histogram bars={bars(held.depth)} />
                {audit.data === undefined ? null : (
                    <LinkAuditPanel card={auditCard(audits, audit.data.totals)} siteId={siteId} />
                )}
            </div>
            <CoverageTable siteId={siteId} rows={coverage} />
            <DriftTable
                baseUrl={site.data?.site.baseUrl ?? ""}
                rows={drifted}
                syncing={sync.isPending}
                onResync={() => {
                    sync.mutate(
                        { siteId },
                        {
                            onSuccess: (answered) => {
                                void navigate(`/s/${siteId}/runs/${answered.runId}`);
                            },
                        },
                    );
                }}
            />
        </div>
    );
}
