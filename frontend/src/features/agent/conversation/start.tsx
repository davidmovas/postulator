import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useCreateConversation, useSendMessage } from "../../../data/hooks/agent.js";
import type { Conversation } from "../../../data/types.js";
import { EmptyState, SmartToyIcon } from "../../../ui/index.js";
import { takePrefill } from "../dock-state.js";
import { Composer } from "./composer.js";

export interface StartConversationProps {
    siteId: string | null;
    prefillSeq: number;
    onStarted: (conversation: Conversation) => void;
}

export function StartConversation({ siteId, prefillSeq, onStarted }: StartConversationProps): ReactElement {
    const create = useCreateConversation();
    const send = useSendMessage();
    const thrown = create.error ?? send.error;
    const reaction = thrown === null ? null : react(thrown);
    const error = reaction === null || reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;

    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="flex flex-1 items-center justify-center p-4">
                <EmptyState icon={SmartToyIcon} title={copy.agent.states.start} body={copy.agent.states.startBody} />
            </div>
            <Composer
                answering={false}
                disabled={create.isPending || send.isPending}
                placeholder={copy.agent.composer.firstPlaceholder}
                error={error}
                prefillSeq={prefillSeq}
                takePrefill={takePrefill}
                onSend={(text) => {
                    create.mutate(
                        { siteId: siteId ?? undefined, mode: "confirm" },
                        {
                            onSuccess: (answered) => {
                                onStarted(answered.conversation);
                                send.mutate({ conversationId: answered.conversation.id, text });
                            },
                        },
                    );
                }}
                onStop={() => undefined}
            />
        </div>
    );
}
