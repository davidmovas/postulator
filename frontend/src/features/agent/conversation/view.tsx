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
    useSendMessage,
    useSetConversationMode,
    useTranscript,
    useTurnReconciler,
} from "../../../data/hooks/agent.js";
import { useRoleProfiles, useUsage } from "../../../data/hooks/models.js";
import { useSite } from "../../../data/hooks/sites.js";
import { useTools } from "../../../data/hooks/tools.js";
import type { Conversation } from "../../../data/types.js";
import { Banner, Button, EmptyState, SkeletonRows, SmartToyIcon } from "../../../ui/index.js";
import type { CardBusy } from "../cards/card.js";
import { Composer } from "./composer.js";
import { ConversationMeta } from "./meta.js";
import { lastUserText, rows as transcriptRows } from "./model/transcript.js";
import { Transcript } from "./transcript.js";
import { TurnStatus } from "./turn-status.js";

function messageOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;
}

interface ComposerBridge {
    prefillSeq: number;
    takePrefill: () => string | null;
}

interface StartPaneProps extends ComposerBridge {
    siteId: string | null;
    starting: boolean;
    startError: unknown;
    onStart: (text: string) => void;
}

function StartPane({ siteId, starting, startError, prefillSeq, takePrefill, onStart }: StartPaneProps): ReactElement {
    const profiles = useRoleProfiles(siteId ?? undefined);
    const chat = (profiles.data?.profiles ?? []).find((profile) => profile.role === "chat")?.effective ?? null;

    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="flex flex-1 items-center justify-center p-4">
                <EmptyState icon={SmartToyIcon} title={copy.agent.states.start} body={copy.agent.states.startBody} />
            </div>
            <Composer
                answering={false}
                stopping={false}
                disabled={starting}
                placeholder={copy.agent.composer.firstPlaceholder}
                error={
                    messageOf(startError) ??
                    (chat === null && profiles.data !== undefined ? copy.agent.composer.noModel : null)
                }
                status={null}
                prefillSeq={prefillSeq}
                takePrefill={takePrefill}
                onSend={onStart}
                onStop={() => undefined}
            />
        </div>
    );
}

interface LivePaneProps extends ComposerBridge {
    conversation: Conversation;
    onOpenTools: () => void;
}

function LivePane({ conversation, prefillSeq, takePrefill, onOpenTools }: LivePaneProps): ReactElement {
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
    const setMode = useSetConversationMode();
    const [settling, setSettling] = useState<{ id: string; busy: CardBusy } | null>(null);

    useTurnReconciler(conversationId);

    const pendingActions = useMemo(() => flatten(pending.data?.pages), [pending.data]);
    const rows = useMemo(
        () => transcriptRows(transcript.messages, turn, pendingActions),
        [transcript.messages, turn, pendingActions],
    );
    const asked = useMemo(() => lastUserText(rows), [rows]);

    const chat = (profiles.data?.profiles ?? []).find((profile) => profile.role === "chat")?.effective ?? null;
    const answering = turn.status === "working" || turn.status === "awaiting-confirm" || turn.status === "stopping";

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

    const earlier = transcript.canLoadEarlier ? (
        <Button
            variant="ghost"
            size="sm"
            className="self-center"
            busy={transcript.loadingEarlier}
            onClick={transcript.loadEarlier}
        >
            {copy.agent.transcript.loadEarlier}
        </Button>
    ) : null;

    return (
        <div className="flex h-full min-h-0 flex-col">
            <ConversationMeta
                conversation={conversation}
                siteName={site.data?.site.name ?? null}
                spend={usage.data === undefined ? null : { usd: usage.data.usd, calls: usage.data.calls }}
                model={chat === null ? null : `${chat.provider}/${chat.model}`}
                toolCount={tools.data?.tools?.length ?? null}
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
                    canRetry={asked !== null}
                    header={earlier}
                    onApprove={(actionId) => {
                        settle(actionId, true);
                    }}
                    onReject={(actionId) => {
                        settle(actionId, false);
                    }}
                    onRetry={() => {
                        if (asked !== null) {
                            ask(asked);
                        }
                    }}
                />
            )}
            <Composer
                answering={answering}
                stopping={turn.status === "stopping"}
                disabled={send.isPending}
                placeholder={copy.agent.composer.placeholder}
                error={
                    messageOf(send.error) ??
                    messageOf(confirm.error) ??
                    (chat === null && profiles.data !== undefined ? copy.agent.composer.noModel : null)
                }
                status={<TurnStatus turn={turn} />}
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

export interface ConversationViewProps extends ComposerBridge {
    conversation: Conversation | null;
    siteId: string | null;
    starting: boolean;
    startError: unknown;
    onStart: (text: string) => void;
    onOpenTools: () => void;
}

export function ConversationView({
    conversation,
    siteId,
    starting,
    startError,
    prefillSeq,
    takePrefill,
    onStart,
    onOpenTools,
}: ConversationViewProps): ReactElement {
    if (conversation === null) {
        return (
            <StartPane
                siteId={siteId}
                starting={starting}
                startError={startError}
                prefillSeq={prefillSeq}
                takePrefill={takePrefill}
                onStart={onStart}
            />
        );
    }
    return (
        <LivePane
            key={conversation.id}
            conversation={conversation}
            prefillSeq={prefillSeq}
            takePrefill={takePrefill}
            onOpenTools={onOpenTools}
        />
    );
}
