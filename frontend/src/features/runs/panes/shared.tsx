import type { ReactElement, ReactNode } from "react";

import { copy } from "../../../copy/index.js";
import { isBrowsable, openExternal } from "../../../data/host.js";
import { Banner, OpenInNewIcon } from "../../../ui/index.js";

export interface RowsProps {
    entries: readonly (readonly [string, ReactNode])[];
}

export function Rows({ entries }: RowsProps): ReactElement {
    return (
        <dl className="grid grid-cols-[minmax(0,10rem)_1fr] gap-x-3 gap-y-1 px-3 py-2 text-xs">
            {entries.map(([label, value]) => (
                <div key={label} className="contents">
                    <dt className="truncate text-ink-faint">{label}</dt>
                    <dd className="min-w-0 truncate font-mono text-ink-soft">{value}</dd>
                </div>
            ))}
        </dl>
    );
}

export interface ExternalUrlProps {
    url: string;
}

export function ExternalUrl({ url }: ExternalUrlProps): ReactElement {
    if (!isBrowsable(url)) {
        return <span className="truncate font-mono text-xs text-ink-soft select-all">{url}</span>;
    }
    return (
        <button
            type="button"
            title={copy.runs.openOnSite}
            onClick={() => {
                void openExternal(url);
            }}
            className="flex min-w-0 items-center gap-1 text-left font-mono text-xs text-accent hover:underline"
        >
            <span className="truncate">{url}</span>
            <OpenInNewIcon size={12} className="shrink-0" />
        </button>
    );
}

export function Unreadable(): ReactElement {
    return (
        <div className="p-3">
            <Banner tone="warn" title={copy.runs.review.unreadable} body={copy.runs.review.unreadableBody} />
        </div>
    );
}
