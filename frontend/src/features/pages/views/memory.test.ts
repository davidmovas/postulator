import { afterEach, describe, expect, it } from "vitest";

import { noSelection, toggle } from "../../../ui/selection.js";
import { blankPages, forgetPages, pagesMemoryOf, rememberPages, rememberSelection } from "./memory.js";

afterEach(() => {
    forgetPages("s1");
    forgetPages("s2");
});

describe("what the Pages screen remembers of a site", () => {
    it("starts on the list with nothing selected in any view", () => {
        const held = pagesMemoryOf("s1");
        expect(held).toBe(blankPages);
        expect(held.view).toBe("list");
        expect(held.openType).toBeNull();
        expect(held.groupMode).toBe("list");
        expect(Object.values(held.selections).every((selection) => selection === noSelection)).toBe(true);
    });

    it("keeps each site's view and open type apart", () => {
        rememberPages("s1", { view: "tree" });
        rememberPages("s1", { openType: "t-guide", groupMode: "tree" });

        expect(pagesMemoryOf("s1")).toMatchObject({ view: "tree", openType: "t-guide", groupMode: "tree" });
        expect(pagesMemoryOf("s2")).toBe(blankPages);
    });

    it("keeps one selection per view and hands back the same value until it changes", () => {
        const picked = toggle(noSelection, "p1") ?? noSelection;
        rememberSelection("s1", "tree", picked);
        const held = pagesMemoryOf("s1");

        expect(held.selections.tree).toBe(picked);
        expect(held.selections.list).toBe(noSelection);
        rememberSelection("s1", "tree", picked);
        expect(pagesMemoryOf("s1")).toBe(held);
    });

    it("forgets a site back to blank", () => {
        rememberPages("s1", { view: "types" });
        forgetPages("s1");
        expect(pagesMemoryOf("s1")).toBe(blankPages);
    });
});
