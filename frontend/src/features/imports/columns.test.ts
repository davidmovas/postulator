import { describe, expect, it } from "vitest";

import { importFields } from "../../generated/vocab.js";
import { assign, mappedFields, takenFrom, targetOf, unmappedHeaders, usable } from "./columns.js";

const detected = { path: "URL", title: "Title", keywords: "Keywords" };

describe("targetOf", () => {
    it("finds the field a column was given to", () => {
        expect(targetOf(detected, "URL")).toBe("path");
        expect(targetOf(detected, "Keywords")).toBe("keywords");
    });

    it("answers nothing for a column nobody wants", () => {
        expect(targetOf(detected, "Notes")).toBeNull();
        expect(targetOf(detected, "")).toBeNull();
        expect(targetOf(null, "URL")).toBeNull();
    });
});

describe("assign", () => {
    it("gives a column to a field", () => {
        expect(assign(detected, "Notes", "h1")).toEqual({ ...detected, h1: "Notes" });
    });

    it("takes the column away from the field that held it", () => {
        expect(assign(detected, "URL", "h1")).toEqual({ h1: "URL", title: "Title", keywords: "Keywords" });
    });

    it("moves a field rather than letting two columns claim it", () => {
        const moved = assign(detected, "Slug", "path");
        expect(moved["path"]).toBe("Slug");
        expect(Object.values(moved).filter((held) => held === "URL")).toHaveLength(0);
    });

    it("ignores a column", () => {
        expect(assign(detected, "URL", null)).toEqual({ title: "Title", keywords: "Keywords" });
    });

    it("starts a map from nothing", () => {
        expect(assign(null, "URL", "path")).toEqual({ path: "URL" });
    });

    it("drops an empty column it was handed", () => {
        expect(assign({ path: "URL", title: "" }, "Notes", "h1")).toEqual({ path: "URL", h1: "Notes" });
    });
});

describe("what the mapping covers", () => {
    it("lists the mapped fields in the order the vocabulary fixes", () => {
        expect(mappedFields({ title: "Title", path: "URL" })).toEqual(["path", "title"]);
        expect(mappedFields(null)).toEqual([]);
    });

    it("lists the columns nothing will read", () => {
        expect(unmappedHeaders(["URL", "Title", "Notes", ""], detected)).toEqual(["Notes"]);
    });

    it("needs a path or an entity to be worth applying", () => {
        expect(usable({ path: "URL" })).toBe(true);
        expect(usable({ entity: "Cluster" })).toBe(true);
        expect(usable({ title: "Title" })).toBe(false);
        expect(usable(null)).toBe(false);
    });

    it("remembers which targets the detector chose", () => {
        expect(takenFrom(detected, detected, "URL")).toBe(true);
        expect(takenFrom({ h1: "URL" }, detected, "URL")).toBe(false);
        expect(takenFrom(detected, detected, "Notes")).toBe(false);
    });

    it("knows every field the backend accepts", () => {
        expect(importFields).toContain("path");
        expect(importFields).toContain("entity");
    });
});
