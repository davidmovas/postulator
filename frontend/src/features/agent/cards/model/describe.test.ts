import { describe, expect, it } from "vitest";

import type { ToolSchema } from "../../../../data/types.js";
import { stepLabel } from "../../../templates/labels.js";
import type { Part } from "./card.js";
import { describeAction } from "./describe.js";
import { confirmableArgs } from "./fixture.js";

const none = { currentTemplate: null };

function texts(parts: readonly Part[]): string[] {
    return parts.filter((part): part is string => typeof part === "string");
}

function refs(parts: readonly Part[]): string[] {
    return parts.filter((part): part is Exclude<Part, string> => typeof part !== "string").map((part) => `${part.kind}:${part.id}`);
}

describe("every confirmable tool has a describer of its own", () => {
    for (const [tool, args] of Object.entries(confirmableArgs)) {
        it(`${tool} is described without JSON`, () => {
            const view = describeAction(tool, args, null, none);
            expect(view.generic).toBe(false);
            expect(view.title.length).toBeGreaterThan(0);
            const strings = [...texts(view.title), ...view.lines.flatMap((line) => texts(line.parts))];
            for (const held of strings) {
                expect(held).not.toMatch(/"\s*:|\{"|"\}|\[\{|\}\]/);
            }
        });
    }
});

describe("the graph cards", () => {
    it("connects two entities as a sentence with both ends resolvable", () => {
        const view = describeAction("graph_add_edge", confirmableArgs["graph_add_edge"] ?? {}, null, none);
        expect(refs(view.title)).toStrictEqual(["entity:e2", "entity:e1"]);
        expect(texts(view.title).join("")).toBe("Connect  under ");
        expect(view.lines.some((line) => texts(line.parts).join("").includes("Travel mugs are a kind of mug."))).toBe(true);
    });

    it("names the entity it creates and counts its keywords and anchors", () => {
        const view = describeAction("graph_create_entity", confirmableArgs["graph_create_entity"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe('Create entity "Travel Mugs"');
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).toContain("travel mug");
        expect(joined).toContain("insulated mug, mug with lid");
        expect(joined).toContain("travel mugs");
    });
});

describe("the run card", () => {
    it("lists the pages, the steps, the publish mode and the budget", () => {
        const view = describeAction("runs_start", confirmableArgs["runs_start"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe("Start a generate run over 3 pages");
        expect(view.lines.flatMap((line) => refs(line.parts))).toEqual(expect.arrayContaining(["page:p1", "page:p2", "page:p3", "template:t1"]));
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).toContain(`${stepLabel("generate_body")}, ${stepLabel("publish")}`);
        expect(joined).toContain("$12.50");
        expect(view.lines.some((line) => line.tone === "warn" && texts(line.parts).join("").includes("Publishes"))).toBe(true);
    });
});

describe("the template cards", () => {
    it("reads a created template's spec as sentences", () => {
        const view = describeAction("templates_create", confirmableArgs["templates_create"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe('Create template "Product page"');
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).toContain("Overview");
        expect(joined).toContain("Care");
        expect(joined).toContain("600");
        expect(joined).not.toContain("targetWords");
    });

    it("describes an update as what changes against the current template", () => {
        const current = JSON.parse(confirmableArgs["templates_create"]?.["spec"] as string);
        const changed = { ...current, tone: "brisk", length: { min: 400, max: 800 } };
        const view = describeAction("templates_update", { id: "t1", spec: JSON.stringify(changed) }, null, { currentTemplate: current });
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).toContain("brisk");
        expect(joined).toContain("400");
        expect(joined).not.toContain("Overview");
    });

    it("reads an override patch as sentences", () => {
        const view = describeAction("templates_set_override", confirmableArgs["templates_set_override"] ?? {}, null, none);
        expect(refs(view.title)).toStrictEqual(["template:t1", "page:p1"]);
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).toContain("playful");
    });
});

describe("the generic describer", () => {
    it("falls back to the schema for a tool nobody described", () => {
        const schema = JSON.parse(
            JSON.stringify({
                type: "object",
                properties: {
                    count: { type: "integer", description: "How many" },
                    enabled: { type: "boolean" },
                    names: { type: "array", items: { type: "string" } },
                    nested: { type: "object", properties: { deep: { type: "string" } } },
                },
            }),
        ) as ToolSchema;
        const view = describeAction("mystery_do_thing", { count: 3, enabled: true, names: ["a", "b"], nested: { deep: "x" } }, schema, none);
        expect(view.generic).toBe(true);
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).toContain("How many: 3");
        expect(joined).toContain("enabled: yes");
        expect(joined).toContain("a, b");
        expect(joined).toContain("deep: x");
        expect(joined).not.toMatch(/[{}[\]]/);
    });

    it("hides a masked secret rather than printing it", () => {
        const view = describeAction("models_set_provider_key", confirmableArgs["models_set_provider_key"] ?? {}, null, none);
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).not.toContain("***");
    });
});
