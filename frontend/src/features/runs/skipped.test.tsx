import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { renderScreen } from "../../testing/render.js";
import { aSkippedPage } from "../../testing/runs.js";
import { SkippedPages } from "./skipped.js";

describe("the pages a run leaves out", () => {
    it("says nothing when every chosen page is written", () => {
        const { container } = renderScreen(<SkippedPages skipped={[]} title="left out" />);

        expect(container.querySelector("[data-run-skipped]")).toBeNull();
    });

    it("names each page with its reason and says where each reason is fixed", () => {
        renderScreen(
            <SkippedPages
                skipped={[aSkippedPage("/shop/", "no_template", "Store category"), aSkippedPage("/cart/", "no_entity", "Technical")]}
                title={copy.runs.skipped.starting(2)}
            />,
        );

        expect(screen.getByText(copy.runs.skipped.starting(2))).toBeDefined();
        expect(screen.getByText("/shop/").closest("li")?.textContent).toBe(
            `/shop/ ${copy.runs.skipped.noTemplate("Store category")}`,
        );
        expect(screen.getByText("/cart/").closest("li")?.textContent).toBe(`/cart/ ${copy.runs.skipped.noEntity}`);
        expect(screen.getByText(copy.runs.skipped.fixTemplate)).toBeDefined();
        expect(screen.getByText(copy.runs.skipped.fixEntity)).toBeDefined();
    });

    it("lists the first pages and counts the rest", () => {
        const many = Array.from({ length: 11 }, (_, at) => aSkippedPage(`/shop/${String(at)}/`, "no_template", "Store category"));
        renderScreen(<SkippedPages skipped={many} title={copy.runs.skipped.starting(11)} />);

        expect(screen.getByText(copy.runs.skipped.more(3))).toBeDefined();
        expect(screen.queryByText("/shop/10/")).toBeNull();
        expect(screen.queryByText(copy.runs.skipped.fixEntity)).toBeNull();
    });
});
