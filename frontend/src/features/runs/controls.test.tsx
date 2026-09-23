import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Run } from "../../data/types.js";

const asked: { runId: string }[] = [];

const idle = { mutate: () => {}, isPending: false };

vi.mock("../../data/hooks/runs.js", () => ({
    usePauseRun: () => idle,
    useResumeRun: () => idle,
    useCancelRun: () => idle,
    useRevertRun: () => ({
        mutate: (request: { runId: string }) => {
            asked.push(request);
        },
        isPending: false,
    }),
}));

const { RunControls } = await import("./controls.js");

function aRun(overrides: Partial<Run> = {}): Run {
    return {
        id: "r1",
        siteId: "s1",
        kind: "generate",
        status: "completed",
        targets: ["p1"],
        recipe: [{ name: "generate_body", enabled: true, params: null }],
        templateId: "t1",
        templateVersion: 3,
        publishMode: "draft",
        budget: { maxUsd: 5, maxTokens: 0 },
        stats: { items: 4, done: 4, failed: 0, tokens: 1200, usd: 0.4 },
        createdBy: "user",
        parentRunId: null,
        pauseReason: "",
        error: "",
        deadlineAt: "2026-09-19T18:00:00Z",
        createdAt: "2026-09-19T10:00:00Z",
        startedAt: "2026-09-19T10:00:01Z",
        finishedAt: "2026-09-19T10:30:00Z",
        ...overrides,
    };
}

function revertOf(run: Run): { disabled: boolean; title: string | null } {
    const view = render(<RunControls run={run} />);
    const button = screen.getByRole("button", { name: copy.runs.revert });
    const read = { disabled: button.hasAttribute("disabled"), title: button.getAttribute("title") };
    view.unmount();
    return read;
}

describe("the revert control", () => {
    it("is offered on a run that has finished, however it finished", () => {
        for (const status of ["completed", "failed", "cancelled"]) {
            expect({ status, ...revertOf(aRun({ status })) }).toStrictEqual({ status, disabled: false, title: null });
        }
    });

    it("refuses a run that is still going and says to cancel it first", () => {
        for (const status of ["running", "paused", "pending", "waiting"]) {
            expect({ status, ...revertOf(aRun({ status })) }).toStrictEqual({
                status,
                disabled: true,
                title: copy.runs.revertRunning,
            });
        }
    });

    it("refuses to revert a revert", () => {
        expect(revertOf(aRun({ kind: "revert", status: "completed" }))).toStrictEqual({
            disabled: true,
            title: copy.runs.revertOfARevert,
        });
    });

    it("asks before it puts anything back", () => {
        asked.length = 0;
        render(<RunControls run={aRun()} />);

        fireEvent.click(screen.getByRole("button", { name: copy.runs.revert }));
        expect(screen.getByText(copy.runs.revertTitle)).toBeDefined();
        expect(asked).toStrictEqual([]);

        fireEvent.click(screen.getByRole("button", { name: copy.runs.revertConfirm }));
        expect(asked).toStrictEqual([{ runId: "r1" }]);
    });
});
