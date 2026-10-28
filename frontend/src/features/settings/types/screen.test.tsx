import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import { readToasts } from "../../../data/toasts.js";
import type { CatalogField, ClassificationPreview, PageType, Rule } from "../../../data/types.js";
import { creatablePageTypePresets } from "../../../generated/vocab.js";
import { aPageType, aScope, hubLinks } from "../../../testing/pagetypes.js";
import { renderScreen } from "../../../testing/render.js";
import { aSite } from "../../../testing/sites.js";
import { forgetPages, pagesMemoryOf } from "../../pages/views/memory.js";

const said = copy.pageTypes;

interface Calls {
    created: unknown[];
    saved: unknown[];
    deleted: unknown[];
    scoped: unknown[];
    followed: unknown[];
}

const calls: Calls = { created: [], saved: [], deleted: [], scoped: [], followed: [] };

function mutation(record: unknown[]) {
    return () => ({
        mutate: (request: unknown) => {
            record.push(request);
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    });
}

const hub = aPageType("t-hub", "hub", {
    name: "Hub",
    pages: 12,
    overrides: 2,
    templateId: "tpl-hub",
    scopes: [
        aScope("title"),
        aScope("body", { enabled: true, links: hubLinks(), stored: true }),
        aScope("seo_title", { enabled: true }),
        aScope("seo_canonical", { enabled: true, aiWrites: true }),
        aScope("product_short_description"),
    ],
});

function aField(key: string, aiWritable: boolean): CatalogField {
    return { key, read: "core", write: "core", text: true, instanced: false, denied: false, scopable: true, aiWritable };
}

const catalog: CatalogField[] = [aField("title", true), aField("body", true), aField("seo_title", true), aField("seo_canonical", false)];

const category = aPageType("t-cat", "category", {
    name: "Category",
    pages: 3,
    suggestedTemplateId: "tpl-hub",
    scopes: [aScope("body", { enabled: true, links: hubLinks() })],
});

const guide = aPageType("t-guide", "guide", {
    name: "Guide",
    pages: 1,
    scopes: [aScope("seo_title", { enabled: true }), aScope("seo_description", { enabled: true })],
});

const store = aPageType("t-store", "store_category", {
    name: "Store category",
    pages: 7,
    scopes: [aScope("body", { links: hubLinks() })],
});

function aRule(id: string, number: number, name: string, typeId: string, conditions: Rule["conditions"], pages: number, enabled = true): Rule {
    return {
        id,
        siteId: "s1",
        kind: "rule",
        number,
        name,
        enabled,
        typeId,
        conditions,
        pages,
        createdAt: "2026-10-26T09:00:00Z",
        updatedAt: "2026-10-26T09:00:00Z",
    };
}

const rules: Rule[] = [
    aRule("r1", 1, "Category pages", "t-cat", { urls: [{ mode: "prefix" as never, value: "/category/" }] }, 3),
    aRule("r2", 2, "Old guides", "t-guide", { slugs: ["old"] }, 0, false),
];

const fallback: Rule = { ...aRule("rf", 3, "Anything no rule matched", "t-guide", {}, 1), kind: "fallback" };

const preview: ClassificationPreview = {
    siteId: "s1",
    hash: "h1",
    groups: [],
    moving: 0,
    refreshed: 0,
    kept: 0,
    waiting: 0,
    unchanged: 23,
    byDefault: 0,
};

const answers: { types: PageType[]; beforeRules: number } = { types: [], beforeRules: 0 };

vi.mock("../../../data/hooks/sites.js", () => ({
    useSites: () => ({ data: { pages: [{ items: [aSite({ id: "s1", name: "Peptide Lab" })], hasMore: false }] }, isPending: false }),
}));

vi.mock("../../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({ data: { types: answers.types, fields: catalog }, isPending: false, error: null, refetch: () => undefined }),
    useTemplates: (filter: { scope?: string }) => ({
        data:
            filter.scope === "global"
                ? { pages: [{ items: [{ id: "tpl-hub", name: "Hub page" }], hasMore: false }] }
                : { pages: [{ items: [], hasMore: false }] },
    }),
    useCreatePageType: mutation(calls.created),
    useSavePageType: mutation(calls.saved),
    useDeletePageType: mutation(calls.deleted),
    useSetScope: mutation(calls.scoped),
}));

