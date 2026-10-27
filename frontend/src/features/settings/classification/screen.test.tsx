import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { ClassificationPreview, Rule, RuleTest } from "../../../data/types.js";
import { aNode } from "../../map/model/fixture.js";
import { aPageType } from "../../../testing/pagetypes.js";
import { renderScreen } from "../../../testing/render.js";
import { aSite } from "../../../testing/sites.js";

const said = copy.classification;

interface Calls {
    saved: unknown[];
    deleted: unknown[];
    reordered: unknown[];
    tested: unknown[];
    applied: unknown[];
    refetched: number;
}

const calls: Calls = { saved: [], deleted: [], reordered: [], tested: [], applied: [], refetched: 0 };

const answers: { saveError: Error | null; applyError: Error | null } = { saveError: null, applyError: null };

function aRule(id: string, number: number, name: string, typeId: string, conditions: Rule["conditions"], enabled = true): Rule {
    return {
        id,
        siteId: "s1",
        kind: "rule",
        number,
        name,
        enabled,
        typeId,
        conditions,
        createdAt: "2026-10-26T09:00:00Z",
        updatedAt: "2026-10-26T09:00:00Z",
    };
}

const rules: Rule[] = [
    aRule("r1", 1, "Front page", "t-landing", { frontPage: true }),
    aRule("r2", 2, "Guides by address", "t-guide", { urls: [{ mode: "segment" as never, value: "guides" }] }),
    aRule("r3", 3, "Under the lab", "t-guide", { ancestor: { pageId: "p-lab" } }, false),
];

const fallback: Rule = { ...aRule("rf", 4, "Anything no rule matched", "t-info", {}), kind: "fallback" };

const types = [
    aPageType("t-landing", "landing", { name: "Landing" }),
    aPageType("t-cat", "category", { name: "Category" }),
    aPageType("t-guide", "guide", { name: "Guide" }),
    aPageType("t-info", "info", { name: "Info" }),
];

const tested: RuleTest = {
    verdict: "met",
    conditions: [{ condition: "url" as never, verdict: "met" as never }],
    facts: {
        pageId: "p-recon",
        path: "/lab-guides/reconstitution/",
        slug: "reconstitution",
        postType: "page",
        technical: "",
        frontPage: false,
        rendered: true,
        listing: false,
        products: 0,
        schema: ["HowTo"],
        words: 1240,
        h2: 6,
        level: 2,
        ancestors: ["p-lab"],
    },
    outcome: { typeId: "t-guide", ruleId: "r2", reason: "rule", waiting: false, kept: false },
};

const preview: ClassificationPreview = {
    siteId: "s1",
    hash: "h1",
    groups: [
        { from: "", to: "t-landing", pages: 1, retyped: true, rules: [{ ruleId: "r1", pages: 1 }], dependsOn: [], samples: ["/"] },
        {
            from: "t-cat",
            to: "t-guide",
            pages: 90,
            retyped: true,
            rules: [
                { ruleId: "r2", pages: 60 },
                { ruleId: "", pages: 30 },
            ],
            dependsOn: [{ from: "", to: "t-landing" }],
            samples: ["/lab-guides/reconstitution/"],
        },
    ],
    moving: 91,
    refreshed: 0,
    kept: 4,
    waiting: 2,
    unchanged: 100,
    byDefault: 3,
};

function writer(record: unknown[], errorOf: () => Error | null = () => null) {
    return () => ({
        mutate: (request: unknown, options?: { onSuccess?: (answered: unknown) => void; onError?: (thrown: unknown) => void }) => {
            record.push(request);
            const failed = errorOf();
            if (failed === null) {
                options?.onSuccess?.({ moved: 91, changed: 1 });
            } else {
                options?.onError?.(failed);
            }
        },
        reset: () => undefined,
        isPending: false,
        error: errorOf(),
    });
}

vi.mock("../../../data/hooks/sites.js", () => ({
    useSites: () => ({ data: { pages: [{ items: [aSite({ id: "s1", name: "Peptide Lab" })], hasMore: false }] }, isPending: false }),
}));

