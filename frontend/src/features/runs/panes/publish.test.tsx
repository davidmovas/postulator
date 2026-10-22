import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { PageModel } from "../../../data/types.js";
import { hubModel } from "../../pages/model/fixture.js";

const held: { model: PageModel | undefined } = { model: undefined };

vi.mock("../../../data/hooks/pages.js", () => ({
    usePageModel: () => ({ data: held.model }),
}));

const { PublishPane, RevertPane, SyncPane } = await import("./publish.js");

const said = copy.runs.review.diff;

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

function aDiffedResult(): Record<string, unknown> {
    return {
        ...aPublishResult([]),
        fields: [
            { key: "excerpt", before: "Old excerpt", beforeHash: "b1", after: "New excerpt", afterHash: "a1" },
            {
                key: "product_short_description",
                before: "<p>Short &amp; old</p>",
                beforeHash: "b2",
                after: "",
                afterHash: "a2",
            },
        ],
        blocks: [
            { kind: "paragraph", hash: "h1", raw: "<p>The client's own words.</p>", owner: "client", change: "kept" },
            { kind: "shortcode", hash: "h2", raw: "[entity_products]", owner: "client", change: "kept" },
            { kind: "heading", hash: "h3", raw: "<h2>Dosage</h2>", owner: "ours", change: "added" },
            { kind: "html", hash: "h4", raw: "<div>a filter's wrapper</div>", owner: "site", change: "added" },
            { kind: "paragraph", hash: "h5", raw: "<p>Our older text.</p>", owner: "ours", change: "removed" },
        ],
        declined: ["seo_title"],
    };
}

function section(name: string): HTMLElement {
    return screen.getByRole("region", { name });
}

describe("the publish pane", () => {
    it("says what it skipped when it skipped something", () => {
        render(<PublishPane payload={aPublishResult(["seo_meta_skipped"])} pageId="" />);

        expect(screen.getByText(copy.runs.review.publish.skipped)).toBeTruthy();
        expect(screen.getByText("seo_meta_skipped")).toBeTruthy();
    });

    it("leaves the row out when nothing was skipped", () => {
        render(<PublishPane payload={aPublishResult([])} pageId="" />);

        expect(screen.queryByText(copy.runs.review.publish.skipped)).toBeNull();
        expect(screen.getByText(copy.runs.review.publish.seoApplied)).toBeTruthy();
    });

    it("sets each written field's value before against after, its hashes behind a tooltip", () => {
        render(<PublishPane payload={aDiffedResult()} pageId="" />);

        const fields = within(section(said.fields));
        expect(fields.getByText(copy.pageTypes.fields.excerpt)).toBeDefined();
        expect(fields.getByText("Old excerpt")).toBeDefined();
        expect(fields.getByText("New excerpt")).toBeDefined();
        expect(fields.getByTitle(said.hashes("b1", "a1"))).toBeDefined();
        expect(fields.getByText("Short & old")).toBeDefined();
        expect(fields.getByText(said.empty)).toBeDefined();
    });

    it("lays the body out block by block with whose each is and what became of it", () => {
        render(<PublishPane payload={aDiffedResult()} pageId="" />);

        const body = within(section(said.body));
        expect(body.getAllByText("Kept: the client's")).toHaveLength(2);
        expect(body.getByText("Added: ours")).toBeDefined();
        expect(body.getByText("Added: the site's")).toBeDefined();
        expect(body.getByText("Removed: ours")).toBeDefined();
        expect(body.getByText("The client's own words.")).toBeDefined();
        expect(body.getAllByText(said.rendered)).toHaveLength(1);
        expect(body.getByText(copy.pages.model.blockKinds.shortcode)).toBeDefined();
    });

    it("names the fields a person left unticked", () => {
        render(<PublishPane payload={aDiffedResult()} pageId="" />);

        expect(screen.getByText(said.declined)).toBeDefined();
        expect(screen.getByText(copy.pageTypes.fields.seo_title)).toBeDefined();
    });

    it("says what the site adds around the body, inside or outside the content, and leaves the content out", () => {
        held.model = {
            ...hubModel,
            render: {
                ...(hubModel.render ?? ({} as NonNullable<PageModel["render"]>)),
                regions: [
                    { kind: "content", inside: true, links: 1, words: 200, signal: "" },
                    { kind: "generated", inside: false, links: 0, words: 30, signal: "" },
                    { kind: "content_dynamic", inside: true, links: 0, words: 12, signal: "" },
                    { kind: "listing", inside: false, links: 6, words: 80, signal: "jsonld" },
                ],
            },
        };
        render(<PublishPane payload={aDiffedResult()} pageId="p-tb500" />);

        const regions = within(section(said.regions));
        expect(regions.getByText(copy.pages.model.regionKinds.content_dynamic)).toBeDefined();
        expect(regions.getByText(copy.pages.model.render.inside)).toBeDefined();
        expect(regions.getByText(copy.pages.model.regionKinds.listing)).toBeDefined();
        expect(regions.getByText(copy.pages.model.render.outside)).toBeDefined();
        expect(regions.queryByText(copy.pages.model.regionKinds.content)).toBeNull();
        expect(regions.queryByText(copy.pages.model.regionKinds.generated)).toBeNull();
        held.model = undefined;
    });

    it("shows no diff for a result written before the diff was recorded", () => {
        render(<PublishPane payload={aPublishResult([])} pageId="" />);

        expect(screen.queryByRole("region", { name: said.fields })).toBeNull();
        expect(screen.queryByRole("region", { name: said.body })).toBeNull();
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

    it("lists each field it put back or handed back, with the reason", () => {
        const moved = "it changed on the site since the run wrote it, so the edit is kept";
        render(
            <RevertPane
                payload={{
                    pageId: "p1",
                    path: "/a/",
                    outcome: "restored",
                    detail: "each field the run wrote that the site still holds as it wrote it was put back",
                    neighbors: [],
                    fields: [
                        { key: "excerpt", outcome: "restored" },
                        { key: "seo_title", outcome: "kept", detail: moved },
                    ],
                    findings: [],
                }}
            />,
        );

        const fields = within(screen.getByRole("region", { name: copy.runs.review.revert.fields }));
        expect(fields.getByText(copy.pageTypes.fields.excerpt)).toBeDefined();
        expect(fields.getByText(copy.runs.review.revert.fieldOutcomes["restored"] ?? "")).toBeDefined();
        expect(fields.getByText(copy.pageTypes.fields.seo_title)).toBeDefined();
        expect(fields.getByText(copy.runs.review.revert.fieldOutcomes["kept"] ?? "")).toBeDefined();
        expect(fields.getByText(moved)).toBeDefined();
    });
});
