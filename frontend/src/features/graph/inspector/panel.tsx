import type { ReactElement } from "react";
import { useMemo } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { relativeTime } from "../../../domain/format.js";
import { Banner, CloseIcon, IconButton, LinkOffIcon, SectionLabel, StatusBadge, toneClasses } from "../../../ui/index.js";
import { entityIcon, formatScore, kindTone } from "../labels.js";
import type { GraphIndex } from "../model/index.js";
import type { Lens } from "../model/lens.js";
import { Summary } from "./summary.js";

const childrenShown = 10;

export interface InspectorProps {
    siteId: string;
    index: GraphIndex;
    selectedId: string | null;
    onSelect: (id: string | null) => void;
    onReveal: (id: string) => void;
    onLens: (lens: Lens) => void;
}

interface RelationRowProps {
    id: string;
    index: GraphIndex;
    note?: string;
    tone?: "info" | "muted";
    onPick: (id: string) => void;
}

function RelationRow({ id, index, note, tone, onPick }: RelationRowProps): ReactElement {
    const held = index.byId.get(id);
    const Icon = entityIcon(held?.kind ?? "");
    return (
        <li>
            <button
                type="button"
                className="flex w-full items-center gap-2 rounded-md px-1 py-0.5 text-left text-xs text-ink-soft hover:bg-inset hover:text-ink"
                onClick={() => {
                    onPick(id);
                }}
            >
                <Icon size={14} className={toneClasses[kindTone(held?.kind ?? "")].ink} />
                <span className="flex-1 truncate">{held?.name ?? id}</span>
                {note === undefined ? null : (
                    <span className={`font-mono text-2xs ${tone === "info" ? "text-info" : "text-ink-faint"}`}>{note}</span>
                )}
            </button>
        </li>
    );
}

