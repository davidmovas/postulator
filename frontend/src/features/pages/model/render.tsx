import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { ModelRender } from "../../../data/types.js";
import { absoluteTime, relativeTime } from "../../../domain/format.js";
import { Panel, PanelHeader, StatusBadge } from "../../../ui/index.js";
import { Fact, Facts, Note } from "./facts.js";
import { regionKindLabel, rootStateLabel, rootStateTone, urlFormLabel } from "./labels.js";

const said = copy.pages.model.render;

export interface RenderPaneProps {
    render: ModelRender | null;
}

export function RenderPane({ render }: RenderPaneProps): ReactElement {
    if (render === null) {
        return (
            <Panel>
                <PanelHeader title={said.title} />
                <Note>{said.none}</Note>
            </Panel>
        );
    }
    const regions = render.regions ?? [];

    return (
        <Panel>
            <PanelHeader title={said.title}>
                <span className="shrink-0 font-mono text-2xs text-ink-faint" title={absoluteTime(render.extractedAt)}>
                    {said.extracted(relativeTime(render.extractedAt))}
                </span>
            </PanelHeader>
            <Facts>
                <Fact label={said.root}>
                    <StatusBadge tone={rootStateTone(render.root)}>{rootStateLabel(render.root)}</StatusBadge>
                </Fact>
                <Fact label={said.words}>{said.wordCounts(render.wordsContent, render.wordsGenerated)}</Fact>
                {render.builder === "" ? null : <Fact label={said.builder}>{render.builder}</Fact>}
                <Fact label={said.urlForm}>{urlFormLabel(render.urlForm)}</Fact>
                <Fact label={said.robots} mono={render.robots !== ""}>
                    {render.robots === "" ? said.robotsUnset : render.robots}
                    {render.noIndex ? <span className="ml-2 font-sans text-warn">{said.noIndex}</span> : null}
                </Fact>
            </Facts>
            {render.finalPath === "" ? null : (
                <p className="px-3 pb-2 text-xs text-warn">{said.landedOn(render.finalPath)}</p>
            )}
            {regions.length === 0 ? null : (
                <section aria-label={said.regions} className="border-t border-hairline">
                    <div className="px-3 pt-2 pb-1 text-xs font-medium text-ink-dim">{said.regions}</div>
                    <ul className="flex flex-col">
                        {regions.map((region, index) => (
                            <li
                                key={`${region.kind}:${index}`}
                                className="flex min-w-0 items-center gap-2 border-b border-inset px-3 py-1.5 last:border-b-0"
                            >
                                <span className="min-w-0 flex-1 truncate text-xs text-ink">{regionKindLabel(region.kind)}</span>
                                <span className="shrink-0 text-2xs text-ink-faint">
                                    {region.inside ? said.inside : said.outside}
                                </span>
                                <span className="w-32 shrink-0 text-right font-mono text-2xs text-ink-dim">
                                    {said.regionCounts(region.links, region.words)}
                                </span>
                            </li>
                        ))}
                    </ul>
                </section>
            )}
        </Panel>
    );
}
