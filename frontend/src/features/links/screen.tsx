import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useGraph } from "../../data/hooks/graph.js";
import { useLinkAudit } from "../../data/hooks/reports.js";
import { Banner, EmptyState, PublicIcon, SkeletonRows } from "../../ui/index.js";
import { buildGraphIndex } from "../graph/model/index.js";
import { StartRunDialog } from "../runs/start.js";
import { LinkFilters } from "./filters.js";
import { AuditMeters } from "./meters.js";
import { rows as auditRows, showCounts } from "./model/audit.js";
import { defaultQuery, narrowed, readQuery, searchOf, writeQuery } from "./model/params.js";
import type { LinksQuery } from "./model/params.js";
import { AuditPanel } from "./panel.js";
import { AuditTable } from "./table.js";

export const relinkCap = 500;
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
    const relinkTargets = useMemo(() => rows.filter((row) => row.missing > 0 && row.skipReason === "").map((row) => row.pageId), [rows]);
    const selected = pageId === null ? undefined : pages.find((row) => row.pageId === pageId);

    const change = (next: LinksQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const open = (id: string | null): void => {
        void navigate(id === null ? `/s/${siteId}/links${search}` : `/s/${siteId}/links/${id}${search}`, { replace: true });
    };

    if (siteId === "") {
        return (
            <div className="flex h-full items-start justify-center p-6">
                <EmptyState icon={PublicIcon} title={copy.shell.noSiteSelected} body={copy.empty.sites} />
            </div>
        );
    }

    const failure = audit.error === null ? null : react(audit.error);

    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex h-8 shrink-0 items-center justify-between gap-3 border-b border-hairline px-3">
                <h1 className="text-sm font-semibold text-ink">{copy.links.title}</h1>
                <p className="hidden truncate text-2xs text-ink-faint lg:block">{copy.links.subtitle}</p>
            </header>
            {audit.data === undefined ? null : (
                <AuditMeters
                    totals={audit.data.totals}
                    policy={audit.data.policy}
                    relinkCount={Math.min(relinkTargets.length, relinkCap)}
                    relinkCapped={relinkTargets.length > relinkCap}
                    relinkCap={relinkCap}
                    onRelink={() => {
                        setRelinking(relinkTargets.slice(0, relinkCap));
                    }}
                />
            )}
            <div className="flex min-h-0 flex-1">
                <LinkFilters query={query} counts={counts} statusCounts={statusCounts} index={index} onChange={change} />
                <div className="flex min-w-0 flex-1 flex-col">
                    {audit.isPending ? (
                        <div className="p-3">
                            <SkeletonRows rows={10} label={copy.links.loading} />
                        </div>
                    ) : failure !== null && failure.kind !== "silent" && failure.kind !== "unlock" ? (
                        <div className="p-3">
                            <Banner tone="danger" title={failure.message} />
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
                </div>
                {selected === undefined ? null : (
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
                )}
            </div>
            <StartRunDialog
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
        </div>
    );
}
