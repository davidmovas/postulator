import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { usePageModel } from "../../../data/hooks/pages.js";
import type { ModelRegion } from "../../../data/types.js";
import { SectionLabel } from "../../../ui/index.js";
import { regionKindLabel } from "../../pages/model/labels.js";

const ours: ReadonlySet<string> = new Set(["content", "generated"]);

export function injected(regions: readonly ModelRegion[] | null | undefined): ModelRegion[] {
    return (regions ?? []).filter((region) => !ours.has(region.kind));
}

export function SiteRegions({ pageId }: { pageId: string }): ReactElement | null {
    const model = usePageModel(pageId);
    const regions = injected(model.data?.render?.regions);
    if (regions.length === 0) {
        return null;
    }
    const render = copy.pages.model.render;
    return (
        <section aria-label={copy.runs.review.diff.regions} className="flex flex-col">
            <SectionLabel className="px-3">{copy.runs.review.diff.regions}</SectionLabel>
            <ul className="flex flex-col">
                {regions.map((region, index) => (
                    <li
                        key={`${region.kind}:${String(index)}`}
                        className="flex min-w-0 items-center gap-2 border-b border-inset px-3 py-1.5 last:border-b-0"
                    >
                        <span className="min-w-0 flex-1 truncate text-xs text-ink">{regionKindLabel(region.kind)}</span>
                        <span className="shrink-0 text-2xs text-ink-faint">{region.inside ? render.inside : render.outside}</span>
                        <span className="w-32 shrink-0 text-right font-mono text-2xs text-ink-dim">
                            {render.regionCounts(region.links, region.words)}
                        </span>
                    </li>
                ))}
            </ul>
        </section>
    );
}
