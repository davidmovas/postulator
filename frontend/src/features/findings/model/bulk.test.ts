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

const names = (code: string, findings: Parameters<typeof bulkOptions>[2]): string[] =>
    bulkOptions(code, "open", findings).map((choice) => choice.name);

describe("the bulk actions a code offers", () => {
    it("offers every bulk option the code's findings carry, in the bulk order", () => {
        expect(bulkOptions("parent_differs", "open", [differs])).toStrictEqual([
            { name: "ignore", kind: "model" },
            { name: "accept_site", kind: "model" },
            { name: "accept_workbook", kind: "model" },
        ]);
    });

    it("never offers a run option or one chosen a finding at a time", () => {
        const proposed = aFinding({ code: "entity_card_proposed", options: [{ name: "accept", kind: "model" }, { name: "ignore", kind: "model" }] });
        expect(names("field_changed_on_site", [changed])).toStrictEqual(["ignore", "accept_site"]);
        expect(names("entity_card_proposed", [proposed])).toStrictEqual(["ignore"]);
    });

    it("offers set_type to a code whose findings carry it", () => {
        const technical = aFinding({ code: "technical_detected", options: [{ name: "set_type", kind: "model" }, { name: "ignore", kind: "model" }] });
        expect(names("technical_detected", [technical])).toStrictEqual(["ignore", "set_type"]);
    });

    it("offers a proposal option with its kind, so the inbox opens the proposals instead of settling", () => {
        const without = aFinding({
            code: "page_without_entity",
            options: [
                { name: "propose_entity", kind: "proposal" },
                { name: "ignore", kind: "model" },
            ],
        });
        expect(bulkOptions("page_without_entity", "open", [without])).toStrictEqual([
            { name: "ignore", kind: "model" },
            { name: "propose_entity", kind: "proposal" },
        ]);
    });

    it("offers nothing without one code chosen, or outside the open findings", () => {
        expect(bulkOptions("", "open", [differs])).toStrictEqual([]);
        expect(bulkOptions("parent_differs", "dismissed", [differs])).toStrictEqual([]);
    });

    it("reads the options from the findings of that code alone", () => {
        expect(names("parent_differs", [changed])).toStrictEqual([]);
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
