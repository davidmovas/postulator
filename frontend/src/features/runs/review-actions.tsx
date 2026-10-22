import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { Button, Kbd, OpenInNewIcon, RefreshIcon, RestartAltIcon, SmartToyIcon, TaskAltIcon } from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import type { RetryState } from "./authority.js";
import type { ItemAction, RegenerateState } from "./hold.js";
import { retryBlockedText } from "./labels.js";

export interface ReviewActionsProps {
    path: string;
    runId: string;
    itemId: string;
    pageId: string;
    blocked: string | null;
    state: RetryState | null;
    primary: ItemAction | null;
    busy: boolean;
    regeneration: RegenerateState | null;
    regenerating: boolean;
    review: boolean;
    nothingTicked: boolean;
    onOpenPage: (pageId: string) => void;
    onRerun: (pageId: string) => void;
    onRetry: () => void;
    onAccept: () => void;
    onRegenerate: () => void;
}

const regenerateRefusals: Readonly<Record<Exclude<RegenerateState["kind"], "ready">, string>> = {
    busy: copy.runs.regenerateBusy,
    published: copy.runs.regeneratePublished,
};

function acceptTitle(review: boolean, nothingTicked: boolean): string {
    if (nothingTicked) {
        return copy.runs.review.plan.nothingTicked;
    }
    return review ? copy.runs.acceptReviewHint : copy.runs.acceptHint;
}

export function ReviewActions({
    path,
    runId,
    itemId,
    pageId,
    blocked,
    state,
    primary,
    busy,
    regeneration,
    regenerating,
    review,
    nothingTicked,
    onOpenPage,
    onRerun,
    onRetry,
    onAccept,
    onRegenerate,
}: ReviewActionsProps): ReactElement {
    const retryReady = state !== null && state.kind === "ready";
    return (
        <div className="flex w-full items-center justify-between gap-2">
            <span className="flex shrink-0 items-center gap-1.5 text-2xs text-ink-faint">
                <Kbd keys={["J", "K"]} />
                {copy.runs.review.moveItems}
            </span>
            <div className="flex shrink-0 gap-2">
                <Button
                    size="sm"
                    icon={OpenInNewIcon}
                    onClick={() => {
                        onOpenPage(pageId);
                    }}
                >
                    {copy.runs.review.openPage}
                </Button>
                <Button
                    size="sm"
                    variant="ghost"
                    icon={SmartToyIcon}
                    onClick={() => {
                        askAgent(copy.agent.ask.runItem(path, runId, itemId));
                    }}
                >
                    {copy.agent.askAbout}
                </Button>
                {blocked === null && regeneration?.kind !== "published" ? null : (
                    <Button
                        size="sm"
                        onClick={() => {
                            onRerun(pageId);
                        }}
                    >
                        {copy.runs.rerunPage}
                    </Button>
                )}
                <Button
                    size="sm"
                    data-item-regenerate={true}
                    variant={primary === "regenerate" ? "primary" : undefined}
                    icon={RefreshIcon}
                    disabled={regeneration === null || regeneration.kind !== "ready"}
                    busy={regenerating}
                    title={
                        regeneration === null || regeneration.kind === "ready"
                            ? copy.runs.regenerateHint
                            : regenerateRefusals[regeneration.kind]
                    }
                    onClick={onRegenerate}
                >
                    {copy.runs.regenerate}
                </Button>
                <Button
                    size="sm"
                    data-item-retry={true}
                    variant={primary === "retry" ? "primary" : undefined}
                    icon={RestartAltIcon}
                    disabled={!retryReady}
                    busy={busy && primary !== "accept"}
                    title={
                        state !== null && state.kind === "busy"
                            ? copy.runs.retryBusy
                            : blocked === null
                              ? undefined
                              : retryBlockedText(blocked)
                    }
                    onClick={onRetry}
                >
                    {copy.runs.retryStep}
                </Button>
                {primary === "accept" ? (
                    <Button
                        size="sm"
                        data-item-accept={true}
                        variant="primary"
                        icon={TaskAltIcon}
                        disabled={!retryReady || nothingTicked}
                        busy={busy}
                        title={acceptTitle(review, nothingTicked)}
                        onClick={onAccept}
                    >
                        {review ? copy.runs.acceptReview : copy.runs.accept}
                    </Button>
                ) : null}
            </div>
        </div>
    );
}
