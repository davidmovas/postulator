import type { InfiniteData, QueryClient } from "@tanstack/react-query";

import type { List } from "../../lib/paging.js";
import { turnStatus } from "../endpoints/agent.js";
import { keys } from "../keys.js";
import type { Message } from "../types.js";
import type { TurnReport } from "./turn.js";
import { getTurn, reconcile, settleUnreported } from "./turn.js";

const clockSlackMs = 2000;

export function answeredBy(
    messages: readonly Message[],
    assistantMessageId: string | null,
    startedAt: number,
): boolean {
    for (const held of messages) {
        if (held.role !== "assistant") {
            continue;
        }
        if (assistantMessageId !== null) {
            if (held.id === assistantMessageId) {
                return true;
            }
            continue;
        }
        const at = held.createdAt === null ? Number.NaN : Date.parse(held.createdAt);
        if (!Number.isNaN(at) && at >= startedAt - clockSlackMs) {
            return true;
        }
    }
    return false;
}

function cachedMessages(client: QueryClient, conversationId: string): Message[] {
    const entries = client.getQueriesData<InfiniteData<List<Message>>>({
        queryKey: keys.agent.messagesOf(conversationId),
    });
    const out: Message[] = [];
    for (const [, data] of entries) {
        if (data === undefined) {
            continue;
        }
        for (const page of data.pages) {
            out.push(...page.items);
        }
    }
    return out;
}

const inFlight = new Set<string>();

export async function reconcileConversation(client: QueryClient, conversationId: string): Promise<void> {
    if (conversationId === "" || inFlight.has(conversationId)) {
        return;
    }
    inFlight.add(conversationId);
    try {
        const report = await turnStatus({ conversationId }).then(
            (answered): TurnReport => ({
                running: answered.running,
                messageId: answered.messageId,
                startedAt: answered.startedAt,
                lastSeq: answered.lastSeq,
            }),
            () => null,
        );
        if (report === null) {
            return;
        }
        const before = getTurn(conversationId);
        const action = reconcile(conversationId, report);
        if (action === "adopt") {
            void client.invalidateQueries({ queryKey: keys.agent.messagesOf(conversationId) });
            return;
        }
        if (action !== "refetch") {
            return;
        }
        await client.refetchQueries({ queryKey: keys.agent.messagesOf(conversationId) }).catch(() => undefined);
        settleUnreported(
            conversationId,
            answeredBy(cachedMessages(client, conversationId), before.assistantMessageId, before.startedAt),
        );
    } finally {
        inFlight.delete(conversationId);
    }
}
