import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { Banner, Button, ErrorIcon, HistoryToggleOffIcon, RestartAltIcon, StopCircleIcon } from "../../../ui/index.js";
import { failure } from "./model/rows.js";

export interface RetryProps {
    canRetry: boolean;
    onRetry: () => void;
}

function RetryButton({ canRetry, onRetry }: RetryProps): ReactElement | null {
    if (!canRetry) {
        return null;
    }
    return (
        <Button size="sm" icon={RestartAltIcon} onClick={onRetry}>
            {copy.agent.states.retry}
        </Button>
    );
}

export interface StoppedCardProps extends RetryProps {
    detail: string;
}

export function StoppedCard({ detail, canRetry, onRetry }: StoppedCardProps): ReactElement {
    return (
        <div className="flex items-center gap-3 rounded-lg border border-dashed border-edge px-3 py-2.5">
            <StopCircleIcon size={17} className="shrink-0 text-ink-faint" />
            <div className="flex min-w-0 flex-1 flex-col">
                <span className="text-xs font-semibold text-ink-soft">{copy.agent.states.stoppedTitle}</span>
                <span className="text-2xs text-ink-faint">
                    {detail === "" ? copy.agent.states.stoppedBody : detail}
                </span>
            </div>
            <RetryButton canRetry={canRetry} onRetry={onRetry} />
        </div>
    );
}

export interface FailedCardProps extends RetryProps {
    code: string;
    message: string;
}

export function FailedCard({ code, message, canRetry, onRetry }: FailedCardProps): ReactElement {
    const told = failure(code, message);
    return (
        <Banner
            tone={told.spent ? "warn" : "danger"}
            icon={told.spent ? HistoryToggleOffIcon : ErrorIcon}
            title={told.title}
            body={told.body}
            actions={<RetryButton canRetry={canRetry} onRetry={onRetry} />}
        />
    );
}

export function LostCard({ canRetry, onRetry }: RetryProps): ReactElement {
    return (
        <Banner
            tone="warn"
            icon={HistoryToggleOffIcon}
            title={copy.agent.states.lostTitle}
            body={copy.agent.states.lostBody}
            actions={<RetryButton canRetry={canRetry} onRetry={onRetry} />}
        />
    );
}
