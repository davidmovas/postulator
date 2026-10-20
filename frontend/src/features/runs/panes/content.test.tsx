import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { fieldsView } from "../artifacts.js";
import { FieldsPane } from "./content.js";

const said = copy.runs.review.fields;

describe("the fields artifact", () => {
    it("reads the values and the skipped fields, leaving out a row with no key", () => {
        expect(
            fieldsView({
                values: [{ key: "excerpt", value: "A short summary." }, { key: "", value: "lost" }],
                skipped: [{ key: "meta:subtitle", reason: "nothing on this site writes it yet" }],
            }),
        ).toStrictEqual({
            values: [{ key: "excerpt", value: "A short summary." }],
            skipped: [{ key: "meta:subtitle", reason: "nothing on this site writes it yet" }],
        });
    });

    it("reads the short description, which is HTML, as the text a buyer sees", () => {
        expect(
            fieldsView({
                values: [
                    { key: "product_short_description", value: "<p>A compact machine &amp; grinder.</p>\n<p>For the counter.</p>" },
                    { key: "excerpt", value: "Fits 5 <b> 7" },
                ],
                skipped: [],
            })?.values,
        ).toStrictEqual([
            { key: "product_short_description", value: "A compact machine & grinder. For the counter." },
            { key: "excerpt", value: "Fits 5 <b> 7" },
        ]);
    });

    it("refuses what is not an object", () => {
        expect(fieldsView(null)).toBeNull();
        expect(fieldsView([])).toBeNull();
    });
});

describe("the fields pane", () => {
    it("shows each written field by its name with its value, and each skipped one with its reason", () => {
        render(
            <FieldsPane
                payload={{
                    values: [{ key: "excerpt", value: "A short summary." }],
                    skipped: [{ key: "meta:subtitle", reason: "nothing on this site writes it yet" }],
                }}
            />,
        );

        const written = within(screen.getByRole("region", { name: said.written }));
        expect(written.getByText(copy.pageTypes.fields.excerpt)).toBeDefined();
        expect(written.getByText("A short summary.")).toBeDefined();

        const skipped = within(screen.getByRole("region", { name: said.skipped }));
        expect(skipped.getByText("subtitle")).toBeDefined();
        expect(skipped.getByText("nothing on this site writes it yet")).toBeDefined();
    });

    it("says so when the step asked for nothing", () => {
        render(<FieldsPane payload={{ values: [], skipped: [] }} />);

        expect(screen.getByText(said.none)).toBeDefined();
    });
});
