import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { CatalogField, PageType } from "../../../data/types.js";
import { aPageType, aScope, hubLinks } from "../../../testing/pagetypes.js";
import { renderScreen } from "../../../testing/render.js";
import { aSite } from "../../../testing/sites.js";

const said = copy.pageTypes;

interface Calls {
    created: unknown[];
    saved: unknown[];
    deleted: unknown[];
    scoped: unknown[];
}

const calls: Calls = { created: [], saved: [], deleted: [], scoped: [] };

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

const category = aPageType("t-cat", "hub", { name: "Category", pages: 3, scopes: [aScope("body", { enabled: true, links: hubLinks() })] });

const guide = aPageType("t-guide", "entity", { name: "Guide", pages: 1 });

const store = aPageType("t-store", "store_category", {
    name: "Store category",
    pages: 7,
    scopes: [aScope("body", { links: hubLinks() })],
});

const types: PageType[] = [store, guide, category, hub];

vi.mock("../../../data/hooks/sites.js", () => ({
    useSites: () => ({ data: { pages: [{ items: [aSite({ id: "s1", name: "Peptide Lab" })], hasMore: false }] }, isPending: false }),
}));

vi.mock("../../../data/hooks/templates.js", () => ({
    usePageTypes: () => ({ data: { types, fields: catalog }, isPending: false, error: null, refetch: () => undefined }),
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

const { PageTypesScreen } = await import("./screen.js");

function rowOf(name: string): HTMLElement {
    const held = screen.getByRole("row", { name: said.screen.open(name) });
    return held;
}

function expand(field: string): void {
    fireEvent.click(screen.getByRole("button", { name: said.scope.expand(field) }));
}

describe("Settings, Page types", () => {
    it("lists the site's types by role with their pages and template", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        const names = screen.getAllByRole("row").slice(1).map((row) => within(row).getAllByRole("cell")[0]?.textContent);
        expect(names).toStrictEqual(["Category", "Hub", "Guide", "Store category"]);
        const cells = within(rowOf("Hub"))
            .getAllByRole("cell")
            .map((cell) => cell.textContent);
        expect(cells).toStrictEqual(["Hub", said.presets.hub, "12", "Hub page"]);
        expect(within(rowOf("Category")).getByText(said.screen.siteDefault)).toBeDefined();
        expect(within(rowOf("Store category")).getByText(said.screen.readOnly)).toBeDefined();
    });

    it("opens a type's fields and saves a scope's instruction through SetScope", () => {
        calls.scoped.length = 0;
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
        calls.scoped.length = 0;
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
        calls.scoped.length = 0;
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        expand(said.fields.body);
        fireEvent.click(screen.getByRole("switch", { name: said.links.childrenSection }));
        fireEvent.click(screen.getByRole("button", { name: said.scope.save }));

        expect(calls.scoped[0]).toMatchObject({ field: "body", links: { links: true, rules: { childrenSection: false } } });
    });

    it("keeps a field this role seldom uses behind a button", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-hub");

        expect(screen.queryByText(said.fields.product_short_description)).toBeNull();
        fireEvent.click(screen.getByRole("button", { name: said.drawer.others(1) }));
        expect(screen.getByText(said.fields.product_short_description)).toBeDefined();
    });

    it("shows a read-only role's fields read only, with nothing to save and no custom field", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-store");

        expect(screen.getByText(said.drawer.readOnly(said.presets.store_category))).toBeDefined();
        expand(said.fields.body);
        expect(screen.getByRole("switch", { name: said.scope.enabled })).toHaveProperty("disabled", true);
        expect(screen.queryByRole("button", { name: said.scope.save })).toBeNull();
        expect(screen.queryByRole("form", { name: said.custom.title })).toBeNull();
    });

    it("refuses a custom field with a blank label before anything is sent, then adds it", () => {
        calls.scoped.length = 0;
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

    it("deletes a type by moving its pages to another type of the same role", () => {
        calls.deleted.length = 0;
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

    it("never offers to delete the last type of a role", () => {
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1&type=t-guide");

        expect(screen.getByRole("button", { name: said.remove.action })).toHaveProperty("disabled", true);
    });

    it("creates a type from a role", () => {
        calls.created.length = 0;
        renderScreen(<PageTypesScreen />, "/settings/types?site=s1");

        fireEvent.click(screen.getByRole("button", { name: said.screen.create }));
        const dialog = screen.getByRole("dialog");
        fireEvent.change(within(dialog).getByRole("textbox", { name: new RegExp(`^${said.form.name}`) }), { target: { value: "Category" } });
        fireEvent.click(within(dialog).getByRole("button", { name: said.form.create }));
        expect(calls.created[0]).toStrictEqual({ siteId: "s1", name: "Category", preset: "hub", templateId: undefined });
    });
});
