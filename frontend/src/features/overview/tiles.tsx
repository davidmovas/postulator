import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { tokens } from "../../domain/format.js";
import type { Tone } from "../../ui/index.js";
import { cx, Panel, SectionLabel, toneClasses } from "../../ui/index.js";
import type { EdgeTile, EntityTile, PageTile } from "./model/overview.js";

export interface TileRow {
    label: string;
    value: number;
    tone: Tone;
}

interface TileProps {
    title: string;
    total: string;
    rows: readonly TileRow[];
}

function Tile({ title, total, rows }: TileProps): ReactElement {
    const sum = rows.reduce((carried, row) => carried + row.value, 0);
    return (
        <Panel className="flex flex-col gap-2.5 p-3">
            <div className="flex items-baseline justify-between gap-2">
                <SectionLabel>{title}</SectionLabel>
                <span className="text-lg font-semibold tracking-tight text-ink">{total}</span>
            </div>
            <div className="flex h-1 gap-px overflow-hidden rounded-sm">
                {rows.map((row) => (
                    <span
                        key={row.label}
                        aria-hidden={true}
                        className={cx("h-full", toneClasses[row.tone].solid)}
                        style={{ width: `${sum === 0 ? 0 : (row.value / sum) * 100}%` }}
                    />
                ))}
            </div>
            <dl className="flex flex-col gap-1">
                {rows.map((row) => (
                    <div key={row.label} className="flex items-center gap-2">
                        <span
                            aria-hidden={true}
                            className={cx("h-1.5 w-1.5 shrink-0 rounded-xs", toneClasses[row.tone].solid)}
                        />
                        <dt className="min-w-0 flex-1 truncate text-xs text-ink-soft">{row.label}</dt>
                        <dd className="shrink-0 font-mono text-xs text-ink">{tokens(row.value)}</dd>
                    </div>
                ))}
            </dl>
        </Panel>
    );
}

export interface TilesProps {
    entities: EntityTile;
    pages: PageTile;
    edges: EdgeTile;
}

export function Tiles({ entities, pages, edges }: TilesProps): ReactElement {
    return (
        <div className="grid grid-cols-1 gap-3 @md:grid-cols-3">
            <Tile
                title={copy.overview.tiles.entities}
                total={tokens(entities.total)}
                rows={[
                    { label: copy.overview.tiles.withPublished, value: entities.withPublished, tone: "ok" },
                    {
                        label: copy.overview.tiles.withPlanned,
                        value: Math.max(0, entities.withCanonical - entities.withPublished),
                        tone: "accent",
                    },
                    { label: copy.overview.tiles.withoutPage, value: entities.withoutPage, tone: "warn" },
                ]}
            />
            <Tile
                title={copy.overview.tiles.pages}
                total={tokens(pages.total)}
                rows={[
                    { label: copy.overview.tiles.mapped, value: pages.mapped, tone: "ok" },
                    { label: copy.overview.tiles.unmapped, value: pages.unmapped, tone: "warn" },
                    { label: copy.overview.tiles.orphans, value: pages.orphans, tone: "danger" },
                ]}
            />
            <Tile
                title={copy.overview.tiles.edges}
                total={edges.capped ? copy.overview.tiles.capped(edges.approved) : tokens(edges.approved)}
                rows={[
                    { label: copy.overview.tiles.realized, value: edges.realized, tone: "ok" },
                    { label: copy.overview.tiles.notLinked, value: edges.gap, tone: "accent" },
                    { label: copy.overview.tiles.proposed, value: edges.proposed, tone: "info" },
                ]}
            />
        </div>
    );
}
