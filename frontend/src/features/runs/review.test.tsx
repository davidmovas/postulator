import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { RunItem } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";

const accepted: unknown[] = [];

const idle = { mutate: () => undefined, isPending: false, error: null };

const plan = {
    hash: "abc",
    pages: [
        {
            pageId: "p1",
            path: "/espresso/",
            body: "body",
            wpId: 12,
            fields: [
                { key: "excerpt", before: "Old", beforeHash: "b1", after: "New", afterHash: "a1" },
                { key: "seo_title", before: "", beforeHash: "", after: "Espresso", afterHash: "a2" },
            ],
            blocks: [{ kind: "heading", hash: "h2", raw: "<h2>Grind</h2>", owner: "ours", change: "added" }],
        },
    ],
};

vi.mock("../../data/hooks/runs.js", () => ({
    useArtifacts: () => ({ data: { artifacts: [{ kind: "publish_result" }, { kind: "edit_plan" }] }, isPending: false }),
    useArtifact: (_itemId: string | null, kind: string) => ({
        data:
            kind === "edit_plan"
                ? {
                      artifact: {
                          kind,
                          step: "publish",
                          content: JSON.stringify(plan),
                          purged: false,
                          size: 120,
                          createdAt: "2026-10-22T09:00:00Z",
                      },
                  }
                : undefined,
        isPending: false,
        isError: false,
        error: null,
    }),
    useRetryStep: () => ({
        mutate: (request: unknown) => {
            accepted.push(request);
        },
        isPending: false,
    }),
    useRegenerate: () => idle,
    useResumeRun: () => idle,
    useRevertRun: () => idle,
    useRun: () => ({ data: undefined }),
}));

vi.mock("../../data/hooks/settings.js", () => ({
    useSetting: () => ({ data: undefined }),
}));

vi.mock("../../data/hooks/pages.js", () => ({
    usePageModel: () => ({ data: undefined }),
}));

const { ReviewDrawer } = await import("./review.js");

function aHeldItem(): RunItem {
    return {
        id: "i1",
        runId: "r1",
        siteId: "s1",
        targetId: "p1",
        status: "paused",
        currentStep: "publish",
        seq: 0,
        blockedBy: "",
        attempts: 1,
        pauseReason: "awaiting_review",
        error: "",
        note: "",
        waitingFor: null,
        retryable: true,
        retryBlockedReason: "",
        wakeAt: null,
        createdAt: "2026-10-22T09:00:00Z",
        updatedAt: "2026-10-22T09:00:00Z",
        finishedAt: null,
    };
}

function drawer(): void {
    const item = aHeldItem();
    renderScreen(
        <ReviewDrawer
            siteId="s1"
            view={{ item, step: "publish", stepStartedAt: null, stepAttempt: 1, stepFailure: null }}
            page={undefined}
            steps={["publish"]}
            timeline={[]}
            retry={undefined}
            path="/espresso/"
            now={0}
            missing={false}
            narrowed={false}
            siblings={[item]}
            onClearFilter={() => undefined}
            onMove={() => undefined}
            onOpenPage={() => undefined}
            onClose={() => undefined}
            onRerun={() => undefined}
        />,
    );
}

describe("accepting a draft run's review in the item drawer", () => {
    it("opens on the changes to review and accepts them whole while every tick stays", () => {
        accepted.length = 0;
        drawer();

        expect(screen.getByRole("checkbox", { name: copy.pageTypes.fields.excerpt })).toHaveProperty("checked", true);
        fireEvent.click(screen.getByRole("button", { name: copy.runs.acceptReview }));

        expect(accepted).toStrictEqual([{ itemId: "i1", acceptFindings: true }]);
    });

    it("sends the ticked keys once a field is unticked", () => {
        accepted.length = 0;
        drawer();

        fireEvent.click(screen.getByRole("checkbox", { name: copy.pageTypes.fields.seo_title }));
        fireEvent.click(screen.getByRole("button", { name: copy.runs.acceptReview }));

        expect(accepted).toStrictEqual([{ itemId: "i1", acceptFindings: true, fields: ["excerpt", "body"] }]);
    });

    it("refuses to accept once nothing is ticked", () => {
        accepted.length = 0;
        drawer();

        for (const name of [copy.pageTypes.fields.excerpt, copy.pageTypes.fields.seo_title, copy.pageTypes.fields.body]) {
            fireEvent.click(screen.getByRole("checkbox", { name }));
        }
        const accept = screen.getByRole("button", { name: copy.runs.acceptReview });
        expect(accept).toHaveProperty("disabled", true);
        expect(accept.getAttribute("title")).toBe(copy.runs.review.plan.nothingTicked);
    });
});
