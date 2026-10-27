import type { Conversation } from "../../../data/types.js";

function stamp(value: string | null): string {
    return value ?? "";
}

export function matches(conversation: Conversation, query: string): boolean {
    const needle = query.trim().toLowerCase();
    if (needle === "") {
        return true;
    }
    return conversation.title.toLowerCase().includes(needle);
}

export function inScope(
    conversations: readonly Conversation[],
    siteId: string | null,
    query: string,
): Conversation[] {
    return conversations
        .filter((held) => (held.siteId ?? null) === siteId && matches(held, query))
        .sort((a, b) => stamp(b.createdAt).localeCompare(stamp(a.createdAt)));
}
