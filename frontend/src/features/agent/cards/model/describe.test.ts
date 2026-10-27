import { describe, expect, it } from "vitest";

import { copy } from "../../../../copy/index.js";
import type { TemplateSpec, ToolSchema } from "../../../../data/types.js";
import { stepLabel } from "../../../templates/labels.js";
import type { CardView, Part } from "./card.js";
import { describeAction } from "./describe.js";
import { argsOf, confirmable } from "./fixture.js";

const none = { currentTemplate: null, finding: null, summary: null };

function texts(parts: readonly Part[]): string[] {
    return parts.filter((part): part is string => typeof part === "string");
}

function refs(parts: readonly Part[]): string[] {
    return parts.filter((part): part is Exclude<Part, string> => typeof part !== "string").map((part) => `${part.kind}:${part.id}`);
}

function described(tool: string, op: string, args?: Record<string, unknown>): CardView {
    return describeAction(tool, args ?? argsOf(tool, op), null, none);
}

function joinedOf(view: CardView): string {
    return view.lines.map((line) => texts(line.parts).join("")).join("\n");
}

const said = copy.agent.describe;

describe("every write a tool offers has a describer of its own", () => {
    for (const [at, [tool, args]] of confirmable.entries()) {
        it(`${tool} ${typeof args["op"] === "string" ? args["op"] : ""} #${at} is described without JSON`, () => {
            const view = describeAction(tool, args, null, none);
            expect(view.generic).toBe(false);
            expect(view.title.length).toBeGreaterThan(0);
            const strings = [...texts(view.title), ...view.lines.flatMap((line) => texts(line.parts))];
            for (const held of strings) {
                expect(held).not.toMatch(/"\s*:|\{"|"\}|\[\{|\}\]/);
                expect(held).not.toMatch(/\b[a-z]+_[a-z_]+\b/);
            }
        });
    }
});

describe("the site cards", () => {
    it("falls back to this site when the call names none", () => {
        expect(texts(described("sites_manage", "delete").title).join("")).toBe(`${said.sites.delete} ${said.sites.thisSite}`);
    });

    it("words the observation settings of an update", () => {
        const joined = joinedOf(described("sites_manage", "update"));
        expect(joined).toContain(said.sites.renders(6));
        expect(joined).toContain(said.sites.shortcodes(said.sites.defaults));
        expect(joined).toContain(said.sites.builders("Elementor"));
    });

    it("says what a sync walks", () => {
        const view = described("sites_manage", "sync");
        expect(joinedOf(view)).toContain(said.sites.syncFull);
        expect(joinedOf(view)).toContain(said.sites.syncRescan);
    });
});

describe("the graph cards", () => {
    it("connects two entities as a sentence with both ends resolvable", () => {
        const view = described("graph_edit", "add_edge");
        expect(refs(view.title)).toStrictEqual(["entity:e2", "entity:e1"]);
        expect(texts(view.title).join("")).toBe("Connect  under ");
        expect(joinedOf(view)).toContain("Travel mugs are a kind of mug.");
    });

    it("names the entity it creates, its parent, and counts its keywords and anchors", () => {
        const view = described("graph_edit", "create");
        expect(texts(view.title).join("")).toBe('Create entity "Travel Mugs"');
        expect(view.lines.flatMap((line) => refs(line.parts))).toContain("entity:e1");
        const joined = joinedOf(view);
        expect(joined).toContain("Keywords travel mug (2,400), insulated mug, mug with lid");
        expect(joined).toContain("travel mugs");
    });

    it("creates a whole tree in one write", () => {
        const view = describeAction("graph_edit", confirmable[6]?.[1] ?? {}, null, none);
        expect(texts(view.title).join("")).toBe(said.graph.createMany(3));
        expect(joinedOf(view)).toContain(said.graph.child("Vitamin D", "Vitamins"));
    });

    it("moves an entity under another and keeps both when asked", () => {
        const view = described("graph_edit", "move");
        expect(refs(view.title)).toStrictEqual(["entity:e2", "entity:e3"]);
        expect(joinedOf(view)).toContain(said.graph.keepBoth);
    });

    it("gives entities their page tree's parents under a page", () => {
        const view = described("graph_edit", "adopt_parents");
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["page:p9", "entity:e2", "entity:e3"]);
        expect(joinedOf(view)).toContain(said.graph.adoptBody);
    });

    it("reads approving cards as accepting them, each with its entity and what it changes", () => {
        const view = described("graph_propose", "cards");
        expect(texts(view.title).join("")).toBe(said.graph.proposeCards(2));
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["entity:e1", "entity:e2"]);
        const written = view.lines.map((line) => texts(line.parts).join(""));
        expect(written).toContain(said.graph.proposeCardsBody);
        expect(written).toContain(said.graph.definition("Mugs made of fired clay."));
        expect(written).toContain(said.graph.fact("Material", "Stoneware"));
        expect(written).toContain(said.graph.proposeCardsReason("The category page says so."));
        expect(written).toContain(said.graph.factsCount(0));
    });

    it("lists the card an entity update carries", () => {
        const joined = joinedOf(described("graph_edit", "update"));
        expect(joined).toContain("Definition: Mugs thrown and fired from stoneware clay.");
        expect(joined).toContain("Capacity: 350 ml");
        expect(joined).toContain('Note "Tone"');
    });
});

