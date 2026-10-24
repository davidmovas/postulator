import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { cx } from "../../ui/index.js";
import { columnUseLabel } from "./labels.js";
import type { PreviewColumn } from "./preview.js";
import { columnsBySheet } from "./preview.js";

function ColumnChips({ columns }: { columns: readonly PreviewColumn[] }): ReactElement {
    return (
        <>
            {columns.map((column, at) => (
                <span
                    key={`${column.header}-${String(at)}`}
                    className={cx(
                        "flex h-5 items-center gap-1 rounded-sm bg-inset px-1.5 text-2xs",
                        column.use === "ignored" ? "text-ink-faint" : "text-ink-soft",
                    )}
                >
                    <span className="font-mono">{column.header === "" ? "—" : column.header}</span>
                    <span aria-hidden={true}>→</span>
                    {columnUseLabel(column)}
                </span>
            ))}
        </>
    );
}

export function ColumnUses({ columns }: { columns: readonly PreviewColumn[] }): ReactElement | null {
    if (columns.length === 0) {
        return null;
    }
    const bySheet = columnsBySheet(columns);
    return (
        <section
            aria-label={copy.imports.preview.columns}
            className="flex shrink-0 flex-col gap-1.5 border-b border-hairline px-3 py-2"
        >
            {bySheet.length < 2 ? (
                <div className="flex flex-wrap items-center gap-1.5">
                    <span className="text-2xs text-ink-faint">{copy.imports.preview.columns}</span>
                    <ColumnChips columns={columns} />
                </div>
            ) : (
                <>
                    <span className="text-2xs text-ink-faint">{copy.imports.preview.columns}</span>
                    {bySheet.map((held) => (
                        <div key={held.sheet} className="flex flex-wrap items-center gap-1.5">
                            <span
                                className="w-24 shrink-0 truncate text-2xs font-medium text-ink-dim"
                                title={held.sheet}
                            >
                                {held.sheet}
                            </span>
                            <ColumnChips columns={held.columns} />
                        </div>
                    ))}
                </>
            )}
        </section>
    );
}
