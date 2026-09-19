import { useMutation, useQueryClient } from "@tanstack/react-query";

import { beginTurn } from "../agent/turn.js";
import {
    cancelTurn,
    confirmAction,
    createConversation,
    listConversations,
    listMessages,
    listPendingActions,
    sendMessage,
    setConversationMode,
} from "../endpoints/agent.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite } from "../query.js";
import type {
    Conversation,
    ConversationFilter,
    Message,
    MessageFilter,
    PendingAction,
    PendingActionFilter,
} from "../types.js";

export function useConversations(filter: ConversationFilter = {}, limit?: number) {
    return useUnlockedInfinite<ConversationFilter, Conversation>({
        queryKey: keys.agent.conversations(filter, limit),
        fetch: listConversations,
        filters: filter,
        sort: null,
        limit,
    });
}

export function useMessages(conversationId: string | null, limit?: number) {
    const filter: MessageFilter = { conversationId: conversationId ?? "" };
    return useUnlockedInfinite<MessageFilter, Message>({
        queryKey: keys.agent.messages(filter, limit),
        fetch: listMessages,
        filters: filter,
        sort: null,
        limit,
        enabled: conversationId !== null && conversationId !== "",
    });
}

export function usePendingActions(filter: PendingActionFilter = {}, limit?: number) {
    return useUnlockedInfinite<PendingActionFilter, PendingAction>({
        queryKey: keys.agent.pending(filter, limit),
        fetch: listPendingActions,
        filters: filter,
        sort: null,
        limit,
    });
}

export function useCreateConversation() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof createConversation>[0]) => createConversation(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.agent.conversationsAll() });
        },
    });
}

export function useSetConversationMode() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setConversationMode>[0]) => setConversationMode(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.agent.conversationsAll() });
        },
    });
}

export function useSendMessage() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof sendMessage>[0]) => sendMessage(request),
        onSuccess: (answered, request) => {
            beginTurn(request.conversationId, answered.messageId);
            void client.invalidateQueries({
                queryKey: keys.agent.messagesOf(request.conversationId),
            });
        },
    });
}

export function useConfirmAction() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof confirmAction>[0]) => confirmAction(request),
        onSettled: () => {
            void client.invalidateQueries({ queryKey: keys.agent.pendingAll() });
            void client.invalidateQueries({ queryKey: keys.agent.messagesAll() });
        },
    });
}

export function useCancelTurn() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof cancelTurn>[0]) => cancelTurn(request),
        onSettled: (_answered, _thrown, request) => {
            void client.invalidateQueries({
                queryKey: keys.agent.messagesOf(request.conversationId),
            });
        },
    });
}