describe("the page cards", () => {
    it("names the chosen pages of a bulk type change in words", () => {
        const view = describeAction("pages_manage", confirmable.find(([tool, args]) => tool === "pages_manage" && args["op"] === "set_types")?.[1] ?? {}, null, none);
        expect(refs(view.title)).toStrictEqual(["pagetype:pt-hub"]);
        const lines = view.lines.map((line) => texts(line.parts).join(""));
        expect(lines.some((held) => held.startsWith(said.selection.under))).toBe(true);
        expect(lines.some((held) => held.startsWith(said.selection.ofType))).toBe(true);
        expect(lines.some((held) => held.startsWith(said.selection.except))).toBe(true);
    });

    it("hands the pages back to the rules on a reset", () => {
        const view = describeAction("pages_manage", { op: "set_types", select: { ofType: "pt-guide" }, reset: true }, null, none);
        expect(texts(view.title).join("")).toBe(said.pages.resetTypes);
        expect(texts(view.lines[0]?.parts ?? []).join("")).toBe(`${said.selection.everyOfType} `);
    });

    it("types a page through an update", () => {
        expect(described("pages_manage", "update").lines.flatMap((line) => refs(line.parts))).toContain("pagetype:pt-hub");
    });

    it("warns that a delete reaches the WordPress trash when asked", () => {
        const view = described("pages_manage", "delete");
        expect(view.lines.map((line) => [line.tone, texts(line.parts).join("")])).toStrictEqual([["danger", said.pages.deleteOnSite]]);
    });
});

describe("the run cards", () => {
    it("lists the pages, the steps, the publish mode, the budget, the brief and the directions", () => {
        const view = described("runs_control", "start");
        expect(texts(view.title).join("")).toBe("Start a generate run over 3 pages");
        expect(view.lines.flatMap((line) => refs(line.parts))).toEqual(
            expect.arrayContaining(["page:p1", "page:p2", "page:p3", "template:t1"]),
        );
        const joined = joinedOf(view);
        expect(joined).toContain(`${stepLabel("generate_body")}, ${stepLabel("publish")}`);
        expect(joined).toContain("$12.50");
        expect(joined).toContain(said.runs.brief.goal("Sell the travel mugs"));
        expect(joined).toContain(said.runs.brief.reference(17));
        expect(joined).toContain(said.runs.directives(1));
        expect(joined).toContain(said.runs.addEdges);
        expect(view.lines.some((line) => line.tone === "warn" && texts(line.parts).join("").includes("Publishes"))).toBe(true);
    });

    it("prices a run over a selection", () => {
        const view = described("runs_control", "estimate");
        expect(texts(view.title).join("")).toBe(said.runs.estimate(said.runs.startSelection("relink")));
        expect(joinedOf(view)).toContain(said.runs.fields("SEO title"));
    });

    it("names the run it puts back and warns of what it takes off the site", () => {
        const view = described("runs_control", "revert");
        expect(texts(view.title).join("")).toBe(`${said.runs.revert} `);
        expect(refs(view.title)).toStrictEqual(["run:r1"]);
        expect(view.lines.map((line) => [line.tone, texts(line.parts).join("")])).toStrictEqual([["warn", said.runs.revertBody]]);
    });
});

describe("the template cards", () => {
    it("reads a created template's spec as sentences", () => {
        const view = described("templates", "create");
        expect(texts(view.title).join("")).toBe('Create template "Product page"');
        const joined = joinedOf(view);
        expect(joined).toContain(said.templates.scopeSite);
        expect(joined).toContain("Overview");
        expect(joined).toContain("600");
        expect(joined).not.toContain("targetWords");
    });

    it("describes an update as what changes against the current template", () => {
        const sent = argsOf("templates", "create")["spec"] as Record<string, unknown>;
        const stored = { ...sent, modelProfiles: { writer: { provider: "openai", model: "gpt-5" } } } as unknown as TemplateSpec;
        const changed = { ...sent, tone: "brisk", length: { min: 400, max: 800 } };
        const view = describeAction("templates", { op: "update", templateId: "t1", spec: changed }, null, {
            currentTemplate: stored,
            finding: null,
            summary: null,
        });
        const joined = joinedOf(view);
        expect(joined).toContain("brisk");
        expect(joined).toContain("400");
        expect(joined).not.toContain("Overview");
    });

    it("reads an override for one page, or for the site, as sentences", () => {
        const page = described("templates", "override");
        expect(refs(page.title)).toStrictEqual(["template:t1", "page:p1"]);
        expect(joinedOf(page)).toContain("playful");
        const site = describeAction("templates", { op: "override", templateId: "t1", spec: { tone: "dry" } }, null, none);
        expect(texts(site.title).join("")).toContain(said.templates.forSite);
    });
});

