import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { PublishPane } from "./publish.js";

function aPublishResult(skipped: readonly string[]): Record<string, unknown> {
    return {
        url: "",
        status: "publish",
        contentHash: "3f1a",
        wpId: 42,
        created: false,
        seoApplied: ["title", "description"],
        skipped,
        findings: [],
    };
}

describe("the publish pane", () => {
    it("says what it skipped when it skipped something", () => {
        render(<PublishPane payload={aPublishResult(["seo_meta_skipped"])} />);

        expect(screen.getByText(copy.runs.review.publish.skipped)).toBeTruthy();
        expect(screen.getByText("seo_meta_skipped")).toBeTruthy();
    });

    it("leaves the row out when nothing was skipped", () => {
        render(<PublishPane payload={aPublishResult([])} />);

        expect(screen.queryByText(copy.runs.review.publish.skipped)).toBeNull();
        expect(screen.getByText(copy.runs.review.publish.seoApplied)).toBeTruthy();
    });
});
