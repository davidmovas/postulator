import { describe, expect, it } from "vitest";

import { copy } from "../../../../copy/index.js";
import { detailLines, failure } from "./rows.js";
import type { Row } from "./transcript.js";

type ToolRow = Extract<Row, { kind: "tool" }>;

function toolRow(status: ToolRow["status"], result: unknown = null): ToolRow {
    return {
        kind: "tool",
        id: "m1",
        callId: "k1",
        tool: "pages_list",
        status,
        durationMs: 12,
        result,
        error: null,
        live: false,
    };
}

describe("failure", () => {
    it("says the turn ran out of tool calls and asks for something narrower", () => {
        const told = failure("BUDGET_EXCEEDED", "the agent used up its tool call budget for this turn");
        expect(told.spent).toBe(true);
        expect(told.title).toBe(copy.agent.states.budgetTitle);
        expect(told.body).toMatch(/narrower/);
    });

    it("hands any other failure the message it was given", () => {
        const told = failure("EXTERNAL", "the model could not answer");
        expect(told.spent).toBe(false);
        expect(told.title).toBe(copy.agent.states.errorTitle);
        expect(told.body).toBe("the model could not answer");
    });
});

describe("detailLines", () => {
    it("says a refusal read nothing and wrote nothing", () => {
        expect(detailLines(toolRow("denied"))).toStrictEqual([copy.agent.transcript.tool.deniedDetail]);
    });

    it("says a failure was handed back to the agent", () => {
        expect(detailLines(toolRow("error"))).toStrictEqual([copy.agent.transcript.tool.failedDetail]);
    });

    it("names every list a shortened answer lost and how many strings it cut", () => {
        const lines = detailLines(
            toolRow("cut", {
                truncated: true,
                totalBytes: 41_000,
                droppedItems: { items: 37, "tree.children": 2 },
                shortenedText: 1,
                result: { items: [] },
            }),
        );
        expect(lines).toStrictEqual([
            "37 rows dropped from items",
            "2 rows dropped from tree.children",
            "1 long value shortened",
            copy.agent.transcript.tool.cutAsk,
        ]);
    });

    it("says so when even shortening was not enough", () => {
        const lines = detailLines(toolRow("cut", { truncated: true, totalBytes: 41_000, preview: "{\"items\":[" }));
        expect(lines).toStrictEqual([copy.agent.transcript.tool.cutPreview, copy.agent.transcript.tool.cutAsk]);
    });

    it("says nothing extra about a clean or running call", () => {
        expect(detailLines(toolRow("ok", { items: [] }))).toStrictEqual([]);
        expect(detailLines(toolRow("running"))).toStrictEqual([]);
    });
});
