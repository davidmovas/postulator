import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import type { Row } from "./model/transcript.js";
import { Transcript } from "./transcript.js";

const words = copy.agent.transcript.tool;

function toolRow(overrides: Partial<Extract<Row, { kind: "tool" }>>): Row {
    return {
        kind: "tool",
        id: "m1",
        callId: "c1",
        tool: "pages_list",
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
            onApprove={() => {}}
            onReject={() => {}}
            onRetry={() => {}}
        />,
    );
}

function openTheOnlyRow(): void {
    fireEvent.click(screen.getByRole("button", { name: /pages_list|Pages/i }));
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
        transcript([toolRow({ status: "denied", error: "the conversation may not use graph_delete_entity" })]);

        expect(screen.getByText(words.denied)).toBeDefined();

        openTheOnlyRow();
        expect(screen.getByText("the conversation may not use graph_delete_entity")).toBeDefined();
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
        transcript([toolRow({ status: "denied", error: "the conversation may not use graph_delete_entity" })]);

        expect(screen.queryByText(words.deniedDetail)).toBeNull();

        openTheOnlyRow();
        expect(screen.getByText(words.deniedDetail)).toBeDefined();
    });

    it("tells a running call apart from one that finished", () => {
        transcript([toolRow({ status: "running", durationMs: null })]);

        expect(screen.getByText(words.running)).toBeDefined();
        expect(screen.queryByText(words.duration(120))).toBeNull();
    });
});