vi.mock("../../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({ data: { types, fields: [] }, isPending: false, error: null, refetch: () => undefined }),
}));

vi.mock("../../../data/hooks/pages.js", () => ({
    usePagePaths: (ids: readonly string[]) => new Map(ids.filter((id) => id === "p-lab").map((id) => [id, "/lab-guides/"])),
    useSiteMap: (filter: { search?: string }) => ({
        data:
            (filter.search ?? "") === ""
                ? undefined
                : { pages: [{ nodes: { items: [aNode("p-recon", "/lab-guides/reconstitution/")], hasMore: false } }] },
        isPending: false,
        hasNextPage: false,
    }),
}));

vi.mock("../../../data/hooks/classification.js", () => ({
    useRules: () => ({ data: { rules, fallback }, isPending: false, error: null, refetch: () => undefined }),
    useSaveRule: writer(calls.saved, () => answers.saveError),
    useDeleteRule: writer(calls.deleted),
    useReorderRules: writer(calls.reordered),
    useRuleTest: (request: unknown) => {
        if (request !== null) {
            calls.tested.push(request);
        }
        return { data: request === null ? undefined : tested, error: null, isPending: false };
    },
    useClassificationPreview: (_siteId: string, enabled: boolean) => ({
        data: enabled ? preview : undefined,
        isPending: !enabled,
        refetch: () => {
            calls.refetched += 1;
            return Promise.resolve();
        },
    }),
    useClassificationPages: () => ({
        data: { pages: [{ items: [{ pageId: "p-recon", path: "/lab-guides/reconstitution/", from: "t-cat", to: "t-guide", ruleId: "r2", reason: "rule", was: "", wasRuleId: "" }], hasMore: false }] },
        isPending: false,
        hasNextPage: false,
    }),
    useApplyClassification: writer(calls.applied, () => answers.applyError),
}));

const { ClassificationScreen } = await import("./screen.js");

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
});

beforeEach(() => {
    calls.saved.length = 0;
    calls.deleted.length = 0;
    calls.reordered.length = 0;
    calls.tested.length = 0;
    calls.applied.length = 0;
    calls.refetched = 0;
    answers.saveError = null;
    answers.applyError = null;
});

function open(): void {
    renderScreen(<ClassificationScreen />, "/settings/classification?site=s1");
}

function rowOf(id: string): HTMLElement {
    const held = document.querySelector<HTMLElement>(`[data-rule-row="${id}"]`);
    if (held === null) {
        throw new Error(`no rule row ${id}`);
    }
    return held;
}

function pick(name: string, option: string): void {
    fireEvent.pointerDown(screen.getByRole("combobox", { name }), { button: 0, ctrlKey: false, pointerType: "mouse" });
    fireEvent.click(screen.getByRole("option", { name: option }));
}

function refused(message: string, field: string): Error {
    return new Error(message, { cause: { code: "INVALID", message, details: { field } } });
}

