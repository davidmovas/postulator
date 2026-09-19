import type { KeyboardEvent, ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import type { Conversation } from "../../../data/types.js";
import { usd } from "../../../domain/format.js";
import { BoltIcon, BuildIcon, cx, Dialog, EditIcon, Input, PublicIcon, Stars2Icon, Switch } from "../../../ui/index.js";

export interface ConversationHeaderProps {
    conversation: Conversation;
    siteName: string | null;
    spend: { usd: number; calls: number } | null;
    model: string | null;
    toolCount: number | null;
    renaming: boolean;
    onRename: (title: string) => void;
    onModeChange: (mode: string) => void;
    onOpenTools: () => void;
}

export function ConversationHeader({
    conversation,
    siteName,
    spend,
    model,
    toolCount,
    renaming,
    onRename,
    onModeChange,
    onOpenTools,
}: ConversationHeaderProps): ReactElement {
    const [editing, setEditing] = useState<string | null>(null);
    const [confirming, setConfirming] = useState(false);
    const autonomous = conversation.mode === "autonomous";
    const title = conversation.title === "" ? copy.agent.untitled : conversation.title;

    useEffect(() => {
        if (!renaming) {
            setEditing(null);
        }
    }, [renaming, conversation.title]);

    const commit = (): void => {
        if (editing !== null && editing.trim() !== "" && editing.trim() !== conversation.title) {
            onRename(editing.trim());
        }
        setEditing(null);
    };

    const keyed = (event: KeyboardEvent<HTMLInputElement>): void => {
        if (event.key === "Enter") {
            event.preventDefault();
            commit();
        } else if (event.key === "Escape") {
            event.preventDefault();
            setEditing(null);
        }
    };

    return (
        <div className="flex shrink-0 flex-col gap-1.5 border-b border-hairline px-3 py-2">
            <div className="flex items-center gap-2">
                {editing === null ? (
                    <button
                        type="button"
                        title={copy.agent.header.rename}
                        onClick={() => {
                            setEditing(conversation.title);
                        }}
                        className="group flex min-w-0 flex-1 items-center gap-1.5 text-left"
                    >
                        <span className={cx("truncate text-sm font-semibold", conversation.title === "" ? "text-ink-dim" : "text-ink")}>
                            {title}
                        </span>
                        <EditIcon size={13} className="shrink-0 text-ink-faint opacity-0 transition-opacity group-hover:opacity-100" />
                    </button>
                ) : (
                    <Input
                        autoFocus={true}
                        aria-label={copy.agent.header.renameTitle}
                        value={editing}
                        maxLength={120}
                        onChange={(event) => {
                            setEditing(event.target.value);
                        }}
                        onKeyDown={keyed}
                        onBlur={commit}
                        className="h-6 flex-1"
                    />
                )}
                <Switch
                    tone="danger"
                    label={copy.agent.header.modeSwitch}
                    checked={autonomous}
                    className="shrink-0 text-xs"
                    onChange={(event) => {
                        if (event.target.checked) {
                            setConfirming(true);
                        } else {
                            onModeChange("confirm");
                        }
                    }}
                />
            </div>
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-ink-faint">
                <span className="flex items-center gap-1">
                    <PublicIcon size={12} />
                    {siteName ?? copy.agent.header.everywhere}
                </span>
                <span className={cx("flex items-center gap-1", model === null && "text-warn")}>
                    <Stars2Icon size={12} />
                    <span className="font-mono">{model ?? copy.agent.header.modelUnset}</span>
                </span>
                <span className="font-mono">
                    {spend === null || spend.calls === 0 ? copy.agent.header.noSpend : copy.agent.header.spend(usd(spend.usd), spend.calls)}
                </span>
                {toolCount === null ? null : (
                    <button
                        type="button"
                        onClick={onOpenTools}
                        className="ml-auto flex items-center gap-1 text-ink-dim hover:text-ink"
                    >
                        <BuildIcon size={12} />
                        {copy.agent.header.tools(toolCount)}
                    </button>
                )}
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
