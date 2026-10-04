import { describe, expect, it } from "vitest";

import { nameDiffers, productOutputsOf } from "./product.js";

describe("the product outputs a run last wrote", () => {
    it("reads the short description and the attributes", () => {
        expect(
            productOutputsOf({
                shortDescription: "<p>Pull a <strong>shot</strong> &amp; steam</p>",
                specifications: [{ name: "Form", value: "Countertop" }, { name: "Size", value: "" }, "not a row"],
            }),
        ).toStrictEqual({
            shortDescription: "Pull a shot & steam",
            specifications: [{ name: "Form", value: "Countertop" }],
        });
    });

    it.each([null, undefined, "text", [], { specifications: "none" }])("reads nothing out of %j", (raw) => {
        const read = productOutputsOf(raw);
        expect(read === null || (read.shortDescription === "" && read.specifications.length === 0)).toBe(true);
    });
});

describe("whether the sheet names a product otherwise", () => {
    it.each([
        { h1: "", name: "Espresso Machine", differs: false },
        { h1: "espresso machine", name: "Espresso Machine", differs: false },
        { h1: "Espresso Maker", name: "Espresso Machine", differs: true },
        { h1: "Espresso Maker", name: "", differs: false },
    ])("$h1 against $name", ({ h1, name, differs }) => {
        expect(nameDiffers(h1, name)).toBe(differs);
    });
});
