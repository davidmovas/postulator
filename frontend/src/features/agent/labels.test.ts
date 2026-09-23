import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import type { ToolRowStatus } from "./conversation/model/transcript.js";
import { toolStatusLabel, toolStatusTone } from "./labels.js";

describe("toolStatusLabel and toolStatusTone", () => {
    it("reads a refusal, a shortening and a failure as three different things", () => {
        const cases: { status: ToolRowStatus; label: string; tone: string }[] = [
            { status: "running", label: copy.agent.transcript.tool.running, tone: "info" },
            { status: "ok", label: copy.agent.transcript.tool.ok, tone: "ok" },
            { status: "cut", label: copy.agent.transcript.tool.cut, tone: "warn" },
            { status: "denied", label: copy.agent.transcript.tool.denied, tone: "warn" },
            { status: "error", label: copy.agent.transcript.tool.error, tone: "danger" },
        ];
        for (const held of cases) {
            expect(toolStatusLabel(held.status)).toBe(held.label);
            expect(toolStatusTone(held.status)).toBe(held.tone);
        }
    });

    it("gives every status a word of its own", () => {
        const statuses: ToolRowStatus[] = ["running", "ok", "cut", "denied", "error"];
        const words = new Set(statuses.map(toolStatusLabel));
        expect(words.size).toBe(statuses.length);
    });
});
