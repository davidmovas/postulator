import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useGraph } from "../../data/hooks/graph.js";
import { useLinkAudit } from "../../data/hooks/reports.js";
import { Banner, Button, CountBadge, PlayCircleIcon, Screen, SkeletonRows } from "../../ui/index.js";
import { buildGraphIndex } from "../graph/model/index.js";
import { StartRunDrawer } from "../runs/start.js";
import { LinkFilters } from "./filters.js";
import { AuditStrip } from "./meters.js";
import { rows as auditRows, showCounts } from "./model/audit.js";
import { defaultQuery, narrowed, readQuery, searchOf, writeQuery } from "./model/params.js";
import type { LinksQuery } from "./model/params.js";
import { AuditPanel } from "./panel.js";
import { relinkCap, relinkSelection } from "./model/relink.js";
import { AuditTable } from "./table.js";

const relinkKind = "relink";

export function LinksScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const pageId = params.pageId ?? null;
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const search = searchOf(query);
    const audit = useLinkAudit(siteId === "" ? null : siteId);
    const graph = useGraph(siteId === "" ? null : siteId);
    const [relinking, setRelinking] = useState<readonly string[] | null>(null);

    const index = useMemo(() => (graph.data === undefined ? null : buildGraphIndex(graph.data.entities ?? [], graph.data.edges ?? [])), [graph.data]);
    const pages = useMemo(() => audit.data?.pages ?? [], [audit.data]);
    const rows = useMemo(() => auditRows(pages, query, index), [pages, query, index]);
    const counts = useMemo(() => (audit.data === undefined ? null : showCounts(pages)), [audit.data, pages]);
    const statusCounts = useMemo(() => {
        const held = new Map<string, number>();
        for (const row of pages) {
            held.set(row.status, (held.get(row.status) ?? 0) + 1);
        }
        return held;
    }, [pages]);
    const owed = useMemo(() => relinkSelection(rows), [rows]);
    const selected = pageId === null ? undefined : pages.find((row) => row.pageId === pageId);
    const relinkCount = owed.pageIds.length;

    const change = (next: LinksQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const open = (id: string | null): void => {
        void navigate(id === null ? `/s/${siteId}/links${search}` : `/s/${siteId}/links/${id}${search}`, { replace: true });
    };

    const failure = audit.error === null ? null : react(audit.error);

    return (
        <Screen
            title={copy.links.title}
            badge={audit.data === undefined ? undefined : <CountBadge tone="muted" count={audit.data.totals.audited} />}
            actions={
                <Button
                    variant="primary"
                    icon={PlayCircleIcon}
                    disabled={relinkCount === 0}
                    title={
                        relinkCount === 0
                            ? copy.links.relink.noneTitle
                            : `${owed.capped ? copy.links.relink.capped(relinkCap) : copy.links.relink.title}. ${copy.links.relink.does}`
                    }
                    onClick={() => {
                        setRelinking(owed.pageIds);
                    }}
                >
                    {relinkCount === 0 ? copy.links.relink.none : copy.links.relink.start(relinkCount)}
                </Button>
            }
            variant="split"
            left={<LinkFilters query={query} counts={counts} statusCounts={statusCounts} index={index} onChange={change} />}
            right={
                selected === undefined ? undefined : (
                    <AuditPanel
                        key={selected.pageId}
                        siteId={siteId}
                        row={selected}
                        onClose={() => {
                            open(null);
                        }}
                        onRelink={(id) => {
                            setRelinking([id]);
                        }}
                    />
                )
            }
        >
            {audit.data === undefined ? null : <AuditStrip totals={audit.data.totals} policy={audit.data.policy} />}
            {audit.isPending ? (
                <div className="p-4">
                    <SkeletonRows rows={10} label={copy.links.loading} />
                </div>
            ) : failure !== null && failure.kind !== "silent" && failure.kind !== "unlock" ? (
                <div className="p-4">
                    <Banner
                        tone="danger"
                        title={failure.message}
                        actions={
                            <Button size="sm" variant="secondary" onClick={() => void audit.refetch()}>
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : (
                <AuditTable
                    siteId={siteId}
                    rows={rows}
                    selectedId={pageId}
                    narrowed={narrowed(query)}
                    onOpen={open}
                    onReset={() => {
                        change({ ...defaultQuery, sort: query.sort });
                    }}
                    onOpenGraph={() => {
                        void navigate(`/s/${siteId}/graph`);
                    }}
                />
            )}
            <StartRunDrawer
                open={relinking !== null}
                onOpenChange={(next) => {
                    if (!next) {
                        setRelinking(null);
                    }
                }}
                siteId={siteId}
                preselect={relinking ?? []}
                initialKind={relinkKind}
                onStarted={(runId) => {
                    setRelinking(null);
                    void navigate(`/s/${siteId}/runs/${runId}`);
                }}
            />
        </Screen>
    );
}
