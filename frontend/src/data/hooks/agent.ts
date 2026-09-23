import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo } from "react";

import { maxLimit } from "../../lib/paging.js";
import { reconcileConversation } from "../agent/reconcile.js";
import { attachAssistant, beginStop, failTurn, startTurn } from "../agent/turn.js";
import { flatten } from "../call.js";
import {
    cancelTurn,
    confirmAction,
    createConversation,
    deleteConversation,
    listConversations,
    listMessages,
    listPendingActions,
    renameConversation,
    sendMessage,
    setConversationMode,
} from "../endpoints/agent.js";
import { failure, messages as errorMessages } from "../errors.js";
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

export const maxAutoPages = 4;

interface TurnContext {
    turnSeq: number;
}

function describedFailure(thrown: unknown): { code: string; message: string } {
    const reported = failure(thrown);
    return {
        code: reported.code,
        message: reported.message === "" ? errorMessages[reported.code] : reported.message,
    };
}

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

export interface Transcript {
    messages: readonly Message[];
    isPending: boolean;
    complete: boolean;
    error: Error | null;
    canLoadEarlier: boolean;
    loadingEarlier: boolean;
    loadEarlier: () => void;
}

export function useTranscript(conversationId: string | null): Transcript {
    const listed = useMessages(conversationId, maxLimit);
    const { hasNextPage, isError, isFetchingNextPage, isFetching, fetchNextPage } = listed;
    const loaded = listed.data?.pages.length ?? 0;

    useEffect(() => {
        if (hasNextPage && !isError && !isFetchingNextPage && !isFetching && loaded < maxAutoPages) {
            void fetchNextPage();
        }
    }, [hasNextPage, isError, isFetchingNextPage, isFetching, fetchNextPage, loaded]);

    const rows = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const loadEarlier = useCallback(() => {
        void fetchNextPage();
    }, [fetchNextPage]);

    return {
        messages: rows,
        isPending: listed.isPending,
        complete: listed.data !== undefined && !hasNextPage,
        error: listed.error,
        canLoadEarlier: hasNextPage && !isFetchingNextPage,
        loadingEarlier: isFetchingNextPage,
        loadEarlier,
    };
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

export function useRenameConversation() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof renameConversation>[0]) => renameConversation(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.agent.conversationsAll() });
        },
    });
}

export function useDeleteConversation() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteConversation>[0]) => deleteConversation(request),
        onSuccess: (_answered, request) => {
            void client.invalidateQueries({ queryKey: keys.agent.conversationsAll() });
            void client.invalidateQueries({ queryKey: keys.agent.pendingAll() });
            client.removeQueries({ queryKey: keys.agent.messagesOf(request.conversationId) });
        },
    });
}

function afterTurnStarted(client: QueryClient, conversationId: string): void {
    void client.invalidateQueries({ queryKey: keys.agent.messagesOf(conversationId) });
    void client.invalidateQueries({ queryKey: keys.agent.conversationsAll() });
}

export function useSendMessage() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof sendMessage>[0]) => sendMessage(request),
        onMutate: (request): TurnContext => ({ turnSeq: startTurn(request.conversationId) }),
        onSuccess: (answered, request, held) => {
            attachAssistant(request.conversationId, answered.assistantMessageId, held.turnSeq);
            afterTurnStarted(client, request.conversationId);
            void reconcileConversation(client, request.conversationId);
        },
        onError: (thrown, request, held) => {
            const described = describedFailure(thrown);
            failTurn(request.conversationId, described.code, described.message, held?.turnSeq ?? 0);
        },
    });
}

export interface StartRequest {
    siteId: string | null;
    mode: string;
    text: string;
}

export function useStartConversation() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: async (request: StartRequest): Promise<Conversation> => {
            const opened = await createConversation({
                siteId: request.siteId ?? undefined,
                mode: request.mode,
            });
            const conversation = opened.conversation;
            const turnSeq = startTurn(conversation.id);
            try {
                const answered = await sendMessage({ conversationId: conversation.id, text: request.text });
                attachAssistant(conversation.id, answered.assistantMessageId, turnSeq);
            } catch (thrown) {
                const described = describedFailure(thrown);
                failTurn(conversation.id, described.code, described.message, turnSeq);
            }
            return conversation;
        },
        onSuccess: (conversation) => {
            afterTurnStarted(client, conversation.id);
            void reconcileConversation(client, conversation.id);
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
        onMutate: (request) => {
            beginStop(request.conversationId);
        },
        onSuccess: (answered, request) => {
            if (!answered.cancelled) {
                void reconcileConversation(client, request.conversationId);
            }
        },
        onError: (_thrown, request) => {
            void reconcileConversation(client, request.conversationId);
        },
        onSettled: (_answered, _thrown, request) => {
            void client.invalidateQueries({ queryKey: keys.agent.messagesOf(request.conversationId) });
        },
    });
}

export function useTurnReconciler(conversationId: string | null): void {
    const client = useQueryClient();

    useEffect(() => {
        if (conversationId === null || conversationId === "") {
            return;
        }
        void reconcileConversation(client, conversationId);
        const wake = (): void => {
            void reconcileConversation(client, conversationId);
        };
        window.addEventListener("focus", wake);
        return () => {
            window.removeEventListener("focus", wake);
        };
    }, [client, conversationId]);
}
