import { fireEvent, screen, within } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { StepSpec } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import { writesBody } from "./start-rewrite.js";

const said = copy.runs.start;

interface Held {
    recipe: StepSpec[] | null;
    priced: unknown[];
    started: unknown[];
}

const held: Held = { recipe: null, priced: [], started: [] };

const answered: { estimate: { tokens: number; usd: number; findings: never[] }; added: never[]; runId: string; fields: string[] } = {
    estimate: { tokens: 1200, usd: 0.4, findings: [] },
    added: [],
    runId: "r1",
    fields: [],
};

function mutation(into: unknown[]) {
    return {
        mutate: (request: unknown, options?: { onSuccess?: (answer: typeof answered) => void }) => {
            into.push(request);
            options?.onSuccess?.(answered);
        },
        reset: () => undefined,
        isPending: false,
        error: null,
    };
}

const paths: Readonly<Record<string, string>> = { "p-hub": "/peptides/", "p-guide": "/lab-guides/" };

vi.mock("../../data/hooks/runs.js", () => ({
    useEstimateRun: () => mutation(held.priced),
    useStartRun: () => mutation(held.started),
}));

vi.mock("../../data/hooks/templates.js", () => ({
    useTemplates: () => ({ data: undefined, isPending: false }),
    useResolvedTemplate: (pageId: string | null) => ({
        data: pageId === null ? undefined : { templateId: "t1", siteId: "s1", version: 1, spec: { recipe: held.recipe } },
        isPending: false,
    }),
}));

vi.mock("../../data/hooks/pages.js", () => ({
    usePagePaths: (ids: readonly string[]) => new Map(ids.flatMap((id) => (paths[id] === undefined ? [] : [[id, paths[id]]]))),
}));

vi.mock("./start-tree.js", () => ({
    TargetTree: () => null,
}));

const { StartRunDrawer } = await import("./start.js");

function drawer(kind = "generate") {
    return renderScreen(
        <StartRunDrawer
            open={true}
            onOpenChange={() => undefined}
            siteId="s1"
            preselect={["p-hub", "p-guide"]}
            initialKind={kind}
            onStarted={() => undefined}
        />,
    );
}

function price(): void {
    fireEvent.click(screen.getByRole("button", { name: said.estimate }));
}

function lastPriced(): Record<string, unknown> {
    return (held.priced.at(-1) ?? {}) as Record<string, unknown>;
}

function switchOn(): void {
    fireEvent.click(screen.getByRole("switch", { name: said.rewrite }));
}

beforeAll(() => {
    Element.prototype.hasPointerCapture = () => false;
    Element.prototype.releasePointerCapture = () => undefined;
    Element.prototype.scrollIntoView = () => undefined;
});

beforeEach(() => {
    held.recipe = [
        { name: "resolve_context", enabled: true },
        { name: "generate_body", enabled: true },
        { name: "publish", enabled: true },
    ];
    held.priced = [];
    held.started = [];
    answered.fields = [];
});

describe("a run scoped by the type of its pages", () => {
    it("says in one line which fields it writes when their type writes no body", () => {
        answered.fields = ["seo_title", "seo_description"];
        drawer();
        expect(document.querySelector("[data-run-type-scope]")).toBeNull();

        price();

        expect(screen.getByText(said.typeScope([copy.pageTypes.fields.seo_title, copy.pageTypes.fields.seo_description]))).toBeDefined();
    });

    it("says nothing of the kind for a whole run", () => {
        drawer();
        price();

        expect(document.querySelector("[data-run-type-scope]")).toBeNull();
    });
});

describe("the recipe that writes a body", () => {
    it.each([
        ["a recipe that generates the body", [{ name: "generate_body", enabled: true }], true],
        ["a recipe whose body step is off", [{ name: "generate_body", enabled: false }], false],
        ["a recipe with no body step", [{ name: "insert_links", enabled: true }], false],
        ["no recipe, which runs the generate recipe", null, true],
        ["an empty recipe, which runs the generate recipe", [], true],
    ] as const)("reads %s", (_name, recipe, want) => {
        expect(writesBody(recipe)).toBe(want);
    });
});

describe("rewriting the owner's own text from the start dialog", () => {
    it("is off by default, so the run keeps the client's text", () => {
        drawer();

        const toggle = screen.getByRole("switch", { name: said.rewrite });
        expect((toggle as HTMLInputElement).checked).toBe(false);
        expect(screen.getByText(said.rewriteHint)).toBeDefined();

        price();
        expect(lastPriced()).not.toHaveProperty("rewrite");
    });

    it("asks first, naming every chosen page and saying its own paragraphs will be replaced", () => {
        drawer();
        switchOn();

        const asked = within(screen.getByRole("dialog", { name: said.rewriteTitle }));
        expect(asked.getByText(said.rewriteBody)).toBeDefined();
        expect((asked.getByRole("checkbox", { name: "/peptides/" }) as HTMLInputElement).checked).toBe(true);
        expect((asked.getByRole("checkbox", { name: "/lab-guides/" }) as HTMLInputElement).checked).toBe(true);
        expect(asked.getByRole("button", { name: said.rewriteConfirm(2) })).toBeDefined();
    });

    it("sends the pages a person confirmed, and only those", () => {
        drawer();
        switchOn();

        const asked = within(screen.getByRole("dialog", { name: said.rewriteTitle }));
        fireEvent.click(asked.getByRole("checkbox", { name: "/peptides/" }));
        fireEvent.click(asked.getByRole("button", { name: said.rewriteConfirm(1) }));

        expect(screen.queryByRole("dialog", { name: said.rewriteTitle })).toBeNull();
        expect(screen.getByText(said.rewriteOn(["/lab-guides/"]))).toBeDefined();

        price();
        expect(lastPriced()["rewrite"]).toEqual(["p-guide"]);

        fireEvent.click(screen.getByRole("button", { name: said.confirm }));
        expect((held.started.at(-1) as Record<string, unknown>)["rewrite"]).toEqual(["p-guide"]);
    });

    it("stays off when the person keeps their text", () => {
        drawer();
        switchOn();

        fireEvent.click(screen.getByRole("button", { name: said.rewriteCancel }));

        expect((screen.getByRole("switch", { name: said.rewrite }) as HTMLInputElement).checked).toBe(false);
        price();
        expect(lastPriced()).not.toHaveProperty("rewrite");
    });

    it("turns off again without asking", () => {
        drawer();
        switchOn();
        fireEvent.click(screen.getByRole("button", { name: said.rewriteConfirm(2) }));

        switchOn();

        expect(screen.queryByRole("dialog", { name: said.rewriteTitle })).toBeNull();
        price();
        expect(lastPriced()).not.toHaveProperty("rewrite");
    });

    it("is not offered when the recipe writes no body", () => {
        held.recipe = [
            { name: "generate_body", enabled: false },
            { name: "insert_links", enabled: true },
        ];
        drawer();

        expect(screen.queryByRole("switch", { name: said.rewrite })).toBeNull();
    });

    it("is not offered to a kind with its own recipe", () => {
        drawer("relink");

        expect(screen.queryByRole("switch", { name: said.rewrite })).toBeNull();
    });
});
