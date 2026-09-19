import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { useAgentTurn } from "../../../data/agent/use-agent-turn.js";
import { flatten } from "../../../data/call.js";
import { react } from "../../../data/errors.js";
import {
    useCancelTurn,
    useConfirmAction,
    usePendingActions,
    useRenameConversation,
    useSendMessage,
    useSetConversationMode,
    useTranscript,
} from "../../../data/hooks/agent.js";
import { useRoleProfiles, useUsage } from "../../../data/hooks/models.js";
import { useSite } from "../../../data/hooks/sites.js";
import { useTools } from "../../../data/hooks/tools.js";
import type { Conversation } from "../../../data/types.js";
import { Banner, SkeletonRows } from "../../../ui/index.js";
import type { CardBusy } from "../cards/card.js";
import { Composer } from "./composer.js";
import { ConversationHeader } from "./header.js";
import { rows as transcriptRows } from "./model/transcript.js";
import { Transcript } from "./transcript.js";

function messageOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;
}

export interface ConversationViewProps {
    conversation: Conversation;
    prefillSeq: number;
    takePrefill: () => string | null;
    onOpenTools: () => void;
}

export function ConversationView({ conversation, prefillSeq, takePrefill, onOpenTools }: ConversationViewProps): ReactElement {
    const conversationId = conversation.id;
    const siteId = conversation.siteId;
    const transcript = useTranscript(conversationId);
    const turn = useAgentTurn(conversationId);
    const pending = usePendingActions({ conversationId, status: "pending" }, 100);
    const site = useSite(siteId);
    const usage = useUsage({ conversationId });
    const profiles = useRoleProfiles(siteId ?? undefined);
    const tools = useTools();
    const send = useSendMessage();
    const cancel = useCancelTurn();
    const confirm = useConfirmAction();
    const rename = useRenameConversation();
    const setMode = useSetConversationMode();
    const [settling, setSettling] = useState<{ id: string; busy: CardBusy } | null>(null);

    const pendingActions = useMemo(() => flatten(pending.data?.pages), [pending.data]);
    const rows = useMemo(
        () => transcriptRows(transcript.messages, turn, pendingActions),
        [transcript.messages, turn, pendingActions],
    );
    const lastUserText = useMemo(() => {
        for (let index = rows.length - 1; index >= 0; index -= 1) {
            const row = rows[index];
            if (row !== undefined && row.kind === "user") {
                return row.text;
            }
        }
        return null;
    }, [rows]);

    const chat = (profiles.data?.profiles ?? []).find((profile) => profile.role === "chat")?.effective ?? null;
    const answering = turn.status === "streaming" || turn.status === "awaiting-confirm";

    const ask = (text: string): void => {
        send.mutate({ conversationId, text });
    };

    const settle = (actionId: string, approve: boolean): void => {
        setSettling({ id: actionId, busy: approve ? "approve" : "reject" });
        confirm.mutate(
            { actionId, approve },
            {
                onSettled: () => {
                    setSettling(null);
                },
            },
        );
    };

    return (
        <div className="flex h-full min-h-0 flex-col">
            <ConversationHeader
                conversation={conversation}
                siteName={site.data?.site.name ?? null}
                spend={usage.data === undefined ? null : { usd: usage.data.usd, calls: usage.data.calls }}
                model={chat === null ? null : `${chat.provider}/${chat.model}`}
                toolCount={tools.data?.tools?.length ?? null}
                renaming={rename.isPending}
                onRename={(title) => {
                    rename.mutate({ conversationId, title });
                }}
                onModeChange={(mode) => {
                    setMode.mutate({ conversationId, mode });
                }}
                onOpenTools={onOpenTools}
            />
            {transcript.error !== null && messageOf(transcript.error) !== null ? (
                <div className="px-3 pt-3">
                    <Banner tone="danger" title={messageOf(transcript.error) ?? ""} />
                </div>
            ) : null}
            {transcript.isPending ? (
                <div className="flex-1 p-3">
                    <SkeletonRows rows={6} label={copy.agent.states.loading} />
                </div>
            ) : (
                <Transcript
                    rows={rows}
                    settling={settling}
                    lastUserText={lastUserText}
                    onApprove={(actionId) => {
                        settle(actionId, true);
                    }}
                    onReject={(actionId) => {
                        settle(actionId, false);
                    }}
                    onRetry={() => {
                        if (lastUserText !== null) {
                            ask(lastUserText);
                        }
                    }}
                />
            )}
            <Composer
                answering={answering}
                disabled={send.isPending}
                placeholder={copy.agent.composer.placeholder}
                error={messageOf(send.error) ?? messageOf(confirm.error) ?? (chat === null && profiles.data !== undefined ? copy.agent.composer.noModel : null)}
                prefillSeq={prefillSeq}
                takePrefill={takePrefill}
                onSend={ask}
                onStop={() => {
                    cancel.mutate({ conversationId });
                }}
            />
        </div>
    );
}
