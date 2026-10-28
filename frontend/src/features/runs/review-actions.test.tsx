import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { renderScreen } from "../../testing/render.js";

vi.mock("../agent/index.js", () => ({ askAgent: () => undefined }));

const { ReviewActions } = await import("./review-actions.js");

function shown(sync: boolean, calls: string[]): void {
    renderScreen(
        <ReviewActions
            path="The whole site"
            runId="r-sync"
            itemId="i-sync"
            pageId="s1"
            sync={sync}
            blocked={null}
            state={{ kind: "ready" }}
            primary="retry"
            busy={false}
            regeneration={{ kind: "ready" }}
            regenerating={false}
            review={false}
            nothingTicked={false}
            onOpenPage={() => {
                calls.push("open");
            }}
            onRerun={() => {
                calls.push("rerun");
            }}
            onRetry={() => {
                calls.push("retry");
            }}
            onAccept={() => {
                calls.push("accept");
            }}
            onRegenerate={() => {
                calls.push("regenerate");
            }}
        />,
    );
}

describe("the actions of a sync's item", () => {
    it("offers no page to open, since a sync reads the whole site", () => {
        shown(true, []);

        expect(screen.queryByRole("button", { name: copy.runs.review.openPage })).toBeNull();
    });

    it("resumes the sync from its last batch and starts it over, in the sync's own words", async () => {
        const user = userEvent.setup();
        const calls: string[] = [];
        shown(true, calls);

        await user.click(screen.getByRole("button", { name: copy.sync.resume }));
        await user.click(screen.getByRole("button", { name: copy.sync.startOver }));

        expect(calls).toEqual(["retry", "regenerate"]);
        expect(screen.queryByRole("button", { name: copy.runs.regenerate })).toBeNull();
    });
});

describe("the actions of a page's item", () => {
    it("still opens the page, retries the step and regenerates the page", () => {
        shown(false, []);

        expect(screen.getByRole("button", { name: copy.runs.review.openPage })).toBeDefined();
        expect(screen.getByRole("button", { name: copy.runs.retryStep })).toBeDefined();
        expect(screen.getByRole("button", { name: copy.runs.regenerate })).toBeDefined();
    });
});
