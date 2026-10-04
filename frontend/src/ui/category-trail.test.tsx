import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { CategoryTrailItem } from "./category-trail.js";
import { CategoryTrail } from "./category-trail.js";
import { toneClasses } from "./tone.js";

const peptides: CategoryTrailItem = { key: "peptides", name: "Peptides", state: "onSite", hint: "WordPress category #12" };
const healing: CategoryTrailItem = {
    key: "healing",
    name: "Healing",
    state: "onPublish",
    hint: "Created on WordPress when a page under it is published",
};

function chipOf(name: string): HTMLElement {
    const item = screen.getByText(name).closest("li");
    if (item === null) {
        throw new Error(`no chip carries ${name}`);
    }
    return item;
}

describe("CategoryTrail", () => {
    it("lists the categories root first, joined by an arrow a screen reader skips", () => {
        render(<CategoryTrail items={[peptides, healing]} label="WordPress categories" />);

        const trail = screen.getByRole("list", { name: "WordPress categories" });
        const chips = within(trail).getAllByRole("listitem");
        expect(chips.map((chip) => chip.textContent?.replace("›", ""))).toStrictEqual(["Peptides", "Healing"]);
        const arrows = trail.querySelectorAll("[aria-hidden='true']");
        expect([...arrows].filter((arrow) => arrow.textContent === "›")).toHaveLength(1);
        expect(trail.querySelectorAll("svg")).toHaveLength(2);
    });

    it.each<[CategoryTrailItem["state"], string, string]>([
        ["onSite", "bg-inset", "WordPress category #12"],
        ["onPublish", "border-dashed", "Created on WordPress when a page under it is published"],
        ["needsPlugin", toneClasses.warn.ink, "Pages carry categories only with the companion plugin 1.3.0"],
        ["becomes", "border-dashed", "Becomes a WordPress category"],
        ["known", "border-hairline", "A category Postulator already holds"],
    ])("draws a %s chip apart and says what it means on hover", (state, mark, hint) => {
        render(<CategoryTrail items={[{ key: "one", name: "Peptides", state, hint }]} label="WordPress categories" />);

        const chip = chipOf("Peptides");
        expect(chip.dataset["categoryState"]).toBe(state);
        expect(chip.getAttribute("title")).toBe(hint);
        expect(screen.getByText("Peptides").parentElement?.className).toContain(mark);
    });

    it("tells the needs-plugin chip by its warning tone and the others by none", () => {
        render(
            <CategoryTrail
                items={[peptides, { ...healing, state: "needsPlugin", hint: "update the plugin" }]}
                label="WordPress categories"
            />,
        );

        expect(screen.getByText("Healing").parentElement?.className).toContain(toneClasses.warn.border);
        expect(screen.getByText("Peptides").parentElement?.className).not.toContain(toneClasses.warn.border);
    });

    it("keeps a compact trail on one line and puts the whole trail in one tooltip", () => {
        render(<CategoryTrail items={[peptides, healing]} label="WordPress categories" compact={true} />);

        const trail = screen.getByRole("list", { name: "WordPress categories" });
        expect(trail.className).toContain("flex-nowrap");
        expect(trail.className).toContain("overflow-hidden");
        expect(trail.getAttribute("title")).toBe(
            [
                "Peptides › Healing",
                "Peptides: WordPress category #12",
                "Healing: Created on WordPress when a page under it is published",
            ].join("\n"),
        );
        expect(chipOf("Peptides").getAttribute("title")).toBeNull();
        expect(screen.getByText("Healing").className).toContain("truncate");
    });

    it("lets the categories above the leaf give up their room first on one line", () => {
        render(<CategoryTrail items={[peptides, healing]} label="WordPress categories" compact={true} />);

        expect(chipOf("Peptides").className).toContain("shrink-[3]");
        expect(chipOf("Healing").className).not.toContain("shrink-[3]");
    });

    it("wraps a full trail and leaves the tooltips to each chip", () => {
        render(<CategoryTrail items={[peptides, healing]} label="WordPress categories" />);

        const trail = screen.getByRole("list", { name: "WordPress categories" });
        expect(trail.className).toContain("flex-wrap");
        expect(trail.getAttribute("title")).toBeNull();
    });

    it("draws nothing for a page no category files", () => {
        const { container } = render(<CategoryTrail items={[]} label="WordPress categories" />);
        expect(container.childElementCount).toBe(0);
    });
});
