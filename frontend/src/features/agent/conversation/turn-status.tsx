import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import type { Turn } from "../../../data/agent/turn.js";
import { duration } from "../../../domain/format.js";
import type { Tone } from "../../../ui/index.js";
import {
    CheckCircleIcon,
    cx,
    ErrorIcon,
    HistoryToggleOffIcon,
    PendingActionsIcon,
    Spinner,
    StopCircleIcon,
    toneClasses,
} from "../../../ui/index.js";

const tickMs = 1000;

function useElapsed(startedAt: number, running: boolean): number {
    const [now, setNow] = useState(() => Date.now());

    useEffect(() => {
        if (!running) {
            return;
        }
        setNow(Date.now());
        const timer = setInterval(() => {
            setNow(Date.now());
        }, tickMs);
        return () => {
            clearInterval(timer);
        };
    }, [running, startedAt]);

    return startedAt === 0 ? 0 : Math.max(0, Math.round((now - startedAt) / tickMs) * tickMs);
}

export interface TurnStatusProps {
    turn: Turn;
}

export function TurnStatus({ turn }: TurnStatusProps): ReactElement | null {
    const running = turn.status === "working" || turn.status === "awaiting-confirm" || turn.status === "stopping";
    const elapsed = useElapsed(turn.startedAt, running);

    if (turn.status === "idle") {
        return null;
    }

    let tone: Tone = "info";
    let label = copy.agent.status.working;
    let icon: ReactElement = <Spinner size={11} className="text-info" />;

    switch (turn.status) {
        case "working":
            label = elapsed === 0 ? copy.agent.status.working : copy.agent.status.workingFor(duration(elapsed));
            break;
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
            if (turn.end === "stopped") {
                tone = "muted";
                label = copy.agent.status.stopped;
                icon = <StopCircleIcon size={13} className="text-ink-faint" />;
                break;
            }
            tone = "ok";
            label = duration(Math.max(0, turn.lastEventAt - turn.startedAt));
            icon = <CheckCircleIcon size={13} className={toneClasses.ok.ink} />;
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
                {label}
            </span>
        </div>
    );
}
