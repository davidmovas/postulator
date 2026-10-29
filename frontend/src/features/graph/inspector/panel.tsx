import type { ReactElement } from "react";
import { useMemo } from "react";

import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import type { PageAudit } from "../../../data/types.js";
import { relativeTime } from "../../../domain/format.js";
import {
    Button,
    CloseIcon,
    cx,
    DeleteIcon,
    EditNoteIcon,
    IconButton,
    SectionLabel,
    SmartToyIcon,
    toneClasses,
} from "../../../ui/index.js";
import { askAgent } from "../../agent/index.js";
import { severityOf } from "../../links/model/audit.js";
import { pageTypesOf } from "../../settings/tabs.js";
import { presetIcon, presetLabel, presetTone } from "../../settings/types/labels.js";
import { entityTypeLabel, formatScore, sourceLabel } from "../labels.js";
import type { EntityType, GraphIndex } from "../model/index.js";
import { presetOf, typeOf } from "../model/index.js";
import type { Lens } from "../model/lens.js";
import { AnchorsEditor } from "./anchors.js";
import { EntityCard } from "./card.js";
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
    proposed: ReadonlySet<string>;
    onReviewCard: (entityId: string) => void;
}

interface PageTypeLineProps {
    siteId: string;
    type: EntityType | null;
}

function PageTypeLine({ siteId, type }: PageTypeLineProps): ReactElement {
    if (type === null) {
        return <p className="text-2xs text-ink-faint">{copy.graph.inspector.typeNone}</p>;
    }
    const Icon = presetIcon(type.preset);
    const role = presetLabel(type.preset);
    return (
        <div className="flex items-center gap-1.5 text-xs" data-entity-page-type={type.preset}>
            <span className="text-ink-faint">{copy.graph.inspector.type}</span>
            <Icon size={14} className={toneClasses[presetTone(type.preset)].ink} />
            <span className="min-w-0 truncate text-ink">{type.name === role ? type.name : `${type.name} · ${role}`}</span>
            <Link to={`${pageTypesOf(siteId)}&type=${encodeURIComponent(type.id)}`} className="ml-auto shrink-0 text-2xs">
                {copy.graph.inspector.typeOpen}
            </Link>
        </div>
    );
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
                                              row.onSite ? copy.graph.inspector.linksRow(row.satisfied, row.targets) : copy.graph.inspector.linksNotWritten,
                                              row.missing > 0 ? copy.graph.inspector.linksMissing(row.missing, row.missingRequired) : null,
                                              row.pending > 0 && row.onSite ? copy.graph.inspector.linksPending(row.pending) : null,
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
    proposed,
    onReviewCard,
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

    const preset = presetOf(index, entity.id);
    const Icon = presetIcon(preset);

    return (
        <section aria-label={copy.graph.inspector.title} className="flex h-full min-h-0 flex-col bg-panel">
            <header className="flex items-start gap-2 border-b border-hairline p-3">
                <Icon size={20} className={`mt-0.5 shrink-0 ${toneClasses[presetTone(preset)].ink}`} />
                <div className="min-w-0 flex-1">
                    <h2 className="truncate text-base font-semibold text-ink">{entity.name}</h2>
                    <p className="flex items-center gap-2 text-2xs text-ink-dim">
                        <span data-entity-type={preset}>{entityTypeLabel(typeOf(index, entity.id))}</span>
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
                    {entity.canonicalPageId === null ? null : <PageTypeLine siteId={siteId} type={typeOf(index, entity.id)} />}
                </section>

                <section className="flex flex-col gap-1">
                    <EntityFields entity={entity} />
                </section>

                <section className="flex flex-col gap-1" data-entity-card={entity.id}>
                    <SectionLabel>{copy.graph.entityCard.title}</SectionLabel>
                    {proposed.has(entity.id) ? (
                        <div
                            className="flex items-center gap-2 rounded-md border border-hairline bg-inset px-2.5 py-1.5"
                            data-card-proposal-waiting={entity.id}
                        >
                            <EditNoteIcon size={14} className={`shrink-0 ${toneClasses.info.ink}`} />
                            <span className="min-w-0 flex-1 text-xs text-ink-soft">{copy.graph.cards.waitingFor}</span>
                            <Button
                                size="sm"
                                variant="secondary"
                                onClick={() => {
                                    onReviewCard(entity.id);
                                }}
                            >
                                {copy.graph.cards.review}
                            </Button>
                        </div>
                    ) : null}
                    <EntityCard entity={entity} />
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
                        {sourceLabel(entity.source)} · {copy.graph.inspector.created}{" "}
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
