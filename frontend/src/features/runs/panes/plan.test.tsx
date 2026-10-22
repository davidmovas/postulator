import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";

vi.mock("../../../data/hooks/pages.js", () => ({
    usePageModel: () => ({ data: undefined }),
}));

const { EditPlanPane } = await import("./plan.js");

const said = copy.runs.review;

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
            blocks: [
                { kind: "paragraph", hash: "h1", raw: "<p>The client's own words.</p>", owner: "client", change: "kept" },
                { kind: "heading", hash: "h2", raw: "<h2>Grind</h2>", owner: "ours", change: "added" },
            ],
        },
    ],
};

const relink = {
    hash: "def",
    pages: [
        {
            pageId: "p2",
            path: "/grinders/",
            body: "body",
            wpId: 13,
            fields: [],
            blocks: [{ kind: "paragraph", hash: "h3", raw: "<p>See our espresso guide.</p>", owner: "ours", change: "added" }],
        },
    ],
};

describe("the review of a draft run", () => {
    it("shows each page's fields and blocks, and says a draft run makes no image for a page on the site", () => {
        render(<EditPlanPane payload={plan} choice={null} />);

        const page = within(screen.getByRole("region", { name: "/espresso/" }));
        expect(page.getByText("Old")).toBeDefined();
        expect(page.getByText("Espresso")).toBeDefined();
        expect(page.getByText("Kept: the client's")).toBeDefined();
        expect(page.getByText("Added: ours")).toBeDefined();
        expect(screen.getByText(said.plan.noImages)).toBeDefined();
        expect(screen.queryByRole("checkbox")).toBeNull();
    });

    it("ticks every field and the body by default, and hands an untick back", () => {
        const toggled: string[] = [];
        render(
            <EditPlanPane
                payload={plan}
                choice={{
                    offered: ["excerpt", "seo_title", "body"],
                    unticked: new Set(["seo_title"]),
                    onToggle: (key) => {
                        toggled.push(key);
                    },
                }}
            />,
        );

        const excerpt = screen.getByRole("checkbox", { name: copy.pageTypes.fields.excerpt });
        const title = screen.getByRole("checkbox", { name: copy.pageTypes.fields.seo_title });
        const body = screen.getByRole("checkbox", { name: copy.pageTypes.fields.body });
        expect(excerpt).toHaveProperty("checked", true);
        expect(title).toHaveProperty("checked", false);
        expect(body).toHaveProperty("checked", true);

        fireEvent.click(body);
        fireEvent.click(title);
        expect(toggled).toStrictEqual(["body", "seo_title"]);
    });

    it("offers no tick on a review that changes bodies only", () => {
        render(<EditPlanPane payload={relink} choice={null} />);

        expect(screen.getByText("Added: ours")).toBeDefined();
        expect(screen.queryByRole("checkbox")).toBeNull();
    });

    it("says so when the review holds no page", () => {
        render(<EditPlanPane payload={{ hash: "x", pages: [] }} choice={null} />);

        expect(screen.getByText(said.plan.none)).toBeDefined();
    });
});