export function Inspector({ siteId, index, selectedId, onSelect, onReveal, onLens }: InspectorProps): ReactElement {
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
            <aside aria-label={copy.graph.summary.title} className="flex w-80 shrink-0 flex-col overflow-auto border-l border-hairline bg-panel">
                <Summary index={index} onLens={onLens} onSelect={onSelect} />
            </aside>
        );
    }

    const Icon = entityIcon(entity.kind);
    const flags = index.problems.get(entity.id);
    const parents = index.approvedParents.get(entity.id) ?? [];
    const proposedParents = index.proposedParents.get(entity.id) ?? [];
    const children = index.children.get(entity.id) ?? [];
    const related = index.related.get(entity.id) ?? [];
    const keywords = entity.secondaryKeywords ?? [];
    const anchors = entity.anchors ?? [];

    return (
        <aside aria-label={copy.graph.inspector.title} className="flex w-80 shrink-0 flex-col overflow-auto border-l border-hairline bg-panel">
            <header className="flex items-start gap-2 border-b border-hairline p-3">
                <Icon size={20} className={`mt-0.5 shrink-0 ${toneClasses[kindTone(entity.kind)].ink}`} />
                <div className="min-w-0 flex-1">
                    <h2 className="truncate text-base font-semibold text-ink">{entity.name}</h2>
                    <p className="flex items-center gap-2 text-2xs text-ink-dim">
                        <span>{entity.kind}</span>
                        <span className="font-mono">{formatScore(entity.score)}</span>
                        <span>{copy.graph.inspector.rank(rank, index.counts.total)}</span>
                    </p>
                </div>
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
                {flags?.noPage ? (
                    <Banner tone="danger" icon={LinkOffIcon} title={copy.graph.inspector.noPage} body={copy.graph.inspector.noPageBody} />
                ) : (
                    <section className="flex flex-col gap-1">
                        <SectionLabel>{copy.graph.inspector.page}</SectionLabel>
                        <Link to={`/s/${siteId}/pages/${entity.canonicalPageId ?? ""}`} className="truncate text-xs">
                            {copy.graph.inspector.openPage}
                        </Link>
                    </section>
                )}

                <section className="flex flex-col gap-1">
                    <SectionLabel>{copy.graph.inspector.primaryKeyword}</SectionLabel>
                    <p className="font-mono text-xs text-ink">{entity.primaryKeyword === "" ? "—" : entity.primaryKeyword}</p>
                    {keywords.length > 0 ? (
                        <ul className="flex flex-wrap gap-1">
                            {keywords.map((keyword) => (
                                <li key={keyword} className="rounded-sm bg-inset px-1.5 py-0.5 font-mono text-2xs text-ink-soft">
                                    {keyword}
                                </li>
                            ))}
                        </ul>
                    ) : null}
                    {entity.intent === "" ? null : <p className="text-xs text-ink-dim">{entity.intent}</p>}
                </section>

                <section className="flex flex-col gap-1">
                    <SectionLabel>{copy.graph.inspector.anchors}</SectionLabel>
                    {anchors.length === 0 ? (
                        <p className="text-xs text-ink-dim">{copy.graph.inspector.noAnchors}</p>
                    ) : (
                        <ul className="flex flex-col gap-0.5">
                            {anchors.map((anchor) => (
                                <li key={anchor.text} className="flex items-center gap-2 text-xs">
                                    <span className="flex-1 truncate font-mono text-ink">{anchor.text}</span>
                                    <StatusBadge tone={anchor.source === "user" ? "ok" : "info"} dot={false}>
                                        {anchor.source}
                                    </StatusBadge>
                                    <span className="font-mono text-2xs text-ink-faint">{anchor.weight.toFixed(1)}</span>
                                </li>
                            ))}
                        </ul>
                    )}
                </section>

                <section className="flex flex-col gap-2">
                    <SectionLabel>{copy.graph.inspector.relations}</SectionLabel>
                    {parents.length + proposedParents.length + children.length + related.length === 0 ? (
                        <p className="text-xs text-ink-dim">{copy.graph.inspector.noRelations}</p>
                    ) : null}
                    {parents.length + proposedParents.length > 0 ? (
                        <div>
                            <p className="text-2xs text-ink-faint">{copy.graph.inspector.parents}</p>
                            <ul>
                                {parents.map((id) => (
                                    <RelationRow key={id} id={id} index={index} onPick={onReveal} />
                                ))}
                                {proposedParents.map((id) => (
                                    <RelationRow key={id} id={id} index={index} note={copy.graph.inspector.proposedParent} tone="info" onPick={onReveal} />
                                ))}
                            </ul>
                        </div>
                    ) : null}
                    {children.length > 0 ? (
                        <div>
                            <p className="text-2xs text-ink-faint">
                                {copy.graph.inspector.children} · {children.length}
                            </p>
                            <ul>
                                {children.slice(0, childrenShown).map((id) => (
                                    <RelationRow key={id} id={id} index={index} note={formatScore(index.byId.get(id)?.score ?? 0)} onPick={onReveal} />
                                ))}
                            </ul>
                            {children.length > childrenShown ? (
                                <p className="px-1 text-2xs text-ink-faint">{copy.graph.inspector.moreChildren(children.length - childrenShown)}</p>
                            ) : null}
                        </div>
                    ) : null}
                    {related.length > 0 ? (
                        <div>
                            <p className="text-2xs text-ink-faint">{copy.graph.inspector.related}</p>
                            <ul>
                                {related.map((link) => (
                                    <RelationRow
                                        key={link.edgeId}
                                        id={link.otherId}
                                        index={index}
                                        note={link.status === "proposed" ? `${link.weight.toFixed(2)} · ${link.status}` : link.weight.toFixed(2)}
                                        tone={link.status === "proposed" ? "info" : "muted"}
                                        onPick={onReveal}
                                    />
                                ))}
                            </ul>
                        </div>
                    ) : null}
                </section>

                <p className="text-2xs text-ink-faint">
                    {copy.graph.inspector.source} {entity.source} · {copy.graph.inspector.created}{" "}
                    {entity.createdAt === null ? "" : relativeTime(entity.createdAt)}
                </p>
            </div>
        </aside>
    );
}
