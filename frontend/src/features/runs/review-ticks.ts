import type { EditPlanView } from "./artifacts.js";
import { sentFields } from "./diff-labels.js";

export interface ReviewTicks {
    offered: readonly string[];
    unticked: ReadonlySet<string>;
}

export interface AcceptRequest {
    itemId: string;
    acceptFindings: true;
    fields?: string[];
}

export function offeredKeys(plan: EditPlanView | null): string[] {
    if (plan === null) {
        return [];
    }
    const fields = plan.pages.flatMap((page) => page.fields.map((change) => change.key));
    if (fields.length === 0) {
        return [];
    }
    const bodies = plan.pages.map((page) => page.body).filter((body) => body !== "");
    return [...new Set([...fields, ...bodies])];
}

export function acceptRequest(itemId: string, ticks: ReviewTicks | null): AcceptRequest | null {
    if (ticks === null) {
        return { itemId, acceptFindings: true };
    }
    const sent = sentFields(ticks.offered, ticks.unticked);
    switch (sent.kind) {
        case "all":
            return { itemId, acceptFindings: true };
        case "some":
            return { itemId, acceptFindings: true, fields: [...sent.fields] };
        default:
            return null;
    }
}

export function toggled(held: ReadonlySet<string>, key: string): ReadonlySet<string> {
    const next = new Set(held);
    if (!next.delete(key)) {
        next.add(key);
    }
    return next;
}
