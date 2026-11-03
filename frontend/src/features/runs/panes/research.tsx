import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { isBrowsable, openExternal } from "../../../data/host.js";
import { OpenInNewIcon, SectionLabel } from "../../../ui/index.js";
import { researchView } from "../artifacts.js";
import { Unreadable } from "./shared.js";

const said = copy.runs.review.research;

function hostOf(url: string): string {
    try {
        return new URL(url).hostname;
    } catch {
        return url;
    }
}

export interface ResearchPaneProps {
    payload: unknown;
}

export function ResearchPane({ payload }: ResearchPaneProps): ReactElement {
    const view = researchView(payload);
    if (view === null) {
        return <Unreadable />;
    }

    return (
        <div className="flex flex-col gap-3 p-3" data-research-pane={true}>
            <p className="text-xs text-ink-dim">{said.summary(view.facts.length, view.sources.length, view.searches)}</p>
            <div>
                <SectionLabel>{said.facts}</SectionLabel>
                {view.facts.length === 0 ? (
                    <p className="pt-1 text-xs text-ink-dim">{said.noFacts}</p>
                ) : (
                    <ul className="flex flex-col gap-1 pt-1">
                        {view.facts.map((fact, index) => (
                            <li key={`${index}:${fact}`} className="text-xs leading-snug text-ink-soft">
                                {fact}
                            </li>
                        ))}
                    </ul>
                )}
            </div>
            <div>
                <SectionLabel>{said.sources}</SectionLabel>
                {view.sources.length === 0 ? (
                    <p className="pt-1 text-xs text-ink-dim">{said.noSources}</p>
                ) : (
                    <ul className="flex flex-col gap-1 pt-1">
                        {view.sources.map((source) => (
                            <li key={source.url} className="min-w-0">
                                {isBrowsable(source.url) ? (
                                    <button
                                        type="button"
                                        title={source.url}
                                        onClick={() => {
                                            void openExternal(source.url);
                                        }}
                                        className="flex max-w-full min-w-0 items-center gap-1.5 text-left text-xs text-accent hover:underline"
                                    >
                                        <span className="truncate">{source.title === "" ? source.url : source.title}</span>
                                        <span className="shrink-0 font-mono text-2xs text-ink-faint">{hostOf(source.url)}</span>
                                        <OpenInNewIcon size={12} className="shrink-0" />
                                    </button>
                                ) : (
                                    <span className="truncate font-mono text-xs text-ink-soft">{source.url}</span>
                                )}
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    );
}