vi.mock("../../../data/hooks/pages.js", () => ({
    useSetPageTemplates: () => ({
        mutate: (request: unknown, options?: { onSuccess?: (answered: { changed: number }) => void }) => {
            calls.followed.push(request);
            options?.onSuccess?.({ changed: 2 });
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    }),
    usePagePaths: () => new Map<string, string>(),
    useSiteMap: () => ({ data: undefined, isPending: false, hasNextPage: false }),
}));

vi.mock("../../../data/hooks/classification.js", () => ({
    useRules: () => ({
        data: { rules, fallback, beforeRules: answers.beforeRules },
        isPending: false,
        error: null,
        refetch: () => undefined,
    }),
    useSaveRule: mutation([]),
    useDeleteRule: mutation([]),
    useReorderRules: mutation([]),
    useRuleTest: () => ({ data: undefined, error: null, isPending: false }),
    useClassificationPreview: (_siteId: string, enabled: boolean) => ({
        data: enabled ? preview : undefined,
        isPending: !enabled,
        refetch: () => Promise.resolve(),
    }),
    useClassificationPages: () => ({ data: undefined, isPending: false, hasNextPage: false }),
    useApplyClassification: mutation([]),
}));

const { PageTypesScreen, RulesRedirect } = await import("./screen.js");

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
});

beforeEach(() => {
    answers.types = [store, guide, category, hub];
    answers.beforeRules = 0;
    calls.created.length = 0;
    calls.saved.length = 0;
    calls.deleted.length = 0;
    calls.scoped.length = 0;
    calls.followed.length = 0;
    forgetPages("s1");
});

function cardOf(name: string): HTMLElement {
    return screen.getByRole("article", { name });
}

function expand(field: string): void {
    fireEvent.click(screen.getByRole("button", { name: said.scope.expand(field) }));
}

describe("Settings, Page types, how a page gets its type", () => {
    it("walks through the three steps from a page to what is written on it, and says a type is not a graph entity", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        const steps = within(screen.getByRole("list", { name: said.screen.steps.label }));
        expect(steps.getAllByRole("listitem").map((step) => step.querySelector("[data-step-title]")?.textContent)).toStrictEqual([
            said.screen.steps.read.title,
            said.screen.steps.rules.title,
            said.screen.steps.type.title,
        ]);
        expect(screen.getByText(said.screen.byHand)).toBeDefined();
        expect(screen.getByText(said.screen.notGraph)).toBeDefined();
        expect(screen.getByRole("link", { name: said.screen.withoutEntity }).getAttribute("href")).toBe(
            "/s/s1/findings?code=page_without_entity",
        );
        expect(screen.queryByText("Role")).toBeNull();
    });

    it("says how many pages still carry a type from before 2.6 and previews what the rules would give them", async () => {
        const user = userEvent.setup();
        answers.beforeRules = 4;
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        expect(screen.getByText(said.screen.before(4))).toBeDefined();
        await user.click(screen.getByRole("button", { name: said.screen.beforeAction }));

        expect(screen.getByRole("dialog", { name: copy.classification.previewDialog.title })).toBeDefined();
    });

    it("says nothing about 2.6 when every page holds a type the rules gave", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        expect(screen.queryByRole("button", { name: said.screen.beforeAction })).toBeNull();
    });

    it("switches to the rules and back", async () => {
        const user = userEvent.setup();
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        await user.click(screen.getByRole("radio", { name: said.screen.views.rules }));
        expect(document.querySelectorAll("[data-rule-row]")).toHaveLength(2);
        expect(screen.queryByRole("article", { name: "Category" })).toBeNull();

        await user.click(screen.getByRole("radio", { name: said.screen.views.types }));
        expect(cardOf("Category")).toBeDefined();
    });

    it("lands an old Classification link on the rules", () => {
        render(
            <MemoryRouter initialEntries={["/settings/classification?site=s1"]}>
                <Routes>
                    <Route path="/settings/classification" element={<RulesRedirect />} />
                    <Route path="/settings/types" element={<PageTypesScreen />} />
                </Routes>
            </MemoryRouter>,
        );

        expect(screen.getByRole<HTMLButtonElement>("radio", { name: said.screen.views.rules }).getAttribute("aria-checked")).toBe("true");
        expect(document.querySelectorAll("[data-rule-row]")).toHaveLength(2);
    });
});

