import type { ReactElement, RefObject } from "react";

import { copy } from "../../copy/index.js";
import { cx, Kbd } from "../../ui/index.js";
import type { Row, Section } from "./sections.js";

export interface PaletteListProps {
    listRef: RefObject<HTMLDivElement | null>;
    siteId: string | null;
    sections: readonly Section[];
    rows: readonly Row[];
    at: number;
    onActive: (index: number) => void;
}

export function PaletteList({ listRef, siteId, sections, rows, at, onActive }: PaletteListProps): ReactElement {
    return (
        <div ref={listRef} className="max-h-96 min-h-0 flex-1 overflow-auto py-1">
            {rows.length === 0 ? (
                <p className="px-3 py-3 text-xs text-ink-dim">{siteId === null ? copy.palette.needSite : copy.palette.nothing}</p>
            ) : (
                sections.map((section) => (
                    <div key={section.key}>
                        <p className="px-3 pt-2 pb-1 text-2xs font-semibold tracking-label text-ink-faint uppercase">{section.label}</p>
                        {section.rows.map((row) => {
                            const picked = rows[at]?.key === row.key;
                            return (
                                <button
                                    key={row.key}
                                    type="button"
                                    tabIndex={-1}
                                    data-active={picked}
                                    onMouseEnter={() => {
                                        onActive(rows.findIndex((held) => held.key === row.key));
                                    }}
                                    onClick={row.run}
                                    className={cx(
                                        "flex h-7 w-full items-center gap-2 px-3 text-left",
                                        picked ? "bg-accent-soft text-ink" : "text-ink-soft",
                                    )}
                                >
                                    <row.Icon size={14} className="shrink-0 text-ink-dim" />
                                    <span className="min-w-0 flex-1 truncate text-sm">{row.label}</span>
                                    <span className="shrink-0 font-mono text-2xs text-ink-faint">{row.kind}</span>
                                </button>
                            );
                        })}
                    </div>
                ))
            )}
        </div>
    );
}

export function PaletteFooter(): ReactElement {
    return (
        <div className="flex shrink-0 items-center gap-3 border-t border-hairline px-3 py-1.5 text-2xs text-ink-faint">
            <span className="flex items-center gap-1">
                <Kbd keys={["↑", "↓"]} />
                {copy.palette.navigate}
            </span>
            <span className="flex items-center gap-1">
                <Kbd keys={["↵"]} />
                {copy.palette.open}
            </span>
            <span className="flex items-center gap-1">
                <Kbd keys={["Esc"]} />
                {copy.palette.close}
            </span>
            <span className="ml-auto flex items-center gap-1">
                <Kbd keys={["F"]} />
                {copy.palette.search}
            </span>
        </div>
    );
}
