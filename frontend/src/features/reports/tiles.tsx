import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { cx, toneClasses } from "../../ui/index.js";
import type { Tone } from "../../ui/index.js";
import { shareOf, shareTone } from "./labels.js";
import type { SiteTiles } from "./model/site.js";

interface TileProps {
    label: string;
    value: string;
    note: string;
    tone?: Tone;
}

function Tile({ label, value, note, tone }: TileProps): ReactElement {
    return (
        <article className="flex min-w-36 flex-1 flex-col gap-0.5 rounded-lg border border-hairline bg-panel px-3 py-2">
            <h3 className="truncate text-2xs font-semibold tracking-label text-ink-faint uppercase">{label}</h3>
            <p className={cx("font-mono text-2xl leading-none", tone === undefined ? "text-ink" : toneClasses[tone].ink)}>
                {value}
            </p>
            <p className="truncate text-2xs text-ink-dim">{note}</p>
        </article>
    );
}

export interface SiteTileRowProps {
    tiles: SiteTiles;
    drifted: number;
}

export function SiteTileRow({ tiles, drifted }: SiteTileRowProps): ReactElement {
    return (
        <div className="flex flex-wrap gap-2">
            <Tile
                label={copy.reports.tiles.pagesMapped}
                value={shareOf(tiles.mappedShare)}
                note={copy.reports.tiles.pagesMappedNote(tiles.pagesMapped, tiles.pagesTotal)}
                tone={shareTone(tiles.mappedShare)}
            />
            <Tile
                label={copy.reports.tiles.entitiesWithPage}
                value={shareOf(tiles.withPageShare)}
                note={copy.reports.tiles.entitiesWithPageNote(tiles.entitiesWithPage, tiles.entitiesTotal)}
                tone={shareTone(tiles.withPageShare)}
            />
            <Tile
                label={copy.reports.tiles.orphans}
                value={String(tiles.orphans)}
                note={copy.reports.tiles.orphansNote}
                tone={tiles.orphans > 0 ? "warn" : "ok"}
            />
            <Tile
                label={copy.reports.tiles.drifted}
                value={String(drifted)}
                note={copy.reports.tiles.driftedNote}
                tone={drifted > 0 ? "warn" : "ok"}
            />
            <Tile
                label={copy.reports.tiles.depth}
                value={tiles.averageDepth.toFixed(1)}
                note={copy.reports.tiles.depthNote}
            />
        </div>
    );
}
