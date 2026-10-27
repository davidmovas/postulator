import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { aPlan } from "../plans/fixture.js";
import type { Row } from "./model/transcript.js";
import { Transcript } from "./transcript.js";

const words = copy.agent.transcript.tool;

function toolRow(overrides: Partial<Extract<Row, { kind: "tool" }>>): Row {
    return {
        kind: "tool",
        id: "m1",
        callId: "c1",
        tool: "pages_find",
        args: null,
        status: "ok",
        durationMs: 120,
        result: null,
        error: null,
        live: false,
        ...overrides,
    };
}

function transcript(rows: readonly Row[]) {
    return render(
        <Transcript
            rows={rows}
            settling={null}
            canRetry={false}
            header={null}
            renderPlan={(plan) => <output data-testid="plan">{plan.title}</output>}
            onApprove={() => {}}
            onReject={() => {}}
            onRetry={() => {}}
        />,
    );
}

function openTheOnlyRow(): void {
    fireEvent.click(screen.getByRole("button", { name: /Find pages/ }));
}

describe("the transcript rows for a tool call", () => {
    it("says a cut result was shortened and how much was dropped", () => {
        transcript([
            toolRow({
                status: "cut",
                result: {
                    truncated: true,
                    totalBytes: 42000,
                    droppedItems: { items: 18 },
                    shortenedText: 2,
                    result: { items: [] },
                },
            }),
        ]);

        expect(screen.getByText(words.cut)).toBeDefined();

        openTheOnlyRow();
        expect(screen.getByText(words.cutSummary(42))).toBeDefined();
        expect(screen.getByText(words.cutRows(18, "items"))).toBeDefined();
        expect(screen.getByText(words.cutText(2))).toBeDefined();
        expect(screen.getByText(words.cutAsk)).toBeDefined();
    });

    it("says a denied call read and wrote nothing", () => {
        transcript([toolRow({ status: "denied", error: "the conversation may not use graph_edit" })]);

        expect(screen.getByText(words.denied)).toBeDefined();

        openTheOnlyRow();
        expect(screen.getByText("the conversation may not use graph_edit")).toBeDefined();
        expect(screen.getByText(words.deniedDetail)).toBeDefined();
    });

    it("says a failed call can be tried again", () => {
        transcript([toolRow({ status: "error", error: "the site refused the write" })]);

        expect(screen.getByText(words.error)).toBeDefined();

        openTheOnlyRow();
        expect(screen.getByText("the site refused the write")).toBeDefined();
        expect(screen.getByText(words.failedDetail)).toBeDefined();
    });

    it("keeps what a call answered folded away until it is asked for", () => {
        transcript([toolRow({ status: "denied", error: "the conversation may not use graph_edit" })]);

        expect(screen.queryByText(words.deniedDetail)).toBeNull();

        openTheOnlyRow();
        expect(screen.getByText(words.deniedDetail)).toBeDefined();
    });

    it("names a live call by what it does, and a saved one by its tool", () => {
        transcript([
            toolRow({ id: "live", tool: "pages_tree", args: { view: "subtree", pageId: "p1" }, live: true }),
            toolRow({ id: "saved", tool: "pages_tree" }),
        ]);

        expect(screen.getByText(copy.agent.acts["pages_tree"]?.acts["subtree"] ?? "")).toBeDefined();
        expect(screen.getByText(copy.agent.acts["pages_tree"]?.label ?? "")).toBeDefined();
    });

    it("tells a running call apart from one that finished", () => {
        transcript([toolRow({ status: "running", durationMs: null })]);

        expect(screen.getByText(words.running)).toBeDefined();
        expect(screen.queryByText(words.duration(120))).toBeNull();
    });

    it("hands a plan row to the plan card it is given", () => {
        transcript([toolRow({ tool: "plans" }), { kind: "plan", id: "plan:p1", plan: aPlan([]) }]);

        expect(screen.getByTestId("plan").textContent).toBe("Link the kits");
    });
});
