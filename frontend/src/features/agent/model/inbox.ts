import type { Conversation, PendingAction } from "../../../data/types.js";

export interface ActionGroup {
    conversationId: string;
    title: string | null;
    siteId: string | null;
    newestAt: string;
    actions: PendingAction[];
}

export type InboxStatus = "pending" | "executed" | "rejected" | "failed";

export const inboxStatuses: readonly InboxStatus[] = ["pending", "executed", "rejected", "failed"];

function stamp(value: string | null): string {
    return value ?? "";
}

export function groupActions(actions: readonly PendingAction[], conversations: readonly Conversation[]): ActionGroup[] {
    const known = new Map(conversations.map((held) => [held.id, held]));
    const groups = new Map<string, ActionGroup>();
    for (const action of actions) {
        let group = groups.get(action.conversationId);
        if (group === undefined) {
            const conversation = known.get(action.conversationId);
            group = {
                conversationId: action.conversationId,
                title: conversation === undefined ? null : conversation.title,
                siteId: conversation?.siteId ?? null,
                newestAt: "",
                actions: [],
            };
            groups.set(action.conversationId, group);
        }
        group.actions.push(action);
        if (stamp(action.createdAt) > group.newestAt) {
            group.newestAt = stamp(action.createdAt);
        }
    }
    const out = [...groups.values()];
    for (const group of out) {
        group.actions.sort((a, b) => stamp(a.createdAt).localeCompare(stamp(b.createdAt)));
    }
    out.sort((a, b) => b.newestAt.localeCompare(a.newestAt));
    return out;
}

export function statusCounts(actions: readonly PendingAction[]): Record<InboxStatus, number> {
    const counts: Record<InboxStatus, number> = { pending: 0, executed: 0, rejected: 0, failed: 0 };
    for (const action of actions) {
        if (action.status === "pending" || action.status === "executed" || action.status === "rejected" || action.status === "failed") {
            counts[action.status] += 1;
        }
    }
    return counts;
}