describe("Settings, Classification", () => {
    it("lists the rules in order with their number, name, type and conditions in words, then the type for anything else", () => {
        open();

        expect([...document.querySelectorAll("[data-rule-row]")].map((row) => row.getAttribute("data-rule-row"))).toStrictEqual(["r1", "r2", "r3"]);
        expect(within(rowOf("r1")).getByText("Pages that are the front page.")).toBeDefined();
        expect(within(rowOf("r1")).getByText("Landing")).toBeDefined();
        expect(within(rowOf("r2")).getByText("Pages that have an address that has the word guides.")).toBeDefined();
        expect(within(rowOf("r3")).getByText("Pages that sit under /lab-guides/.")).toBeDefined();
        expect(within(rowOf("r3")).getByText(said.off)).toBeDefined();
        expect(screen.getByText(said.fallback.name)).toBeDefined();
        expect(screen.getByRole("combobox", { name: said.fallback.type }).textContent).toContain("Info");
    });

    it("turns a rule off with the rule as it stands", () => {
        open();

        fireEvent.click(screen.getByRole("switch", { name: said.enable("Guides by address") }));

        expect(calls.saved).toStrictEqual([
            { id: "r2", siteId: "s1", name: "Guides by address", enabled: false, typeId: "t-guide", conditions: rules[1]?.conditions },
        ]);
    });

    it("moves a rule down, and never the first rule up", () => {
        open();

        expect(screen.getByRole("button", { name: said.moveUp("Front page") })).toHaveProperty("disabled", true);
        fireEvent.click(screen.getByRole("button", { name: said.moveDown("Front page") }));

        expect(calls.reordered).toStrictEqual([{ siteId: "s1", ruleIds: ["r2", "r1", "r3"] }]);
    });

    it("sets the type for anything no rule matched", () => {
        open();

        pick(said.fallback.type, "Guide");

        expect(calls.saved).toStrictEqual([{ id: "rf", siteId: "s1", name: "Anything no rule matched", enabled: true, typeId: "t-guide", conditions: {} }]);
    });

    it("edits a rule's address and saves it", () => {
        open();
        fireEvent.click(screen.getByRole("button", { name: said.open("Guides by address") }));

        const drawer = within(screen.getByRole("dialog"));
        fireEvent.change(drawer.getByRole("textbox", { name: `${said.drawer.urlValue} 1` }), { target: { value: " lab " } });
        fireEvent.click(drawer.getByRole("button", { name: said.drawer.save }));

        expect(calls.saved).toStrictEqual([
            { id: "r2", siteId: "s1", name: "Guides by address", enabled: true, typeId: "t-guide", conditions: { urls: [{ mode: "segment", value: "lab" }] } },
        ]);
    });

    it("shows Go's refusal beside the address it names", () => {
        answers.saveError = refused("A segment is one word of the address.", "conditions.urls[0].value");
        open();
        fireEvent.click(screen.getByRole("button", { name: said.open("Guides by address") }));

        const row = document.querySelector<HTMLElement>('[data-url-row="0"]');
        if (row === null) {
            throw new Error("no address row");
        }
        expect(within(row).getByText("A segment is one word of the address.")).toBeDefined();
        expect(within(row).getByRole("textbox").getAttribute("aria-invalid")).toBe("true");
    });

    it("asks for a name, a type and a condition before it creates a rule", () => {
        open();
        fireEvent.click(screen.getByRole("button", { name: said.add }));
        const drawer = within(screen.getByRole("dialog"));
        const create = (): void => {
            fireEvent.click(drawer.getByRole("button", { name: said.drawer.create }));
        };

        create();
        expect(drawer.getByText(said.drawer.nameMissing)).toBeDefined();
        fireEvent.change(drawer.getByRole("textbox", { name: new RegExp(`^${said.drawer.name}`) }), { target: { value: "Shop pages" } });
        create();
        expect(drawer.getAllByText(said.drawer.noType).length).toBeGreaterThan(0);
        pick(said.drawer.type, "Category");
        create();
        expect(drawer.getByText(said.drawer.empty)).toBeDefined();
        fireEvent.click(drawer.getByRole("button", { name: said.drawer.addUrl }));
        fireEvent.change(drawer.getByRole("textbox", { name: `${said.drawer.urlValue} 1` }), { target: { value: "/shop/" } });
        create();

        expect(calls.saved).toStrictEqual([
            { siteId: "s1", name: "Shop pages", enabled: true, typeId: "t-cat", conditions: { urls: [{ mode: "prefix", value: "/shop/" }] } },
        ]);
    });

    it("deletes a rule once the person confirms", () => {
        open();
        fireEvent.click(screen.getByRole("button", { name: said.open("Guides by address") }));
        fireEvent.click(screen.getByRole("button", { name: said.drawer.remove }));

        expect(calls.deleted).toHaveLength(0);
        fireEvent.click(screen.getByRole("button", { name: said.drawer.removeConfirm }));
        expect(calls.deleted).toStrictEqual([{ id: "r2" }]);
    });

    it("tests the rule being edited on a page, condition by condition, with the page's facts and what the rules give it", async () => {
        open();
        fireEvent.click(screen.getByRole("button", { name: said.open("Guides by address") }));
        const tester = within(screen.getByRole("region", { name: said.tester.title }));

        fireEvent.change(tester.getByRole("searchbox"), { target: { value: "recon" } });
        await waitFor(() => {
            expect(tester.getByRole("option")).toBeDefined();
        });
        fireEvent.click(tester.getByRole("option"));

        expect(tester.getByText(said.tester.verdict.met("/lab-guides/reconstitution/"))).toBeDefined();
        expect(within(document.querySelector<HTMLElement>('[data-condition-verdict="url"]') ?? document.body).getByText(said.verdicts.met)).toBeDefined();
        expect(tester.getByText("/lab-guides/")).toBeDefined();
        expect(tester.getByText("How-to")).toBeDefined();
        expect(tester.getByText(said.tester.outcomeRule("Guide", said.ruleName(2, "Guides by address")))).toBeDefined();
        expect(calls.tested.at(-1)).toMatchObject({ siteId: "s1", pageId: "p-recon", rule: { id: "r2", typeId: "t-guide" } });
    });
});

