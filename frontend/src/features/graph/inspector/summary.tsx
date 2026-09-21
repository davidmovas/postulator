import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { entityKinds } from "../../../generated/vocab.js";
import { Button, CalculateIcon, SectionLabel, toneClasses } from "../../../ui/index.js";
import { entityIcon, formatScore, kindLabel, kindTone, lensLabel, lensTone, scored } from "../labels.js";
import type { GraphIndex } from "../model/index.js";
import type { Lens } from "../model/lens.js";

const topCount = 5;

export interface SummaryProps {
    index: GraphIndex;
    recomputing: boolean;
    onLens: (lens: Lens) => void;
    onSelect: (id: string) => void;
    onRecompute: () => void;
}

interface AttentionRow {
    lens: Lens;
    label: string;
}

export function Summary({ index, recomputing, onLens, onSelect, onRecompute }: SummaryProps): ReactElement {
    const byKind = useMemo(() => {
        const counts = new Map<string, number>();
        for (const held of index.byId.values()) {
            counts.set(held.kind, (counts.get(held.kind) ?? 0) + 1);
        }
        return entityKinds.map((kind) => ({ kind, count: counts.get(kind) ?? 0 })).filter((entry) => entry.count > 0);
    }, [index]);

    const ranked = scored(index.entities);

    const top = useMemo(
        () => [...index.entities].sort((left, right) => right.score - left.score || left.name.localeCompare(right.name)).slice(0, topCount),
        [index],
    );

    const attention: AttentionRow[] = [];
    if (index.counts.noPage > 0) {
        attention.push({ lens: "noPage", label: copy.graph.summary.noPage(index.counts.noPage) });
    }
    if (index.counts.orphan > 0) {
        attention.push({ lens: "orphan", label: copy.graph.summary.orphan(index.counts.orphan) });
    }
    if (index.counts.proposedEdges > 0) {
        attention.push({ lens: "proposed", label: copy.graph.summary.proposed(index.counts.proposedEdges) });
    }

    return (
        <div className="flex flex-col gap-4 p-3">
            <div>
                <h2 className="text-base font-semibold text-ink">{copy.graph.summary.title}</h2>
                <p className="text-xs text-ink-dim">{copy.graph.total(index.counts.total)}</p>
                <p className="mt-1 text-2xs text-ink-faint">{copy.graph.summary.hint}</p>
            </div>

            <section className="flex flex-col gap-1">
                <SectionLabel>{copy.graph.summary.byKind}</SectionLabel>
                <ul className="flex flex-col gap-0.5">
                    {byKind.map(({ kind, count }) => {
                        const Icon = entityIcon(kind);
                        return (
                            <li key={kind} className="flex items-center justify-between text-xs">
                                <span className="flex items-center gap-2 text-ink-soft">
                                    <Icon size={14} className={toneClasses[kindTone(kind)].ink} />
                                    {kindLabel(kind)}
                                </span>
                                <span className="font-mono text-2xs text-ink-dim">{count}</span>
                            </li>
                        );
                    })}
                </ul>
            </section>

            <section className="flex flex-col gap-1">
                <SectionLabel>{copy.graph.summary.attention}</SectionLabel>
                {attention.length === 0 ? (
                    <p className="text-xs text-ink-dim">{copy.graph.summary.clean}</p>
                ) : (
                    <ul className="flex flex-col gap-0.5">
                        {attention.map((row) => (
                            <li key={row.lens}>
                                <button
                                    type="button"
                                    className="flex w-full items-center gap-2 rounded-md px-1 py-0.5 text-left text-xs text-ink-soft hover:bg-inset hover:text-ink"
                                    onClick={() => {
                                        onLens(row.lens);
                                    }}
                                >
                                    <span className={`h-1.5 w-1.5 rounded-full ${toneClasses[lensTone(row.lens)].solid}`} aria-hidden={true} />
                                    <span className="flex-1 truncate">{row.label}</span>
                                    <span className="text-2xs text-ink-faint">{lensLabel(row.lens)}</span>
                                </button>
                            </li>
                        ))}
                        {index.counts.multiParent > 0 ? (
                            <li className="px-1 py-0.5 text-xs text-ink-dim">{copy.graph.summary.multiParent(index.counts.multiParent)}</li>
                        ) : null}
                    </ul>
                )}
            </section>

            {index.entities.length === 0 ? null : (
                <section className="flex flex-col gap-1">
                    <SectionLabel>{copy.graph.summary.top}</SectionLabel>
                    {ranked ? (
                        <ol className="flex flex-col gap-0.5">
                            {top.map((held) => {
                                const Icon = entityIcon(held.kind);
                                return (
                                    <li key={held.id}>
                                        <button
                                            type="button"
                                            className="flex w-full items-center gap-2 rounded-md px-1 py-0.5 text-left text-xs text-ink-soft hover:bg-inset hover:text-ink"
                                            onClick={() => {
                                                onSelect(held.id);
                                            }}
                                        >
                                            <Icon size={14} className={toneClasses[kindTone(held.kind)].ink} />
                                            <span className="flex-1 truncate">{held.name}</span>
                                            <span className="font-mono text-2xs text-ink-dim">{formatScore(held.score)}</span>
                                        </button>
                                    </li>
                                );
                            })}
                        </ol>
                    ) : (
                        <div className="flex flex-col items-start gap-1.5 pt-0.5">
                            <p className="text-xs text-ink-dim">{copy.graph.score.notComputed}</p>
                            <p className="text-2xs text-ink-faint">{copy.graph.score.notComputedHint}</p>
                            <Button size="sm" variant="secondary" icon={CalculateIcon} busy={recomputing} onClick={onRecompute}>
                                {copy.graph.score.compute}
                            </Button>
                        </div>
                    )}
                </section>
            )}
        </div>
    );
}
