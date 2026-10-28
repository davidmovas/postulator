import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { SkipReason } from "../../generated/vocab.js";
import type { SkippedPage } from "../../data/types.js";
import { Banner } from "../../ui/index.js";

const skipNoEntity: SkipReason = "no_entity";
const pagesShown = 8;

function why(page: SkippedPage): string {
    return page.reason === skipNoEntity ? copy.runs.skipped.noEntity : copy.runs.skipped.noTemplate(page.pageType);
}

export interface SkippedPagesProps {
    skipped: readonly SkippedPage[];
    title: string;
}

export function SkippedPages({ skipped, title }: SkippedPagesProps): ReactElement | null {
    if (skipped.length === 0) {
        return null;
    }
    const shown = skipped.slice(0, pagesShown);
    const more = skipped.length - shown.length;
    const unmapped = skipped.some((page) => page.reason === skipNoEntity);
    const untemplated = skipped.some((page) => page.reason !== skipNoEntity);

    return (
        <Banner
            tone="warn"
            title={title}
            body={
                <div className="flex flex-col gap-1.5" data-run-skipped={skipped.length}>
                    <ul className="flex flex-col gap-0.5 text-xs">
                        {shown.map((page) => (
                            <li key={page.pageId} className="wrap-anywhere">
                                <span className="font-mono text-ink">{page.path}</span> {why(page)}
                            </li>
                        ))}
                        {more > 0 ? <li className="text-ink-dim">{copy.runs.skipped.more(more)}</li> : null}
                    </ul>
                    {untemplated ? <p className="text-xs text-ink-dim">{copy.runs.skipped.fixTemplate}</p> : null}
                    {unmapped ? <p className="text-xs text-ink-dim">{copy.runs.skipped.fixEntity}</p> : null}
                </div>
            }
        />
    );
}
