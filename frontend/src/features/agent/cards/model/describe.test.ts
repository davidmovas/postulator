import { describe, expect, it } from "vitest";

import { copy } from "../../../../copy/index.js";
import type { TemplateSpec, ToolSchema } from "../../../../data/types.js";
import { aFinding } from "../../../../testing/findings.js";
import { findingCodeLabel } from "../../../pages/model/labels.js";
import { findingSentence } from "../../../findings/model/subject.js";
import { optionDoes, optionLabel } from "../../../findings/labels.js";
import { stepLabel } from "../../../templates/labels.js";
import type { Part } from "./card.js";
import { describeAction } from "./describe.js";
import { confirmableArgs, confirmableOps } from "./fixture.js";

const none = { currentTemplate: null, finding: null };

function texts(parts: readonly Part[]): string[] {
    return parts.filter((part): part is string => typeof part === "string");
}

function refs(parts: readonly Part[]): string[] {
    return parts.filter((part): part is Exclude<Part, string> => typeof part !== "string").map((part) => `${part.kind}:${part.id}`);
}

describe("every confirmable tool has a describer of its own", () => {
    const cases = [...Object.entries(confirmableArgs), ...confirmableOps];
    for (const [at, [tool, args]] of cases.entries()) {
        it(`${tool} ${typeof args["op"] === "string" ? args["op"] : ""} #${at} is described without JSON`, () => {
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
        expect(joined).toContain("Keywords travel mug (2,400), insulated mug, mug with lid");
        expect(joined).toContain("travel mugs");
    });
});

describe("the proposed cards", () => {
    it("reads approving the call as accepting the cards, each with its entity and what it changes", () => {
        const view = describeAction("graph_propose_cards", confirmableArgs["graph_propose_cards"] ?? {}, null, none);
        const said = copy.agent.describe.graph;
        expect(texts(view.title).join("")).toBe(said.proposeCards(2));
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["entity:e1", "entity:e2"]);
        const written = view.lines.map((line) => texts(line.parts).join(""));
        expect(written).toContain(said.proposeCardsBody);
        expect(written).toContain(said.definition("Mugs made of fired clay."));
        expect(written).toContain(said.fact("Material", "Stoneware"));
        expect(written).toContain(said.proposeCardsReason("The category page says so."));
        expect(written).toContain(said.factsCount(0));
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

describe("the revert card", () => {
    it("names the run it puts back and warns of what it takes off the site", () => {
        const view = describeAction("runs_revert", confirmableArgs["runs_revert"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe(`${copy.agent.describe.runs.revert} `);
        expect(refs(view.title)).toStrictEqual(["run:r1"]);
        expect(view.lines.map((line) => [line.tone, texts(line.parts).join("")])).toStrictEqual([
            ["warn", copy.agent.describe.runs.revertBody],
        ]);
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
        const sent = confirmableArgs["templates_create"]?.["spec"] as Record<string, unknown>;
        const stored = { ...sent, modelProfiles: { writer: { provider: "openai", model: "gpt-5" } } } as unknown as TemplateSpec;
        const changed = { ...sent, tone: "brisk", length: { min: 400, max: 800 } };
        const view = describeAction("templates_update", { id: "t1", spec: changed }, null, { currentTemplate: stored, finding: null });
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

describe("the page type cards", () => {
    const joinedOf = (tool: string, args?: Record<string, unknown>): string =>
        describeAction(tool, args ?? confirmableArgs[tool] ?? {}, null, none)
            .lines.map((line) => texts(line.parts).join(""))
            .join("\n");

    it("creates a type from a role on a site, with its template", () => {
        const view = describeAction("pagetypes_save", confirmableArgs["pagetypes_save"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe('Create page type "Category"');
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["site:s1", "template:t1"]);
        expect(joinedOf("pagetypes_save")).toContain(`Role ${copy.pageTypes.presets.hub}`);
    });

    it("renames a type and clears its template in words", () => {
        const view = describeAction("pagetypes_save", { id: "pt-hub", name: "Hubs", templateId: "" }, null, none);
        expect(refs(view.title)).toStrictEqual(["pagetype:pt-hub"]);
        const joined = view.lines.map((line) => texts(line.parts).join("")).join("\n");
        expect(joined).toContain('Renamed to "Hubs"');
        expect(joined).toContain(copy.agent.describe.pagetypes.noTemplate);
    });

    it("says where a deleted type's pages go", () => {
        const view = describeAction("pagetypes_delete", confirmableArgs["pagetypes_delete"] ?? {}, null, none);
        expect(refs(view.title)).toStrictEqual(["pagetype:pt-guide"]);
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["pagetype:pt-entity"]);
        expect(view.lines[0]?.tone).toBe("danger");
    });

    it("reads a scope and its link policy as sentences, never as JSON", () => {
        const view = describeAction("pagetypes_set_scope", confirmableArgs["pagetypes_set_scope"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe(`Set ${copy.pageTypes.fields.body} on `);
        expect(refs(view.title)).toStrictEqual(["pagetype:pt-hub"]);
        const joined = joinedOf("pagetypes_set_scope");
        expect(joined).toContain("Open with what the category covers.");
        expect(joined).toContain("At most 6000 characters");
        expect(joined).toContain("At most 20 links");
        expect(joined).toContain(`Anchors: ${copy.pageTypes.anchors.rotate}`);
        expect(joined).not.toMatch(/[{}[\]]/);
    });

    it("names a custom field by its label", () => {
        const view = describeAction(
            "pagetypes_set_scope",
            { pageTypeId: "pt-hub", metaKey: "dosage_note", label: "Dosage note", provider: "plugin" },
            null,
            none,
        );
        expect(texts(view.title).join("")).toBe("Set Dosage note on ");
        expect(view.lines.map((line) => texts(line.parts).join("")).join("\n")).toContain(
            `Stored by ${copy.pageTypes.providers.plugin}`,
        );
    });

    it("types a page through pages_update", () => {
        const view = describeAction("pages_update", confirmableArgs["pages_update"] ?? {}, null, none);
        expect(view.lines.flatMap((line) => refs(line.parts))).toContain("pagetype:pt-hub");
    });

    it("lists the card an entity update carries", () => {
        const joined = joinedOf("graph_update_entity");
        expect(joined).toContain("Definition: Mugs thrown and fired from stoneware clay.");
        expect(joined).toContain("Capacity: 350 ml");
        expect(joined).toContain('Note "Tone"');
    });
});

describe("the agent's edits to a page", () => {
    const linesOf = (tool: string, args: Record<string, unknown>): string[] =>
        describeAction(tool, args, null, none).lines.map((line) => texts(line.parts).join(""));
    const said = copy.agent.describe;

    it("names the field it sets, the page and the new value", () => {
        const view = describeAction("pages_fields", confirmableArgs["pages_fields"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe("Set the SEO description of ");
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        expect(linesOf("pages_fields", confirmableArgs["pages_fields"] ?? {})).toContain(
            said.fields.value("Hand-thrown mugs that keep coffee hot."),
        );
    });

    it("names a page by its path when the call gave no id, and says an empty value clears the field", () => {
        const args = { op: "set", path: "/mugs/", key: "title", value: "", hash: "" };
        const view = describeAction("pages_fields", args, null, none);
        expect(texts(view.title).join("")).toBe("Set the title of /mugs/");
        expect(view.lines.map((line) => [line.tone, texts(line.parts).join("")])).toContainEqual(["warn", said.fields.clears]);
    });

    it("shows a replaced block's new html as its text, never as markup", () => {
        const view = describeAction("pages_blocks", confirmableArgs["pages_blocks"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe("Replace block 11 of ");
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        const written = linesOf("pages_blocks", confirmableArgs["pages_blocks"] ?? {});
        expect(written).toContain(said.blocks.text("Fired at 1240 °C & glazed."));
        expect(written.join("\n")).not.toMatch(/[<>]/);
    });

    it("inserts at the start of the body or after a block", () => {
        expect(texts(describeAction("pages_blocks", { op: "insert_after", pageId: "p1", index: -1, html: "<p>A</p>" }, null, none).title).join("")).toBe(
            "Insert a block at the start of ",
        );
        expect(texts(describeAction("pages_blocks", { op: "insert_after", pageId: "p1", index: 3, html: "<p>A</p>" }, null, none).title).join("")).toBe(
            "Insert a block after block 3 of ",
        );
    });

    it("reads a deleted block as a danger line", () => {
        const view = describeAction("pages_blocks", { op: "delete", path: "/mugs/", index: 4, hash: "b1" }, null, none);
        expect(texts(view.title).join("")).toBe("Delete block 4 of /mugs/");
        expect(view.lines.map((line) => [line.tone, texts(line.parts).join("")])).toContainEqual(["danger", said.blocks.deleteBody]);
    });

    it("names the anchor a placed link takes and the page it points to", () => {
        const args = { op: "place_link", pageId: "p1", index: 2, hash: "b1", targetId: "p2", anchor: "travel mugs" };
        const view = describeAction("pages_blocks", args, null, none);
        expect(texts(view.title).join("")).toBe(said.blocks.placeLink("travel mugs", 2));
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["page:p2"]);
    });

    it("keeps no card of its own for a read", () => {
        expect(describeAction("pages_blocks", { op: "list", pageId: "p1" }, null, none).generic).toBe(true);
        expect(describeAction("pages_fields", { op: "get", pageId: "p1" }, null, none).generic).toBe(true);
        expect(describeAction("findings", { op: "list" }, null, none).generic).toBe(true);
    });
});

describe("the finding card", () => {
    const finding = aFinding();
    const context = { currentTemplate: null, finding };

    it("names the code, the page and the subject in words, then what the option does", () => {
        const view = describeAction("findings", confirmableArgs["findings"] ?? {}, null, context);
        expect(texts(view.title).join("")).toBe(copy.agent.describe.findings.resolve(findingCodeLabel(finding.code)));
        expect(refs(view.title)).toStrictEqual(["page:p1"]);
        const written = view.lines.map((line) => texts(line.parts).join(""));
        expect(written).toContain(findingSentence(finding.code, finding.subject));
        expect(written).toContain(copy.agent.describe.findings.option(optionLabel("accept_site"), optionDoes("accept_site")));
    });

    it("says only the option while the finding is not read yet", () => {
        const view = describeAction("findings", confirmableArgs["findings"] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe(copy.agent.describe.findings.resolveAny);
        expect(view.lines).toHaveLength(1);
    });

    it("names the page a row maps to and the type a page is given", () => {
        const mapped = describeAction("findings", { op: "resolve", findingId: "f2", option: "map_to_page", pageId: "p3" }, null, none);
        expect(mapped.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["page:p3"]);
        const typed = describeAction("findings", { op: "resolve", findingId: "f3", option: "set_type", pageTypeId: "pt-hub" }, null, none);
        expect(typed.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["pagetype:pt-hub"]);
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
