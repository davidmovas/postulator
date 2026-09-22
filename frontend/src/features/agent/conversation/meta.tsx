import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import type { Conversation } from "../../../data/types.js";
import { usd } from "../../../domain/format.js";
import {
    BoltIcon,
    BuildIcon,
    cx,
    Dialog,
    PublicIcon,
    Segmented,
    Stars2Icon,
} from "../../../ui/index.js";
import type { Spend } from "./model/spend.js";
import { cachedPercent } from "./model/spend.js";

const modes = [
    { value: "confirm", label: copy.agent.header.modeConfirm },
    { value: "autonomous", label: copy.agent.header.modeAutonomous },
];

export interface ConversationMetaProps {
    conversation: Conversation;
    siteName: string | null;
    spend: Spend | null;
    model: string | null;
    toolCount: number | null;
    onModeChange: (mode: string) => void;
    onOpenTools: () => void;
}

export function ConversationMeta({
    conversation,
    siteName,
    spend,
    model,
    toolCount,
    onModeChange,
    onOpenTools,
}: ConversationMetaProps): ReactElement {
    const [confirming, setConfirming] = useState(false);
    const autonomous = conversation.mode === "autonomous";

    return (
        <div className="flex shrink-0 flex-col gap-2 border-b border-hairline px-3 py-2">
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-ink-faint">
                <span className="flex items-center gap-1">
                    <PublicIcon size={12} />
                    {siteName ?? copy.agent.header.everywhere}
                </span>
                <span className={cx("flex items-center gap-1", model === null && "text-warn")}>
                    <Stars2Icon size={12} />
                    <span className="font-mono">{model ?? copy.agent.header.modelUnset}</span>
                </span>
                <span className="font-mono" title={copy.agent.header.spendTooltip}>
                    {spend === null || spend.calls === 0
                        ? copy.agent.header.noSpend
                        : copy.agent.header.spend(usd(spend.usd), spend.calls)}
                </span>
                {spend === null || spend.cachedInput === 0 ? null : (
                    <span className="font-mono text-ink-dim" title={copy.agent.header.spendTooltip}>
                        {copy.agent.header.cached(cachedPercent(spend))}
                    </span>
                )}
                {toolCount === null ? null : (
                    <button
                        type="button"
                        onClick={onOpenTools}
                        className="flex items-center gap-1 text-ink-dim hover:text-ink"
                    >
                        <BuildIcon size={12} />
                        {copy.agent.header.tools(toolCount)}
                    </button>
                )}
                <div className="ml-auto">
                    <Segmented
                        label={copy.agent.header.modeLabel}
                        size="sm"
                        options={modes}
                        value={conversation.mode}
                        onValueChange={(mode) => {
                            if (mode === "autonomous") {
                                setConfirming(true);
                                return;
                            }
                            onModeChange(mode);
                        }}
                    />
                </div>
            </div>
            {autonomous ? (
                <div className="flex items-start gap-2 rounded-md border border-danger-border bg-danger-soft px-2.5 py-2">
                    <BoltIcon size={15} className="mt-px shrink-0 text-danger" />
                    <div className="flex min-w-0 flex-col">
                        <span className="text-xs font-semibold text-danger">{copy.agent.autonomousTitle}</span>
                        <span className="text-2xs text-ink-soft">{copy.agent.autonomousBody}</span>
                    </div>
                </div>
            ) : null}
            <Dialog
                open={confirming}
                onOpenChange={setConfirming}
                title={copy.agent.autonomousDialog.title}
                description={copy.agent.autonomousDialog.body}
                confirmLabel={copy.agent.autonomousDialog.confirm}
                cancelLabel={copy.agent.autonomousDialog.cancel}
                destructive={true}
                icon={BoltIcon}
                onConfirm={() => {
                    setConfirming(false);
                    onModeChange("autonomous");
                }}
            />
        </div>
    );
}
