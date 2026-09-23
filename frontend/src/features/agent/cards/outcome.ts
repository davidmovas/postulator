import { copy } from "../../../copy/index.js";
import type { PendingAction } from "../../../data/types.js";
import { resultSummary } from "../conversation/model/tools.js";

export function outcomeOf(action: PendingAction): string | null {
    switch (action.status) {
        case "pending":
            return null;
        case "approved":
            return copy.agent.card.approved;
        case "executed": {
            if ((action.error ?? "") !== "") {
                return copy.agent.card.undelivered;
            }
            const summary = resultSummary(action.tool, action.result ?? null);
            return summary === "done" ? copy.agent.card.executed : copy.agent.card.executedWith(summary);
        }
        case "rejected":
            return copy.agent.card.rejected;
        default:
            return copy.agent.card.failed(action.error ?? "");
    }
}