describe("Settings, Page types, a card per type", () => {
    it("shows each type's purpose, pages, the rules that give it, what Postulator writes and its template", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        const written = within(screen.getByRole("region", { name: said.screen.written }));
        expect(written.getAllByRole("article").map((card) => card.getAttribute("aria-label"))).toStrictEqual(["Category", "Hub", "Guide"]);

        const categoryCard = within(cardOf("Category"));
        expect(categoryCard.getByText(said.presetHints.category)).toBeDefined();
        expect(categoryCard.getByRole("link", { name: said.screen.showPages("Category") }).textContent).toBe(said.screen.pages(3));
        expect(categoryCard.getByText("Pages that have an address that starts with /category/.")).toBeDefined();
        expect(categoryCard.getByText("The body and its links.")).toBeDefined();
        expect(categoryCard.getByText(said.screen.siteDefault)).toBeDefined();

        const guideCard = within(cardOf("Guide"));
        expect(guideCard.getByText(said.screen.anythingElse)).toBeDefined();
        expect(guideCard.queryByText("Pages that have the slug old.")).toBeNull();
        expect(guideCard.getByText("The SEO title and description.")).toBeDefined();

        const hubCard = within(cardOf("Hub"));
        expect(hubCard.getByText(said.presetHints.hub)).toBeDefined();
        expect(hubCard.getByText(said.screen.noRule)).toBeDefined();
        expect(hubCard.getByText("The body and its links, the SEO title, the canonical address.")).toBeDefined();
        expect(hubCard.getByText("Hub page")).toBeDefined();
    });

    it("keeps the types Postulator never writes apart, without a writing or a template", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        const never = within(screen.getByRole("region", { name: said.screen.never }));
        expect(never.getByText(said.screen.neverBody)).toBeDefined();
        const storeCard = within(never.getByRole("article", { name: "Store category" }));
        expect(storeCard.getByText(said.presetHints.store_category)).toBeDefined();
        expect(storeCard.queryByText(said.screen.writes)).toBeNull();
        expect(storeCard.queryByText(said.screen.template)).toBeNull();
    });

    it("opens Pages by type on the type from a card's page count", async () => {
        const user = userEvent.setup();
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        const count = within(cardOf("Category")).getByRole("link", { name: said.screen.showPages("Category") });
        expect(count.getAttribute("href")).toBe("/s/s1/pages");
        await user.click(count);

        expect(pagesMemoryOf("s1").view).toBe("types");
        expect(pagesMemoryOf("s1").openType).toBe("t-cat");
    });

    it("opens a type's fields from anywhere on its card", async () => {
        const user = userEvent.setup();
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        await user.click(within(cardOf("Guide")).getByText("The SEO title and description."));

        expect(document.querySelector('[data-page-type-drawer="t-guide"]')).not.toBeNull();
        expect(within(screen.getByRole("dialog")).getByText(said.drawer.startsFrom)).toBeDefined();
    });

    it("offers a type without a template the built-in one, and sets it with one click", async () => {
        const user = userEvent.setup();
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        const offer = within(cardOf("Category"));
        expect(offer.getByText(said.screen.suggested("Hub page"))).toBeDefined();
        expect(within(cardOf("Hub")).queryByText(said.screen.useSuggested)).toBeNull();
        await user.click(offer.getByRole("button", { name: said.screen.useSuggested }));

        expect(calls.saved).toStrictEqual([{ id: "t-cat", templateId: "tpl-hub" }]);
        expect(screen.queryByRole("dialog")).toBeNull();
    });

    it("makes the pages that keep their own template follow the type, once the person confirms", async () => {
        const user = userEvent.setup();
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        const hubCard = within(cardOf("Hub"));
        expect(hubCard.getByText(said.screen.overrides(2))).toBeDefined();
        expect(within(cardOf("Guide")).queryByText(said.screen.follow)).toBeNull();
        await user.click(hubCard.getByRole("button", { name: said.screen.follow }));

        const dialog = within(screen.getByRole("dialog"));
        expect(dialog.getByText(said.follow.body(2, "Hub page"))).toBeDefined();
        expect(calls.followed).toHaveLength(0);
        await user.click(dialog.getByRole("button", { name: said.follow.confirm }));

        expect(calls.followed).toStrictEqual([{ selection: { siteId: "s1", filter: { pageTypeId: "t-hub" } }, templateId: null }]);
        expect(readToasts().map((toast) => toast.message)).toContain(said.follow.done(2));
    });

    it("says the first sync seeds the types when the site has none yet", () => {
        answers.types = [];
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        expect(screen.getByText(said.screen.empty)).toBeDefined();
        expect(screen.getByText(said.screen.emptyBody)).toBeDefined();
    });
});

