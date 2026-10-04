import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { PublishPane, RevertPane, SyncPane } from "./publish.js";

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

describe("the sync pane", () => {
    it("lists what a read-back found the site holding against the plan", () => {
        render(
            <SyncPane
                payload={{
                    url: "",
                    status: "publish",
                    links: 3,
                    findings: [
                        {
                            severity: "warn",
                            code: "plan_not_kept",
                            message: "the slug of /a/ was planned as a and the site holds a-2",
                        },
                    ],
                }}
            />,
        );
        expect(screen.getByText("plan_not_kept")).toBeDefined();
        expect(screen.getByText("the slug of /a/ was planned as a and the site holds a-2")).toBeDefined();
    });

    it("shows no finding list for a page read back cleanly", () => {
        render(<SyncPane payload={{ url: "", status: "publish", links: 3 }} />);
        expect(screen.queryByText(copy.runs.review.links.clean)).toBeNull();
    });
});

describe("the revert pane", () => {
    it("says what the revert did, and why the SEO meta it wrote stays", () => {
        render(
            <RevertPane
                payload={{
                    pageId: "p1",
                    path: "/accessories/knock-boxes/",
                    outcome: "restored",
                    detail: "the content the run replaced was written back",
                    neighbors: [],
                    findings: [
                        {
                            severity: "warn",
                            code: "revert_meta_kept",
                            message: "the SEO meta the run wrote to /accessories/knock-boxes/ stays as it is: no plugin",
                        },
                    ],
                }}
            />,
        );

        expect(screen.getByText(copy.runs.review.revert.outcomes["restored"] ?? "")).toBeDefined();
        expect(screen.getByText(copy.runs.review.revert.none)).toBeDefined();
        expect(screen.getByText("revert_meta_kept")).toBeDefined();
        expect(screen.getByText(/stays as it is: no plugin$/)).toBeDefined();
    });
});
