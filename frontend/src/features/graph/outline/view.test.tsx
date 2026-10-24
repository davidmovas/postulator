import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { entity, pageType, typedPage } from "../model/fixture.js";
import { expandAll, visibleRows } from "../model/fold.js";
import { buildGraphIndex } from "../model/index.js";
import { OutlineView } from "./view.js";

function outline(preset: string | null, name = "Hub") {
    const index = buildGraphIndex(
        [entity({ id: "mugs", name: "Ceramic Mugs" })],
        [],
        preset === null ? [] : [typedPage("mugs", preset)],
        preset === null ? [] : [pageType(preset, name)],
    );
    return render(
        <OutlineView
            siteId="s1"
            index={index}
            rows={visibleRows(index, expandAll(index), "score")}
            selectedId={null}
            matched={null}
            onSelect={() => {}}
            onPick={() => {}}
            onToggle={() => {}}
            onLiftMore={() => {}}
        />,
    );
}

describe("the outline's type column", () => {
    it("names the canonical page's type by the client's own name", () => {
        outline("hub", "Category");
        expect(screen.getByText("Category")).toBeDefined();
        expect(screen.getByText(copy.graph.outline.columns.type)).toBeDefined();
    });

    it("says an entity without a typed page has no page type", () => {
        outline(null);
        expect(screen.getByText(copy.pageTypes.untyped)).toBeDefined();
    });
});
