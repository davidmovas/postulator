import { describe, expect, it } from "vitest";

import generatedSource from "./generated.tsx?raw";
import iconList from "../../../scripts/icons.txt?raw";

const listed = iconList
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0);

const exported = [...generatedSource.matchAll(/^export const (\w+) = createIcon\("([^"]+)"\);$/gm)];

function expectedName(icon: string): string {
    return `${icon
        .split("_")
        .map((part) => part[0].toUpperCase() + part.slice(1))
        .join("")}Icon`;
}

describe("the generated icon module", () => {
    it("exports one component per listed name, in order, and nothing else", () => {
        expect(exported.map((match) => match[1])).toStrictEqual(listed.map(expectedName));
    });

    it("has no export the regenerator would not produce", () => {
        const exportLines = generatedSource.split("\n").filter((line) => line.startsWith("export "));
        expect(exportLines).toHaveLength(listed.length);
    });

    it("carries a non-empty path for every icon", () => {
        expect(exported.filter((match) => match[2].length < 8).map((match) => match[1])).toStrictEqual([]);
    });

    it("lists names sorted and unique so the module order is stable", () => {
        expect(listed).toStrictEqual([...new Set(listed)].sort());
    });
});
