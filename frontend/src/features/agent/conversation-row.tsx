import type { KeyboardEvent, ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Conversation } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { CountBadge, cx, DeleteIcon, IconButton } from "../../ui/index.js";

export interface ConversationRowProps {
    conversation: Conversation;
    active: boolean;
    pending: number;
    onOpen: () => void;
    onDelete: () => void;
}

export function ConversationRow({ conversation, active, pending, onOpen, onDelete }: ConversationRowProps): ReactElement {
    const keyed = (event: KeyboardEvent<HTMLDivElement>): void => {
        if (event.key === "Enter") {
            event.preventDefault();
            onOpen();
        }
    };
    return (
        <div
            role="button"
            tabIndex={0}
            onClick={onOpen}
            onKeyDown={keyed}
            className={cx(
                "group flex items-center gap-2 rounded-md px-2 py-1.5 text-left",
                active ? "bg-accent-soft" : "hover:bg-inset",
            )}
        >
            <span
                aria-hidden={true}
                className={cx(
                    "h-1.5 w-1.5 shrink-0 rounded-full",
                    conversation.mode === "autonomous" ? "bg-danger" : "bg-ink-faint",
                )}
            />
            <div className="flex min-w-0 flex-1 flex-col">
                <span
                    className={cx(
                        "truncate text-xs",
                        conversation.title === ""
                            ? "text-ink-dim"
                            : active
                              ? "font-semibold text-ink"
                              : "text-ink-soft",
                    )}
                >
                    {conversation.title === "" ? copy.agent.untitled : conversation.title}
                </span>
                <span className="font-mono text-2xs text-ink-faint" title={absoluteTime(conversation.createdAt)}>
                    {relativeTime(conversation.createdAt)}
                </span>
            </div>
            {pending > 0 ? <CountBadge tone="warn" count={pending} /> : null}
            <IconButton
                icon={DeleteIcon}
                label={copy.agent.screen.delete}
                variant="ghost"
                size="sm"
                className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                onClick={(event) => {
                    event.stopPropagation();
                    onDelete();
                }}
            />
        </div>
    );
}
