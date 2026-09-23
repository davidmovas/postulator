import { describe, expect, it } from "vitest";

import type { Template } from "../../data/types.js";
import { defaultQuery } from "./params.js";
import { filtered, kindsOf, namesIn, ordered } from "./rows.js";

function template(name: string, scope: string, pageKind: string): Template {
    return {
        id: name,
        scope,
        siteId: scope === "site" ? "site-1" : null,
        name,
        pageKind,
        version: 1,
        spec: {} as Template["spec"],
        createdAt: "2026-09-01T00:00:00Z",
        updatedAt: "2026-09-01T00:00:00Z",
    };
}

const rows: readonly Template[] = [
    template("Guide", "global", "guide"),
    template("Hub", "global", "hub"),
    template("Kiln care guide", "site", "guide"),
    template("Product — spec table", "site", "product"),
];

describe("filtered", () => {
    it("shows everything when nothing narrows it", () => {
        expect(filtered(rows, defaultQuery)).toHaveLength(4);
    });

    it("keeps one scope", () => {
        expect(filtered(rows, { ...defaultQuery, scope: "site" }).map((held) => held.name)).toEqual([
            "Kiln care guide",
            "Product — spec table",
        ]);
    });

    it("keeps one page kind", () => {
        expect(filtered(rows, { ...defaultQuery, pageKind: "guide" }).map((held) => held.name)).toEqual([
            "Guide",
            "Kiln care guide",
        ]);
    });

    it("searches the name and the page kind, ignoring case", () => {
        expect(filtered(rows, { ...defaultQuery, search: "KILN" }).map((held) => held.name)).toEqual([
            "Kiln care guide",
        ]);
        expect(filtered(rows, { ...defaultQuery, search: "hub" }).map((held) => held.name)).toEqual(["Hub"]);
    });

    it("combines the facets", () => {
        expect(
            filtered(rows, { ...defaultQuery, scope: "global", pageKind: "guide", search: "gui" }).map(
                (held) => held.name,
            ),
        ).toEqual(["Guide"]);
    });

    it("ignores surrounding space in the search", () => {
        expect(filtered(rows, { ...defaultQuery, search: "   " })).toHaveLength(4);
    });
});

describe("ordered", () => {
    it("sorts the merged global and site rows by name, ignoring case", () => {
        expect(ordered(rows, { field: "name", desc: false }).map((held) => held.name)).toEqual([
            "Guide",
            "Hub",
            "Kiln care guide",
            "Product — spec table",
        ]);
    });

    it("reverses on demand", () => {
        expect(ordered(rows, { field: "name", desc: true }).map((held) => held.name)[0]).toBe(
            "Product — spec table",
        );
    });

    it("sorts by when the template was made", () => {
        const older = { ...template("Older", "site", "guide"), createdAt: "2026-01-01T00:00:00Z" };
        expect(ordered([...rows, older], { field: "createdAt", desc: false })[0].name).toBe("Older");
        expect(ordered([...rows, older], { field: "createdAt", desc: true }).at(-1)?.name).toBe("Older");
    });

    it("leaves the rows it was given alone", () => {
        const before = rows.map((held) => held.name);
        ordered(rows, { field: "name", desc: true });
        expect(rows.map((held) => held.name)).toEqual(before);
    });
});

describe("kindsOf", () => {
    it("lists every kind once, sorted, with the blanks dropped", () => {
        expect(kindsOf([...rows, template("Nameless", "site", "")])).toEqual(["guide", "hub", "product"]);
    });
});

describe("namesIn", () => {
    it("lists the names a new name in that scope must avoid", () => {
        expect(namesIn(rows, "site")).toEqual(["Kiln care guide", "Product — spec table"]);
    });
});
