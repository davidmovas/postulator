import { describe, expect, it } from "vitest";

import { aFinding } from "../../../testing/findings.js";
import { batches, bulkLimit, bulkOptions } from "./bulk.js";

const differs = aFinding({
    id: "f1",
    code: "parent_differs",
    options: [
        { name: "accept_site", kind: "model" },
        { name: "accept_workbook", kind: "model" },
        { name: "ignore", kind: "model" },
    ],
});

const changed = aFinding({ id: "f2" });

describe("the bulk actions a code offers", () => {
    it("offers every bulk option the code's findings carry, in the bulk order", () => {
        expect(bulkOptions("parent_differs", "open", [differs])).toStrictEqual(["ignore", "accept_site", "accept_workbook"]);
    });

    it("never offers a run option or one chosen a finding at a time", () => {
        const proposed = aFinding({ code: "entity_card_proposed", options: [{ name: "accept", kind: "model" }, { name: "ignore", kind: "model" }] });
        expect(bulkOptions("field_changed_on_site", "open", [changed])).toStrictEqual(["ignore", "accept_site"]);
        expect(bulkOptions("entity_card_proposed", "open", [proposed])).toStrictEqual(["ignore"]);
    });

    it("offers set_type to a code whose findings carry it", () => {
        const technical = aFinding({ code: "technical_detected", options: [{ name: "set_type", kind: "model" }, { name: "ignore", kind: "model" }] });
        expect(bulkOptions("technical_detected", "open", [technical])).toStrictEqual(["ignore", "set_type"]);
    });

    it("offers nothing without one code chosen, or outside the open findings", () => {
        expect(bulkOptions("", "open", [differs])).toStrictEqual([]);
        expect(bulkOptions("parent_differs", "dismissed", [differs])).toStrictEqual([]);
    });

    it("reads the options from the findings of that code alone", () => {
        expect(bulkOptions("parent_differs", "open", [changed])).toStrictEqual([]);
        expect(bulkOptions("parent_differs", "open", [aFinding({ code: "parent_differs", status: "resolved", options: differs.options })])).toStrictEqual([]);
    });
});

describe("the batches a bulk action sends", () => {
    it("sends at most the limit the site takes in one call", () => {
        const ids = Array.from({ length: bulkLimit * 2 + 3 }, (_, at) => `f${at}`);
        const sent = batches(ids);
        expect(sent.map((batch) => batch.length)).toStrictEqual([bulkLimit, bulkLimit, 3]);
        expect(sent.flat()).toStrictEqual(ids);
    });

    it("sends nothing for no ids", () => {
        expect(batches([])).toStrictEqual([]);
    });
});
