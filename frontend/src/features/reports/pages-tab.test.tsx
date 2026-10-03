import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { Page, PageReport } from "../../data/types.js";
import { aPage } from "../../testing/pages.js";
import { renderScreen } from "../../testing/render.js";
import { toneClasses } from "../../ui/index.js";

const held: { pages: Page[]; report: PageReport | undefined } = { pages: [], report: undefined };

vi.mock("../../data/hooks/pages.js", () => ({
    usePages: () => ({ data: { pages: [{ items: held.pages, hasMore: false }] }, isPending: false, error: null }),
}));

vi.mock("../../data/hooks/reports.js", () => ({
    usePageReport: () => ({ data: held.report, isPending: held.report === undefined, error: null }),
}));

const { PagesTab } = await import("./pages-tab.js");

function reportOf(status: string): PageReport {
    return { pageId: "p1", path: "/mugs/", runId: "r1", itemId: "i1", status };
}

function show(pageStatus: string, itemStatus: string) {
    held.pages = [aPage("p1", "/mugs/", { status: pageStatus })];
    held.report = reportOf(itemStatus);
    return renderScreen(
        <PagesTab siteId="s1" prefix="" pageId="p1" onPrefix={() => {}} onSelect={() => {}} />,
    );
}

describe("the page report's header", () => {
    it.each([
        ["published", "failed", "danger"],
        ["planned", "completed", "ok"],
        ["published", "paused", "warn"],
    ] as const)("colours a %s page's %s item by the item", (pageStatus, itemStatus, tone) => {
        show(pageStatus, itemStatus);
        const badge = screen.getByText(copy.runs.status[itemStatus]);
        expect(badge.className).toContain(toneClasses[tone].ink);
    });
});