describe("Settings, Page types, a new type", () => {
    it("starts from one of the kinds a new type may take, never Hub or Article", async () => {
        const user = userEvent.setup();
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        await user.click(screen.getByRole("button", { name: said.screen.create }));
        const dialog = within(screen.getByRole("dialog"));
        fireEvent.pointerDown(dialog.getByRole("combobox", { name: said.form.preset }), { button: 0, ctrlKey: false, pointerType: "mouse" });
        expect(screen.getAllByRole("option").map((option) => option.textContent)).toStrictEqual(
            creatablePageTypePresets.map((preset) => said.presets[preset]),
        );
        expect(screen.queryByRole("option", { name: said.presets.entity })).toBeNull();
        fireEvent.keyDown(screen.getAllByRole("option")[0] ?? document.body, { key: "Escape" });

        fireEvent.change(dialog.getByRole("textbox", { name: new RegExp(`^${said.form.name}`) }), { target: { value: "Category" } });
        fireEvent.click(dialog.getByRole("button", { name: said.form.create }));
        expect(calls.created[0]).toStrictEqual({ siteId: "s1", name: "Category", preset: "landing", templateId: undefined });
    });
});

describe("Settings, Page types, a type's fields", () => {
    it("opens a type's fields and saves a scope's instruction through SetScope", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        const drawer = screen.getByRole("dialog");
        expect(within(drawer).getByText(said.drawer.fields)).toBeDefined();
        expand(said.fields.body);
        fireEvent.change(within(drawer).getByRole("textbox", { name: said.scope.instruction }), {
            target: { value: "Open with what the hub covers." },
        });
        fireEvent.click(within(drawer).getByRole("button", { name: said.scope.save }));

        expect(calls.scoped).toHaveLength(1);
        expect(calls.scoped[0]).toMatchObject({
            pageTypeId: "t-hub",
            field: "body",
            enabled: true,
            instruction: "Open with what the hub covers.",
            links: hubLinks(),
        });
    });

    it("says that rewrites, off by default, keep the client's own paragraphs", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        expand(said.fields.body);
        expect(screen.getByRole("switch", { name: said.scope.rewritesLabel })).toHaveProperty("checked", false);
        expect(screen.getByText(said.scope.rewritesHint)).toBeDefined();
    });

    it("keeps the model off a field a person writes, says why, and saves it so", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        expand(said.fields.seo_canonical);
        const writes = screen.getByRole("switch", { name: said.scope.aiWrites });
        expect(writes).toHaveProperty("disabled", true);
        expect(writes).toHaveProperty("checked", false);
        expect(writes.getAttribute("title")).toBe(said.scope.personOnly);

        fireEvent.change(screen.getByRole("textbox", { name: said.scope.instruction }), {
            target: { value: "The canonical is set by hand." },
        });
        fireEvent.click(screen.getByRole("button", { name: said.scope.save }));
        expect(calls.scoped[0]).toMatchObject({ field: "seo_canonical", aiWrites: false });
    });

    it("leaves the model's switch to a person on a field the model may write", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        expand(said.fields.body);
        expect(screen.getByRole("switch", { name: said.scope.aiWrites })).toHaveProperty("disabled", false);
    });

    it("edits the link rules of the field that carries the links", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        expand(said.fields.body);
        fireEvent.click(screen.getByRole("switch", { name: said.links.childrenSection }));
        fireEvent.click(screen.getByRole("button", { name: said.scope.save }));

        expect(calls.scoped[0]).toMatchObject({ field: "body", links: { links: true, rules: { childrenSection: false } } });
    });

    it("keeps a field pages like these seldom use behind a button", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        expect(screen.queryByText(said.fields.product_short_description)).toBeNull();
        fireEvent.click(screen.getByRole("button", { name: said.drawer.others(1) }));
        expect(screen.getByText(said.fields.product_short_description)).toBeDefined();
    });

    it("shows a read-only type's fields read only, with nothing to save and no custom field", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-store");

        expect(screen.getByText(said.drawer.readOnly(said.presets.store_category))).toBeDefined();
        expand(said.fields.body);
        expect(screen.getByRole("switch", { name: said.scope.enabled })).toHaveProperty("disabled", true);
        expect(screen.queryByRole("button", { name: said.scope.save })).toBeNull();
        expect(screen.queryByRole("form", { name: said.custom.title })).toBeNull();
    });

    it("refuses a custom field with a blank label before anything is sent, then adds it", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        const form = within(screen.getByRole("form", { name: said.custom.title }));
        fireEvent.change(form.getByRole("textbox", { name: new RegExp(`^${said.custom.key}`) }), { target: { value: "dosage_note" } });
        fireEvent.click(form.getByRole("button", { name: said.custom.add }));
        expect(form.getByText(said.custom.labelMissing)).toBeDefined();
        expect(calls.scoped).toHaveLength(0);

        fireEvent.change(form.getByRole("textbox", { name: new RegExp(`^${said.custom.label}`) }), { target: { value: "Dosage note" } });
        fireEvent.click(form.getByRole("button", { name: said.custom.add }));
        expect(calls.scoped[0]).toMatchObject({ pageTypeId: "t-hub", field: "meta:dosage_note", label: "Dosage note", provider: "plugin", enabled: true });
    });

    it("deletes a type by moving its pages to another type of the same kind", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-cat");

        fireEvent.click(screen.getByRole("button", { name: said.remove.action }));
        const dialog = screen.getByText(said.remove.body(3)).closest<HTMLElement>("[role=dialog]");
        if (dialog === null) {
            throw new Error("the delete dialog is not open");
        }
        expect(within(dialog).getByText(said.remove.target)).toBeDefined();
        fireEvent.click(within(dialog).getByRole("button", { name: said.remove.confirm }));
        expect(calls.deleted[0]).toStrictEqual({ id: "t-cat", reassignTo: "t-hub" });
    });

    it("never offers to delete the last type of its kind", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-guide");

        expect(screen.getByRole("button", { name: said.remove.action })).toHaveProperty("disabled", true);
    });
});
