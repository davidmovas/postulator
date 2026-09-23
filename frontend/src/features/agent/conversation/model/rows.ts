import { budgetCode } from "../../../../data/agent/turn.js";
import { copy } from "../../../../copy/index.js";
import { truncationOf } from "./tools.js";
import type { Row } from "./transcript.js";

export interface Failure {
    spent: boolean;
    title: string;
    body: string;
}

export function failure(code: string, message: string): Failure {
    if (code === budgetCode) {
        return { spent: true, title: copy.agent.states.budgetTitle, body: copy.agent.states.budgetBody };
    }
    return { spent: false, title: copy.agent.states.errorTitle, body: message };
}

export function detailLines(row: Extract<Row, { kind: "tool" }>): string[] {
    const words = copy.agent.transcript.tool;
    switch (row.status) {
        case "denied":
            return [words.deniedDetail];
        case "error":
            return [words.failedDetail];
        case "cut":
            return cutLines(row.result);
        default:
            return [];
    }
}

function cutLines(result: unknown): string[] {
    const words = copy.agent.transcript.tool;
    const cut = truncationOf(result);
    if (cut === null) {
        return [];
    }

    const lines = cut.dropped.map((held) => words.cutRows(held.count, held.path));
    if (cut.shortened > 0) {
        lines.push(words.cutText(cut.shortened));
    }
    if (!cut.whole) {
        lines.push(words.cutPreview);
    }
    lines.push(words.cutAsk);
    return lines;
}
