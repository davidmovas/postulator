import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import type { Estimate } from "../../data/types.js";
import { StartPrice } from "./start-price.js";

type Finding = Omit<NonNullable<Estimate["findings"]>[number], "severity"> & { severity: "info" | "warn" | "error" };

function priced(findings: readonly Finding[]): Estimate {
    return { tokens: 1200, usd: 0.04, findings: findings as unknown as Estimate["findings"] };
}

describe("the findings of a run's estimate", () => {
    it("heads a category finding with what to do and keeps the page and the details beneath", () => {
        render(
            <StartPrice
                estimate={priced([
                    {
                        severity: "warn",
                        code: "page_categories_need_plugin",
                        message: "the page goes up without them",
                        pageId: "p1",
                        path: "/peptides/healing/",
                    },
                ])}
                added={[]}
                over={false}
                refusal={null}
            />,
        );

        expect(screen.getByText(copy.runs.findingTitles["page_categories_need_plugin"] ?? "")).toBeDefined();
        expect(screen.getByText(copy.runs.start.findingAt("/peptides/healing/", "the page goes up without them"))).toBeDefined();
    });

    it("shows a finding it has no headline for as its own sentence", () => {
        render(
            <StartPrice
                estimate={priced([{ severity: "warn", code: "seo_meta_skipped", message: "no SEO plugin", path: "/a/" }])}
                added={[]}
                over={false}
                refusal={null}
            />,
        );

        expect(screen.getByText(copy.runs.start.findingAt("/a/", "no SEO plugin"))).toBeDefined();
    });
});
