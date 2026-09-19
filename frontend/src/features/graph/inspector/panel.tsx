import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { relativeTime } from "../../../domain/format.js";
import { Button, CloseIcon, DeleteIcon, IconButton, SectionLabel, toneClasses } from "../../../ui/index.js";
import { entityIcon, formatScore, kindTone } from "../labels.js";
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
}

export function Inspector({ siteId, index, selectedId, onSelect, onReveal, onLens, onConnect, onDelete, onPlanPage }: InspectorProps): ReactElement {
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
        </aside>
    );
}