describe("the classification preview", () => {
    function preview() {
        open();
        fireEvent.click(screen.getByRole("button", { name: said.preview }));
        return within(screen.getByRole("dialog"));
    }

    it("lists what would change by group, with each rule's count and what a group needs first", () => {
        const shown = preview();

        expect(shown.getByText(said.previewDialog.moving(91))).toBeDefined();
        expect(shown.getByText(said.previewDialog.kept(4))).toBeDefined();
        expect(shown.getByText(said.previewDialog.group("Category", "Guide"))).toBeDefined();
        expect(shown.getByText(`${said.ruleName(2, "Guides by address")}: 60`)).toBeDefined();
        expect(shown.getByText(`${said.previewDialog.noRule}: 30`)).toBeDefined();
        expect(shown.getByText(said.previewDialog.needs([said.previewDialog.group(said.previewDialog.untyped, "Landing")]))).toBeDefined();
        expect(shown.getByRole("link", { name: said.previewDialog.byDefaultLink }).getAttribute("href")).toBe("/s/s1/findings?code=type_by_default");

        fireEvent.click(shown.getByRole("button", { name: said.previewDialog.expand(said.previewDialog.group("Category", "Guide")) }));
        expect(shown.getByText("/lab-guides/reconstitution/")).toBeDefined();
    });

    it("ticks the group a chosen change depends on and applies the chosen ones after a confirmation", () => {
        const shown = preview();

        fireEvent.click(shown.getByRole("checkbox", { name: said.previewDialog.choose(said.previewDialog.group("Category", "Guide")) }));
        expect(shown.getByRole<HTMLInputElement>("checkbox", { name: said.previewDialog.choose(said.previewDialog.group(said.previewDialog.untyped, "Landing")) }).checked).toBe(true);
        fireEvent.click(shown.getByRole("button", { name: said.previewDialog.applyChosen(2, 91) }));

        expect(calls.applied).toHaveLength(0);
        fireEvent.click(screen.getByRole("button", { name: said.previewDialog.confirm }));
        expect(calls.applied).toHaveLength(1);
        const sent = calls.applied[0] as { siteId: string; hash: string; groups: { from: string; to: string }[] };
        expect(sent.siteId).toBe("s1");
        expect(sent.hash).toBe("h1");
        expect([...sent.groups].sort((a, b) => a.from.localeCompare(b.from))).toStrictEqual([
            { from: "", to: "t-landing" },
            { from: "t-cat", to: "t-guide" },
        ]);
    });

    it("applies everything when nothing is ticked, and works the preview out again when the site moved", () => {
        answers.applyError = new Error("moved", { cause: { code: "CONFLICT", message: "moved", details: { reason: "preview_moved" } } });
        const shown = preview();

        fireEvent.click(shown.getByRole("button", { name: said.previewDialog.applyAll(91) }));
        fireEvent.click(screen.getByRole("button", { name: said.previewDialog.confirm }));

        expect(calls.applied).toStrictEqual([{ siteId: "s1", hash: "h1", groups: [] }]);
        expect(screen.getByText(said.previewDialog.moved)).toBeDefined();
        expect(calls.refetched).toBe(1);
    });
});
