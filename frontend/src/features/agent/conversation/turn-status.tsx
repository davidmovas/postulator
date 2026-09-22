import type { ReactElement, ReactNode } from "react";

import { copy } from "../../../copy/index.js";
import type { Turn } from "../../../data/agent/turn.js";
import { isActive } from "../../../data/agent/turn.js";
import type { Tone } from "../../../ui/index.js";
import {
    cx,
    ErrorIcon,
    HistoryToggleOffIcon,
    PendingActionsIcon,
    Spinner,
    StopCircleIcon,
    toneClasses,
} from "../../../ui/index.js";

export interface TurnStatusProps {
    turn: Turn;
}

function waitingSeconds(afterMs: number): number {
    return Math.max(1, Math.round(afterMs / 1000));
}

export function TurnStatus({ turn }: TurnStatusProps): ReactElement | null {
    const held = turn.waiting;
    if (held !== null && isActive(turn.status)) {
        return (
            <Pill tone="warn" icon={<Spinner size={11} className={toneClasses.warn.ink} />}>
                {copy.agent.status.waiting(waitingSeconds(held.afterMs))}
            </Pill>
        );
    }
    if (turn.status === "idle" || turn.status === "working" || (turn.status === "done" && turn.end === "answered")) {
        return null;
    }

    let tone: Tone = "info";
    let label = copy.agent.status.working;
    let icon: ReactElement = <Spinner size={11} className="text-info" />;

    switch (turn.status) {
        case "awaiting-confirm":
            tone = "warn";
            label = copy.agent.status.awaiting;
            icon = <PendingActionsIcon size={13} className={toneClasses.warn.ink} />;
            break;
        case "stopping":
            tone = "muted";
            label = copy.agent.status.stopping;
            break;
        case "done":
            tone = "muted";
            label = copy.agent.status.stopped;
            icon = <StopCircleIcon size={13} className="text-ink-faint" />;
            break;
        default:
            tone = "danger";
            label = turn.end === "lost" ? copy.agent.status.lost : copy.agent.status.failed;
            icon = <ErrorIcon size={13} className={toneClasses.danger.ink} />;
            break;
    }

    if (turn.status === "error" && turn.end === "lost") {
        icon = <HistoryToggleOffIcon size={13} className={toneClasses.danger.ink} />;
    }

    return (
        <Pill tone={tone} icon={icon}>
            {label}
        </Pill>
    );
}

interface PillProps {
    tone: Tone;
    icon: ReactElement;
    children: ReactNode;
}

function Pill({ tone, icon, children }: PillProps): ReactElement {
    return (
        <div className="flex h-5 items-center gap-2">
            <span
                className={cx(
                    "inline-flex h-5 shrink-0 items-center gap-1.5 rounded-full border px-2 text-2xs font-medium",
                    toneClasses[tone].border,
                    toneClasses[tone].soft,
                    toneClasses[tone].ink,
                )}
            >
                {icon}
                {children}
            </span>
        </div>
    );
}
