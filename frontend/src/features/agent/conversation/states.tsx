import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { Banner, Button, ErrorIcon, HistoryToggleOffIcon, RestartAltIcon, StopCircleIcon } from "../../../ui/index.js";

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

export function CancelledCard({ canRetry, onRetry }: RetryProps): ReactElement {
    return (
        <div className="flex items-center gap-3 rounded-lg border border-dashed border-edge px-3 py-2.5">
            <StopCircleIcon size={17} className="shrink-0 text-ink-faint" />
            <div className="flex min-w-0 flex-1 flex-col">
                <span className="text-xs font-semibold text-ink-soft">{copy.agent.states.cancelledTitle}</span>
                <span className="text-2xs text-ink-faint">{copy.agent.states.cancelledBody}</span>
            </div>
            <RetryButton canRetry={canRetry} onRetry={onRetry} />
        </div>
    );
}

export interface ErrorCardProps extends RetryProps {
    message: string;
}

export function ErrorCard({ message, canRetry, onRetry }: ErrorCardProps): ReactElement {
    return (
        <Banner
            tone="danger"
            icon={ErrorIcon}
            title={copy.agent.states.errorTitle}
            body={message}
            actions={<RetryButton canRetry={canRetry} onRetry={onRetry} />}
        />
    );
}

export function StalledCard({ canRetry, onRetry }: RetryProps): ReactElement {
    return (
        <Banner
            tone="warn"
            icon={HistoryToggleOffIcon}
            title={copy.agent.states.stalledTitle}
            body={copy.agent.turnStalled}
            actions={<RetryButton canRetry={canRetry} onRetry={onRetry} />}
        />
    );
}
