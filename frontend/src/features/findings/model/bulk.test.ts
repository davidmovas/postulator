import { describe, expect, it } from "vitest";

import { messages } from "../../../data/errors.js";
import { aFinding } from "../../../testing/findings.js";
import { batches, bulkLimit, bulkOptions, refusalOf, settleInBatches } from "./bulk.js";

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

const refused = (code: string, message: string): Error => new Error(message, { cause: { code, message } });

describe("what a bulk action settled, batch by batch", () => {
    const ids = Array.from({ length: bulkLimit * 2 + 500 }, (_, at) => `f${at}`);

    it("settles every batch in turn and reports all of them settled", async () => {
        const sent: number[] = [];
        const outcome = await settleInBatches(ids, (batch) => {
            sent.push(batch.length);
            return Promise.resolve();
        });
        expect(sent).toStrictEqual([bulkLimit, bulkLimit, 500]);
        expect(outcome).toStrictEqual({ settled: ids.length, total: ids.length, refusal: null });
    });

    it("stops at the first refused batch, the earlier ones staying settled and the later ones never sent", async () => {
        const conflict = refused("CONFLICT", "the finding is no longer open");
        const sent: string[] = [];
        const outcome = await settleInBatches(ids, (batch) => {
            sent.push(batch[0]);
            return sent.length === 2 ? Promise.reject(conflict) : Promise.resolve();
        });
        expect(sent).toStrictEqual(["f0", `f${bulkLimit}`]);
        expect(outcome).toStrictEqual({ settled: bulkLimit, total: ids.length, refusal: conflict });
    });

    it("reports nothing settled when the first batch is refused", async () => {
        const invalid = refused("INVALID", "the page has no observed value for this field");
        const outcome = await settleInBatches(ids, () => Promise.reject(invalid));
        expect(outcome).toStrictEqual({ settled: 0, total: ids.length, refusal: invalid });
    });
});

describe("why a batch was refused, in plain words", () => {
    it("says what the site's model refused", () => {
        expect(refusalOf(refused("CONFLICT", "the finding is no longer open"))).toBe("the finding is no longer open");
        expect(refusalOf(refused("INVALID", "the page has no observed value for this field"))).toBe(
            "the page has no observed value for this field",
        );
    });

    it("keeps an internal failure's own words out of the sentence", () => {
        expect(refusalOf(refused("INTERNAL", "database is locked"))).toBe(messages.INTERNAL);
        expect(refusalOf(new Error("lost"))).toBe(messages.INTERNAL);
    });
});
