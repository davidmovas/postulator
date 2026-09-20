import type { Conversation } from "../../../data/types.js";
import { globalDockKey } from "../dock/state.js";

export interface SiteGroup {
    key: string;
    name: string | null;
    conversations: Conversation[];
}

function stamp(value: string | null): string {
    return value ?? "";
}

export function groupBySite(
    conversations: readonly Conversation[],
    siteNames: ReadonlyMap<string, string>,
    currentSiteId: string | null,
): SiteGroup[] {
    const groups = new Map<string, SiteGroup>();
    for (const held of conversations) {
        const key = held.siteId ?? globalDockKey;
        let group = groups.get(key);
        if (group === undefined) {
            group = { key, name: held.siteId === null ? null : (siteNames.get(held.siteId) ?? held.siteId), conversations: [] };
            groups.set(key, group);
        }
        group.conversations.push(held);
    }
    const out = [...groups.values()];
    for (const group of out) {
        group.conversations.sort((a, b) => stamp(b.createdAt).localeCompare(stamp(a.createdAt)));
    }
    const newest = (group: SiteGroup): string => stamp(group.conversations[0]?.createdAt ?? null);
    out.sort((a, b) => {
        if (a.key === currentSiteId) {
            return -1;
        }
        if (b.key === currentSiteId) {
            return 1;
        }
        if (a.key === globalDockKey) {
            return 1;
        }
        if (b.key === globalDockKey) {
            return -1;
        }
        return newest(b).localeCompare(newest(a));
    });
    return out;
}

export function matches(conversation: Conversation, query: string): boolean {
    const needle = query.trim().toLowerCase();
    if (needle === "") {
        return true;
    }
    return conversation.title.toLowerCase().includes(needle);
}
