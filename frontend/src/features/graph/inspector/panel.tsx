import type { ReactElement } from "react";
import { useMemo } from "react";

import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import type { PageAudit } from "../../../data/types.js";
import { relativeTime } from "../../../domain/format.js";
import { Button, CloseIcon, cx, DeleteIcon, IconButton, SectionLabel, SmartToyIcon, toneClasses } from "../../../ui/index.js";
import { askAgent } from "../../agent/index.js";
import { severityOf } from "../../links/model/audit.js";
import { entityIcon, formatScore, kindLabel, kindTone } from "../labels.js";
import type { GraphIndex } from "../model/index.js";
import type { Lens } from "../model/lens.js";
import { AnchorsEditor } from "./anchors.js";
import { EntityFields } from "./fields.js";
import { CanonicalPage } from "./page.js";
import { Relations } from "./relations.js";
import { Summary } from "./summary.js";

export interface InspectorProps {
    siteId: string;
    index: GraphIndex;
    selectedId: string | null;
    onSelect: (id: string | null) => void;
    onReveal: (id: string) => void;
    onLens: (lens: Lens) => void;
    onConnect: (id: string) => void;
    onDelete: (id: string) => void;
    onPlanPage: (id: string) => void;
    onRecompute: () => void;
    recomputing: boolean;
    audit: readonly PageAudit[] | null;
}

interface AuditRowsProps {
    siteId: string;
    entityId: string;
    audit: readonly PageAudit[] | null;
}

function AuditRows({ siteId, entityId, audit }: AuditRowsProps): ReactElement {
    const rows = audit === null ? null : audit.filter((row) => row.entityId === entityId);
    return (
        <section className="flex flex-col gap-1">
            <div className="flex items-baseline justify-between">
                <SectionLabel>{copy.graph.inspector.links}</SectionLabel>
                <Link to={`/s/${siteId}/links?entity=${entityId}`} className="text-2xs">
                    {copy.graph.inspector.linksOpen}
                </Link>
            </div>
            {rows === null ? (
                <p className="text-2xs text-ink-faint">{copy.graph.inspector.linksOff}</p>
            ) : rows.length === 0 ? (
                <p className="text-2xs text-ink-faint">{copy.graph.inspector.linksNone}</p>
            ) : (
                <ul className="flex flex-col gap-1">
                    {rows.map((row) => {
                        const severity = severityOf(row);
                        return (
                            <li key={row.pageId} className="flex flex-col gap-0.5 text-xs">
                                <Link to={`/s/${siteId}/links/${row.pageId}`} className="flex min-w-0 items-center gap-2 text-ink-soft hover:text-ink">
                                    <span className={cx("h-1.5 w-1.5 shrink-0 rounded-full", toneClasses[severity].solid)} aria-hidden={true} />
                                    <span className="truncate font-mono">{row.path}</span>
                                </Link>
                                <span className="pl-3.5 text-2xs text-ink-dim">
                                    {row.skipReason !== ""
                                        ? (copy.links.cell.skipped[row.skipReason] ?? row.skipReason)
                                        : [
                                              copy.graph.inspector.linksRow(row.satisfied, row.targets),
                                              row.missing > 0 ? copy.graph.inspector.linksMissing(row.missing, row.missingRequired) : null,
                                              row.blocked > 0 ? copy.graph.inspector.linksBlocked(row.blocked) : null,
                                          ]
                                              .filter((part) => part !== null)
                                              .join(" · ")}
                                </span>
                            </li>
                        );
                    })}
                </ul>
            )}
        </section>
    );
}

export function Inspector({
    siteId,
    index,
    selectedId,
    onSelect,
    onReveal,
    onLens,
    onConnect,
    onDelete,
    onPlanPage,
    onRecompute,
    recomputing,
    audit,
}: InspectorProps): ReactElement {
    const entity = selectedId === null ? undefined : index.byId.get(selectedId);
    const rank = useMemo(() => {
        if (entity === undefined) {
            return 0;
        }
        let ahead = 0;
        for (const other of index.byId.values()) {
            if (other.score > entity.score || (other.score === entity.score && other.name.localeCompare(entity.name) < 0)) {
                ahead += 1;
            }
        }
        return ahead + 1;
    }, [entity, index]);

    if (entity === undefined) {
        return (
            <section aria-label={copy.graph.summary.title} className="flex h-full min-h-0 flex-col bg-panel">
                <Summary
                    index={index}
                    recomputing={recomputing}
                    onLens={onLens}
                    onSelect={onSelect}
                    onRecompute={onRecompute}
                />
            </section>
        );
    }

    const Icon = entityIcon(entity.kind);

    return (
        <section aria-label={copy.graph.inspector.title} className="flex h-full min-h-0 flex-col bg-panel">
            <header className="flex items-start gap-2 border-b border-hairline p-3">
                <Icon size={20} className={`mt-0.5 shrink-0 ${toneClasses[kindTone(entity.kind)].ink}`} />
                <div className="min-w-0 flex-1">
                    <h2 className="truncate text-base font-semibold text-ink">{entity.name}</h2>
                    <p className="flex items-center gap-2 text-2xs text-ink-dim">
                        <span>{kindLabel(entity.kind)}</span>
                        <span className="font-mono">{formatScore(entity.score)}</span>
                        <span>{copy.graph.inspector.rank(rank, index.counts.total)}</span>
                    </p>
                </div>
                <IconButton
                    icon={SmartToyIcon}
                    label={copy.agent.askAbout}
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                        askAgent(copy.agent.ask.entity(entity.name, entity.id));
                    }}
                />
                <IconButton
                    icon={CloseIcon}
                    label={copy.graph.inspector.close}
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                        onSelect(null);
                    }}
                />
            </header>

            <div className="flex flex-col gap-4 p-3">
                <section className="flex flex-col gap-1">
                    <SectionLabel>{copy.graph.inspector.page}</SectionLabel>
                    <CanonicalPage
                        siteId={siteId}
                        entity={entity}
                        onPlan={() => {
                            onPlanPage(entity.id);
                        }}
                    />
                </section>

                <section className="flex flex-col gap-1">
                    <EntityFields entity={entity} />
                </section>

                <section className="flex flex-col gap-1">
                    <SectionLabel>{copy.graph.inspector.anchors}</SectionLabel>
                    <p className="text-2xs text-ink-faint">{copy.graph.inspector.anchorsHint}</p>
                    <AnchorsEditor entity={entity} />
                </section>

                <AuditRows siteId={siteId} entityId={entity.id} audit={audit} />

                <section className="flex flex-col gap-1">
                    <SectionLabel>{copy.graph.inspector.relations}</SectionLabel>
                    <Relations
                        entity={entity}
                        index={index}
                        onReveal={onReveal}
                        onConnect={() => {
                            onConnect(entity.id);
                        }}
                    />
                </section>

                <footer className="flex items-center justify-between gap-2 border-t border-hairline pt-3">
                    <p className="text-2xs text-ink-faint">
                        {copy.graph.inspector.source} {entity.source} · {copy.graph.inspector.created}{" "}
                        {entity.createdAt === null ? "" : relativeTime(entity.createdAt)}
                    </p>
                    <Button
                        size="sm"
                        variant="ghost"
                        icon={DeleteIcon}
                        className="text-danger"
                        onClick={() => {
                            onDelete(entity.id);
                        }}
                    >
                        {copy.graph.remove.confirm}
                    </Button>
                </footer>
            </div>
        </section>
    );
}
