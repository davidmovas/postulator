import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../copy/index.js";
import type { RunEventsState } from "../../data/runs/log.js";
import { absoluteTime, duration, relativeTime, tokens as formatTokens, usd as formatUsd } from "../../domain/format.js";
import {
    CloudSyncIcon,
    cx,
    EmptyState,
    HistoryIcon,
    PanelHeader,
    Spinner,
    toneClasses,
} from "../../ui/index.js";
import { countdown } from "./authority.js";
import { eventIcon, eventName, eventTone } from "./labels.js";
import type { FeedEntry } from "./log-view.js";
import { feed } from "./log-view.js";

const feedLimit = 200;

function detailOf(entry: FeedEntry): string {
    const parts: string[] = [];
    if (entry.step !== null) {
        parts.push(entry.step);
    }
    if (entry.items !== null) {
        parts.push(copy.runs.detail.targets(entry.items));
    }
    if (entry.reason !== null && entry.reason !== "") {
        parts.push(entry.reason);
    }
    if (entry.attempt !== null) {
        parts.push(copy.runs.events.attempt(entry.attempt));
    }
    if (entry.afterMs !== null) {
        parts.push(copy.runs.events.retryIn(countdown(entry.afterMs)));
    }
    if (entry.durationMs !== null) {
        parts.push(duration(entry.durationMs));
    }
    if (entry.model !== null && entry.model !== "") {
        parts.push(entry.model);
    }
    if (entry.tokens !== null) {
        parts.push(formatTokens(entry.tokens));
    }
    if (entry.usd !== null) {
        parts.push(formatUsd(entry.usd));
    }
    if (entry.code !== null && entry.code !== "") {
        parts.push(entry.code);
    }
    if (entry.message !== null && entry.message !== "") {
        parts.push(entry.message);
    }
    return parts.join(" · ");
}

export interface RunEventFeedProps {
    events: RunEventsState;
    gap: boolean;
    itemId: string | null;
    paths: ReadonlyMap<string, string>;
}

export function RunEventFeed({ events, gap, itemId, paths }: RunEventFeedProps): ReactElement {
    const entries = useMemo(() => feed(events.events, feedLimit, itemId), [events, itemId]);
    const catching = events.phase === "catching-up";

    return (
        <section className="flex min-h-0 w-80 shrink-0 flex-col border-l border-hairline bg-panel">
            <PanelHeader title={copy.runs.events.title}>
                {gap || catching ? (
                    <span className="inline-flex items-center gap-1 text-2xs text-info">
                        {catching ? <Spinner size={9} /> : <CloudSyncIcon size={12} />}
                        {copy.runs.events.catchingUp}
                    </span>
                ) : events.terminal ? (
                    <span className="text-2xs text-ink-faint">{copy.runs.events.settled}</span>
                ) : (
                    <span className="inline-flex items-center gap-1 text-2xs text-ok">
                        <span aria-hidden={true} className="h-1.5 w-1.5 rounded-full bg-ok" />
                        {copy.runs.events.live}
                    </span>
                )}
            </PanelHeader>
            {entries.length === 0 ? (
                <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-4">
                    <EmptyState
                        icon={HistoryIcon}
                        title={copy.runs.events.title}
                        body={copy.empty.runEvents}
                    />
                </div>
            ) : (
                <ol className="min-h-0 flex-1 overflow-auto">
                    {entries.map((entry) => {
                        const Icon = eventIcon(entry.type);
                        const detail = detailOf(entry);
                        const path = entry.itemId === null ? undefined : paths.get(entry.itemId);
                        return (
                            <li
                                key={entry.seq}
                                className="flex gap-2 border-b border-inset px-2.5 py-1.5"
                            >
                                <Icon
                                    size={14}
                                    className={cx("mt-0.5 shrink-0", toneClasses[eventTone(entry.type)].ink)}
                                />
                                <div className="flex min-w-0 flex-1 flex-col">
                                    <div className="flex items-baseline justify-between gap-2">
                                        <span className="truncate text-xs text-ink">{eventName(entry.type)}</span>
                                        <span
                                            title={absoluteTime(entry.at)}
                                            className="shrink-0 font-mono text-2xs text-ink-faint"
                                        >
                                            {relativeTime(entry.at)}
                                        </span>
                                    </div>
                                    {detail === "" ? null : (
                                        <span className="truncate font-mono text-2xs text-ink-dim" title={detail}>
                                            {detail}
                                        </span>
                                    )}
                                    {path === undefined ? null : (
                                        <span className="truncate font-mono text-2xs text-ink-faint">{path}</span>
                                    )}
                                </div>
                            </li>
                        );
                    })}
                </ol>
            )}
            <footer className="flex shrink-0 items-center justify-between gap-2 border-t border-hairline px-2.5 py-1.5 font-mono text-2xs text-ink-faint">
                <span>{copy.runs.events.seq(events.contiguousSeq)}</span>
                <span>{copy.runs.step.noPercent}</span>
            </footer>
        </section>
    );
}
