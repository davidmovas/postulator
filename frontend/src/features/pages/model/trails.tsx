import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { ModelCrumb, ModelTrail } from "../../../data/types.js";
import { cx, Panel, PanelHeader, StatusBadge } from "../../../ui/index.js";
import { Note, PageRef } from "./facts.js";
import { trailSourceLabel } from "./labels.js";

const said = copy.pages.model.trails;

const errorStatus = 400;

interface CrumbNote {
    text: string;
    warn: boolean;
}

function notesOf(crumb: ModelCrumb): CrumbNote[] {
    const notes: CrumbNote[] = [];
    if (crumb.href === "") {
        notes.push({ text: said.noLink, warn: false });
    }
    if (crumb.status >= errorStatus) {
        notes.push({ text: said.status(crumb.status), warn: true });
    } else if (crumb.final !== "" && crumb.final !== crumb.path) {
        notes.push({ text: said.redirects(crumb.final), warn: false });
    }
    if (!crumb.self && crumb.targetId === "") {
        notes.push({ text: said.unresolved, warn: true });
    }
    return notes;
}

interface CrumbRowProps {
    crumb: ModelCrumb;
    index: number;
    siteId: string;
    search: string;
}

function CrumbRow({ crumb, index, siteId, search }: CrumbRowProps): ReactElement {
    return (
        <li className="flex min-w-0 items-center gap-2 border-b border-inset px-3 py-1.5 last:border-b-0">
            <span className="w-4 shrink-0 text-right font-mono text-2xs text-ink-faint">{index + 1}</span>
            <span className="min-w-0 shrink truncate text-xs text-ink">{crumb.name === "" ? said.unnamed : crumb.name}</span>
            {crumb.self ? (
                <StatusBadge tone="accent" dot={false}>
                    {said.self}
                </StatusBadge>
            ) : crumb.targetId === "" ? null : (
                <span className="min-w-0 shrink">
                    <PageRef
                        pageId={crumb.targetId}
                        siteId={siteId}
                        search={search}
                        label={crumb.targetTitle === "" ? crumb.path : crumb.targetTitle}
                    />
                </span>
            )}
            <span className="ml-auto flex shrink-0 items-center gap-2">
                {notesOf(crumb).map((note) => (
                    <span key={note.text} className={cx("text-2xs", note.warn ? "text-warn" : "text-ink-faint")}>
                        {note.text}
                    </span>
                ))}
            </span>
        </li>
    );
}

export interface TrailsPaneProps {
    trails: readonly ModelTrail[];
    siteId: string;
    search: string;
}

export function TrailsPane({ trails, siteId, search }: TrailsPaneProps): ReactElement {
    return (
        <Panel>
            <PanelHeader title={said.title} />
            {trails.length === 0 ? (
                <Note>{said.none}</Note>
            ) : (
                trails.map((trail, at) => {
                    const crumbs = trail.crumbs ?? [];
                    return (
                        <section
                            key={`${trail.source}:${at}`}
                            aria-label={trailSourceLabel(trail.source)}
                            className="border-b border-hairline last:border-b-0"
                        >
                            <div className="flex items-baseline justify-between gap-2 px-3 pt-2 pb-1">
                                <span className="text-xs font-medium text-ink-dim">{trailSourceLabel(trail.source)}</span>
                                <span className="font-mono text-2xs text-ink-faint">{said.crumbs(crumbs.length)}</span>
                            </div>
                            <ol className="flex flex-col">
                                {crumbs.map((crumb, index) => (
                                    <CrumbRow
                                        key={`${index}:${crumb.name}`}
                                        crumb={crumb}
                                        index={index}
                                        siteId={siteId}
                                        search={search}
                                    />
                                ))}
                            </ol>
                        </section>
                    );
                })
            )}
        </Panel>
    );
}
