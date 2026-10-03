import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { entity } from "../model/fixture.js";
import { expandAll, visibleRows } from "../model/fold.js";
import { buildGraphIndex } from "../model/index.js";
import { OutlineView } from "./view.js";

function outline(kind: string) {
    const index = buildGraphIndex([entity({ id: "mugs", name: "Ceramic Mugs", kind })], []);
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

describe("the outline's kind column", () => {
    it.each([
        ["hub", copy.graph.kinds.hub],
        ["product", copy.graph.kinds.product],
        ["category", copy.graph.kinds.category],
    ])("names a %s in words", (kind, words) => {
        outline(kind);
        expect(screen.getByText(words)).toBeDefined();
        expect(screen.queryByText(kind)).toBeNull();
    });

    it("shows a kind this build does not know as it arrived", () => {
        outline("galaxy");
        expect(screen.getByText("galaxy")).toBeDefined();
    });
});