describe("the page type cards", () => {
    it("creates a type from a role, with its template", () => {
        const view = described("pagetypes", "save");
        expect(texts(view.title).join("")).toBe('Create page type "Category"');
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["template:t1"]);
        expect(joinedOf(view)).toContain(`Role ${copy.pageTypes.presets.hub}`);
    });

    it("renames a type and clears its template in words", () => {
        const view = describeAction("pagetypes", { op: "save", pageTypeId: "pt-hub", name: "Hubs", templateId: "" }, null, none);
        expect(refs(view.title)).toStrictEqual(["pagetype:pt-hub"]);
        expect(joinedOf(view)).toContain('Renamed to "Hubs"');
        expect(joinedOf(view)).toContain(said.pagetypes.noTemplate);
    });

    it("says where a deleted type's pages go", () => {
        const view = described("pagetypes", "delete");
        expect(refs(view.title)).toStrictEqual(["pagetype:pt-guide"]);
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["pagetype:pt-entity"]);
        expect(view.lines[0]?.tone).toBe("danger");
    });

    it("reads a scope and its link policy as sentences, never as JSON", () => {
        const view = described("pagetypes", "scope");
        expect(texts(view.title).join("")).toBe(`Set ${copy.pageTypes.fields.body} on `);
        expect(refs(view.title)).toStrictEqual(["pagetype:pt-hub"]);
        const joined = joinedOf(view);
        expect(joined).toContain("Open with what the category covers.");
        expect(joined).toContain("At most 6000 characters");
        expect(joined).toContain("At most 20 links");
        expect(joined).toContain(`Anchors: ${copy.pageTypes.anchors.rotate}`);
        expect(joined).not.toMatch(/[{}[\]]/);
    });

    it("names a custom field by its label", () => {
        const view = describeAction(
            "pagetypes",
            { op: "scope", pageTypeId: "pt-hub", metaKey: "dosage_note", label: "Dosage note", provider: "plugin" },
            null,
            none,
        );
        expect(texts(view.title).join("")).toBe("Set Dosage note on ");
        expect(joinedOf(view)).toContain(`Stored by ${copy.pageTypes.providers.plugin}`);
    });

    it("reads a classification rule's conditions as sentences", () => {
        const view = described("pagetypes_classify", "save_rule");
        expect(texts(view.title).join("")).toBe(said.pagetypes.rules.add("Guides under the journal"));
        const joined = joinedOf(view);
        expect(joined).toContain(said.pagetypes.rules.off);
        expect(joined).toContain(said.pagetypes.conditions.url("prefix", "/journal/"));
        expect(joined).toContain(said.pagetypes.conditions.level(1, 3));
        expect(joined).toContain(said.pagetypes.conditions.listsNone);
        expect(joined).toContain(said.pagetypes.conditions.notFrontPage);
        expect(view.lines.flatMap((line) => refs(line.parts))).toStrictEqual(["pagetype:pt-guide", "page:p9"]);
    });
});

describe("the findings cards", () => {
    it("settles many findings of one code with one option", () => {
        const view = described("findings", "resolve_many");
        expect(texts(view.title).join("")).toBe(said.findings.resolveMany(copy.pages.model.findingCodes.page_without_entity, 3));
    });
});

describe("the generic describer", () => {
    const schema = JSON.parse(
        JSON.stringify({
            type: "object",
            properties: {
                op: { type: "string" },
                count: { type: "integer", description: "How many" },
                enabled: { type: "boolean" },
                names: { type: "array", items: { type: "string" } },
                nested: { type: "object", properties: { deep: { type: "string" } } },
            },
        }),
    ) as ToolSchema;
    const args = { op: "do", count: 3, enabled: true, names: ["a", "b"], nested: { deep: "x" } };

    it("falls back to the schema for a tool nobody described, and leaves the op out", () => {
        const view = describeAction("mystery_do_thing", args, schema, none);
        expect(view.generic).toBe(true);
        const joined = joinedOf(view);
        expect(joined).toContain("How many: 3");
        expect(joined).toContain("enabled: yes");
        expect(joined).toContain("a, b");
        expect(joined).toContain("deep: x");
        expect(joined).not.toContain("op:");
        expect(joined).not.toMatch(/[{}[\]]/);
    });

    it("titles the card with the sentence the call said of itself", () => {
        const view = describeAction("mystery_do_thing", args, schema, { ...none, summary: "Do the thing to three pages" });
        expect(view.title).toStrictEqual(["Do the thing to three pages"]);
    });

    it("titles a known tool's unknown op with its words when the call said nothing", () => {
        const view = describeAction("pagetypes_classify", { op: "test_rule", ruleId: "r1" }, null, none);
        expect(view.title).toStrictEqual([copy.agent.acts["pagetypes_classify"]?.acts["test_rule"]]);
    });

    it("hides a masked secret rather than printing it", () => {
        const view = describeAction("sites_manage", { op: "rotate", password: "***" }, null, none);
        expect(joinedOf(view)).not.toContain("***");
    });
});
