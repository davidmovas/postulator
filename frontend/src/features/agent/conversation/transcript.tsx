import type { ReactElement, UIEvent } from "react";
import { useEffect, useLayoutEffect, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { absoluteTime, tokens, usd } from "../../../domain/format.js";
import type { TurnUsage } from "../../../data/agent/turn.js";
import type { Timestamp } from "../../../data/wire.js";
import { ArrowDownwardIcon, Button, cx, SmartToyIcon, Spinner, toneClasses } from "../../../ui/index.js";
import type { CardBusy } from "../cards/card.js";
import { ConfirmationCard } from "../cards/card.js";
import { outcomeOf } from "../cards/outcome.js";
import { familyIcon, toolStatusLabel, toolStatusTone } from "../labels.js";
import { familyOf, resultSummary, verbOf } from "./model/tools.js";
import type { Row } from "./model/transcript.js";
import { CancelledCard, ErrorCard, StalledCard } from "./states.js";

const bottomSlackPx = 32;

interface KickerProps {
    at: Timestamp;
}

function AssistantKicker({ at }: KickerProps): ReactElement {
    return (
        <div className="flex items-center gap-1.5">
            <SmartToyIcon size={14} className="text-accent" />
            <span className="text-2xs font-semibold text-ink-dim">{copy.agent.assistant}</span>
            {at === null ? null : (
                <span className="font-mono text-2xs text-ink-faint" title={absoluteTime(at)}>
                    {absoluteTime(at)}
                </span>
            )}
        </div>
    );
}

interface UsageLineProps {
    usage: TurnUsage;
}

function UsageLine({ usage }: UsageLineProps): ReactElement {
    return (
        <span className="font-mono text-2xs text-ink-faint">
            {copy.agent.transcript.usage(tokens(usage.inputTokens), tokens(usage.outputTokens), usd(usage.usd))}
        </span>
    );
}

interface ToolRowProps {
    row: Extract<Row, { kind: "tool" }>;
}

function ToolRow({ row }: ToolRowProps): ReactElement {
    const [open, setOpen] = useState(false);
    const family = familyOf(row.tool);
    const Icon = familyIcon(family);
    const tone = toolStatusTone(row.status);
    const summary = row.status === "running" ? null : row.error ?? resultSummary(row.tool, row.result);

    return (
        <div className="flex flex-col rounded-md border border-hairline bg-inset">
            <button
                type="button"
                disabled={summary === null}
                onClick={() => {
                    setOpen((held) => !held);
                }}
                className="flex h-6 w-full items-center gap-2 px-2 text-left disabled:cursor-default"
            >
                {row.status === "running" ? (
                    <Spinner size={12} className="shrink-0 text-info" />
                ) : (
                    <Icon size={14} className={cx("shrink-0", toneClasses[tone].ink)} />
                )}
                <span className="min-w-0 flex-1 truncate font-mono text-xs text-ink-soft">{verbOf(row.tool)}</span>
                <span className={cx("shrink-0 text-2xs font-semibold tracking-label uppercase", toneClasses[tone].ink)}>
                    {toolStatusLabel(row.status)}
                </span>
                {row.durationMs === null ? null : (
                    <span className="w-12 shrink-0 text-right font-mono text-2xs text-ink-faint">
                        {copy.agent.transcript.tool.duration(row.durationMs)}
                    </span>
                )}
            </button>
            {open && summary !== null ? (
                <div className={cx("border-t border-hairline px-2 py-1.5 text-xs", row.error === null ? "text-ink-dim" : "text-danger")}>
                    {summary}
                </div>
            ) : null}
        </div>
    );
}

export interface TranscriptProps {
    rows: readonly Row[];
    settling: { id: string; busy: CardBusy } | null;
    lastUserText: string | null;
    onApprove: (actionId: string) => void;
    onReject: (actionId: string) => void;
    onRetry: () => void;
}

export function Transcript({ rows, settling, lastUserText, onApprove, onReject, onRetry }: TranscriptProps): ReactElement {
    const scroller = useRef<HTMLDivElement>(null);
    const [pinned, setPinned] = useState(true);
    const lastRowId = rows[rows.length - 1]?.id ?? "";
    const lastText = rows[rows.length - 1]?.kind === "streaming" ? (rows[rows.length - 1] as Extract<Row, { kind: "streaming" }>).text.length : 0;

    useLayoutEffect(() => {
        const held = scroller.current;
        if (held !== null && pinned) {
            held.scrollTop = held.scrollHeight;
        }
    }, [pinned, rows.length, lastRowId, lastText]);

    useEffect(() => {
        const held = scroller.current;
        if (held !== null) {
            held.scrollTop = held.scrollHeight;
        }
    }, []);

    const scrolled = (event: UIEvent<HTMLDivElement>): void => {
        const held = event.currentTarget;
        setPinned(held.scrollHeight - held.scrollTop - held.clientHeight <= bottomSlackPx);
    };

    const canRetry = lastUserText !== null;

    return (
        <div className="relative min-h-0 flex-1">
            <div ref={scroller} onScroll={scrolled} className="flex h-full flex-col gap-3 overflow-y-auto px-3 py-3">
                {rows.map((row) => {
                    switch (row.kind) {
                        case "user":
                            return (
                                <div key={row.id} className="flex justify-end">
                                    <div
                                        className="max-w-[88%] rounded-lg rounded-br-sm border border-hairline bg-raised px-3 py-2 text-sm whitespace-pre-wrap text-ink"
                                        title={absoluteTime(row.at)}
                                    >
                                        {row.text}
                                    </div>
                                </div>
                            );
                        case "assistant":
                            return (
                                <div key={row.id} className="flex flex-col gap-1.5">
                                    <AssistantKicker at={row.at} />
                                    <p className="text-sm leading-relaxed whitespace-pre-wrap text-ink-soft">{row.text}</p>
                                    {row.usage === null ? null : <UsageLine usage={row.usage} />}
                                </div>
                            );
                        case "streaming":
                            return (
                                <div key={row.id} className="flex flex-col gap-1.5">
                                    <AssistantKicker at={null} />
                                    <p className="text-sm leading-relaxed whitespace-pre-wrap text-ink-soft">
                                        {row.text}
                                        <span aria-hidden={true} className="ml-0.5 inline-block h-3.5 w-0.5 animate-caret bg-accent align-middle" />
                                    </p>
                                </div>
                            );
                        case "working":
                            return (
                                <div key={row.id} className="flex items-center gap-2 text-xs text-ink-faint">
                                    <Spinner size={11} className="text-accent" />
                                    {copy.agent.thinking}
                                </div>
                            );
                        case "tool":
                            return <ToolRow key={row.id} row={row} />;
                        case "confirm": {
                            const tool = row.action?.tool ?? row.live?.tool ?? "";
                            const risk = row.live?.risk ?? "write";
                            return (
                                <ConfirmationCard
                                    key={row.id}
                                    tool={tool}
                                    risk={risk}
                                    status={row.action?.status ?? "pending"}
                                    createdAt={row.action?.createdAt ?? null}
                                    outcome={row.action === null ? null : outcomeOf(row.action)}
                                    busy={settling !== null && settling.id === row.id ? settling.busy : null}
                                    focus={true}
                                    onApprove={() => {
                                        onApprove(row.id);
                                    }}
                                    onReject={() => {
                                        onReject(row.id);
                                    }}
                                />
                            );
                        }
                        case "cancelled":
                            return <CancelledCard key={row.id} canRetry={canRetry} onRetry={onRetry} />;
                        case "error":
                            return <ErrorCard key={row.id} message={row.message} canRetry={canRetry} onRetry={onRetry} />;
                        default:
                            return <StalledCard key={row.id} canRetry={canRetry} onRetry={onRetry} />;
                    }
                })}
            </div>
            {pinned ? null : (
                <div className="pointer-events-none absolute inset-x-0 bottom-2 flex justify-center">
                    <Button
                        size="sm"
                        icon={ArrowDownwardIcon}
                        className="pointer-events-auto"
                        onClick={() => {
                            setPinned(true);
                        }}
                    >
                        {copy.agent.transcript.jumpToLatest}
                    </Button>
                </div>
            )}
        </div>
    );
}
