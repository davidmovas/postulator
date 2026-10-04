import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { PublishPane, RevertPane, SyncPane } from "./publish.js";

function aPublishResult(skipped: readonly string[], extra: Record<string, unknown> = {}): Record<string, unknown> {
    return {
        url: "",
        status: "publish",
        contentHash: "3f1a",
        wpId: 42,
        created: false,
        seoApplied: ["title", "description"],
        skipped,
        findings: [],
        ...extra,
    };
}

const said = copy.runs.review.publish;

function filedUnder(): HTMLElement {
    const section = document.querySelector<HTMLElement>("[data-publish-categories]");
    if (section === null) {
        throw new Error("the publish pane shows no categories");
    }
    return section;
}

function rowValue(container: HTMLElement, label: string): string {
    const term = within(container).getByText(label);
    return term.nextElementSibling?.textContent ?? "";
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

    it("shows what the page is filed under, which terms the run created and what it added", () => {
        render(
            <PublishPane
                payload={aPublishResult([], {
                    categories: {
                        taxonomy: "category",
                        terms: [
                            { categoryId: "c1", name: "Peptides", termId: 12, parentId: 0, created: false },
                            { categoryId: "c2", name: "Healing", termId: 31, parentId: 12, created: true },
                        ],
                        previous: [5, 12],
                        added: [31],
                        taken: true,
                    },
                })}
            />,
        );

        const section = filedUnder();
        expect(within(section).getByText(said.filedUnder)).toBeDefined();
        const trail = within(section).getByRole("list", { name: copy.categories.trail });
        const chips = [...trail.querySelectorAll("li")];
        expect(chips.map((chip) => chip.getAttribute("title"))).toStrictEqual([
            copy.categories.onSite(12),
            copy.categories.createdByRun(31),
        ]);
        expect(rowValue(section, said.termsCreated)).toBe("Healing");
        expect(rowValue(section, said.termsAdded)).toBe("Healing");
        expect(rowValue(section, said.termsBefore)).toBe("#5, Peptides");
        expect(rowValue(section, said.termsTaken)).toBe(said.termsTakenYes);
    });

    it("says when nothing was new and WordPress did not keep the categories", () => {
        render(
            <PublishPane
                payload={aPublishResult([], {
                    categories: {
                        taxonomy: "product_cat",
                        terms: [{ entityId: "e1", name: "Peptides", termId: 12, parentId: 0, created: false }],
                        previous: [12],
                        added: [],
                        taken: false,
                    },
                    findings: [
                        { severity: "warn", code: "categories_not_taken", message: "WordPress did not keep the categories Peptides on /a/" },
                    ],
                })}
            />,
        );

        const section = filedUnder();
        expect(within(section).getByText(said.filedUnderProducts)).toBeDefined();
        expect(rowValue(section, said.termsCreated)).toBe(said.termsCreatedNone);
        expect(rowValue(section, said.termsAdded)).toBe(said.termsAddedNone);
        expect(rowValue(section, said.termsTaken)).toBe(said.termsTakenNo);
        expect(screen.getByText(copy.runs.findingTitles["categories_not_taken"] ?? "")).toBeDefined();
        expect(screen.queryByText("categories_not_taken")).toBeNull();
    });

    it("leaves the categories out for a page filed under none", () => {
        render(<PublishPane payload={aPublishResult([])} />);
        expect(document.querySelector("[data-publish-categories]")).toBeNull();
    });
});

describe("the sync pane", () => {
    it("says a site sync could not read the site's categories", () => {
        render(
            <SyncPane
                payload={{
                    source: "plugin",
                    done: true,
                    findings: [{ severity: "warn", code: "categories_unread", message: "Crema Bench would not list its category terms" }],
                }}
            />,
        );
        expect(screen.getByText(copy.runs.findingTitles["categories_unread"] ?? "")).toBeDefined();
        expect(screen.getByText("Crema Bench would not list its category terms")).toBeDefined();
    });

    it("shows no finding list for a page read back cleanly", () => {
        render(<SyncPane payload={{ url: "", status: "publish", links: 3 }} />);
        expect(screen.queryByText(copy.runs.review.links.clean)).toBeNull();
    });
});

describe("the revert pane", () => {
    it("says what the revert did, and why the categories it created stay", () => {
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
                            severity: "info",
                            code: "revert_terms_kept",
                            message: "the categories the run created for /accessories/knock-boxes/ stay on the site",
                        },
                    ],
                }}
            />,
        );

        expect(screen.getByText(copy.runs.review.revert.outcomes["restored"] ?? "")).toBeDefined();
        expect(screen.getByText(copy.runs.review.revert.none)).toBeDefined();
        expect(screen.getByText(copy.runs.findingTitles["revert_terms_kept"] ?? "")).toBeDefined();
        expect(screen.getByText(/stay on the site$/)).toBeDefined();
    });
});
