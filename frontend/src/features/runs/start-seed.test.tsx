import { fireEvent, screen } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import type { StepSpec } from "../../data/types.js";
import { renderScreen } from "../../testing/render.js";
import type { StartSeed } from "./start-seed.js";

const said = copy.runs.start;

const held: { recipe: StepSpec[] | null; priced: unknown[] } = { recipe: null, priced: [] };

const answered = { estimate: { tokens: 1200, usd: 0.4, findings: [] }, added: [], runId: "r1", fields: [] };

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

const paths: Readonly<Record<string, string>> = { "p-mugs": "/mugs/" };

vi.mock("../../data/hooks/runs.js", () => ({
    useEstimateRun: () => mutation(held.priced),
    useStartRun: () => mutation([]),
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

function seeded(seed: StartSeed) {
    return renderScreen(
        <StartRunDrawer open={true} onOpenChange={() => undefined} siteId="s1" seed={seed} onStarted={() => undefined} />,
    );
}

function priced(): Record<string, unknown> {
    fireEvent.click(screen.getByRole("button", { name: said.estimate }));
    return (held.priced.at(-1) ?? {}) as Record<string, unknown>;
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
});

describe("a run start a finding's rewrite option opens", () => {
    const rewriteField: StartSeed = { pageIds: ["p-mugs"], kind: "generate", fields: ["seo_description"], rewrite: ["p-mugs"] };

    it("sends the page, the kind, the field scope and the page whose value is rewritten", () => {
        seeded(rewriteField);

        expect(priced()).toStrictEqual({
            siteId: "s1",
            pageIds: ["p-mugs"],
            publishMode: "draft",
            kind: "generate",
            budget: { maxUsd: 5, maxTokens: 0 },
            rewrite: ["p-mugs"],
            fields: ["seo_description"],
        });
    });

    it("shows the field scope and says the client's value is rewritten, with the rewrite already ticked", () => {
        seeded(rewriteField);

        expect(screen.getByText(said.fieldScope(["SEO description"]))).toBeDefined();
        expect(screen.getByText(said.rewriteField(["SEO description"], ["/mugs/"]))).toBeDefined();
        expect((screen.getByRole("switch", { name: said.rewrite }) as HTMLInputElement).checked).toBe(true);
    });

    it("keeps the client's value once the person switches the rewrite off", () => {
        seeded(rewriteField);

        fireEvent.click(screen.getByRole("switch", { name: said.rewrite }));

        const request = priced();
        expect(request).not.toHaveProperty("rewrite");
        expect(request["fields"]).toStrictEqual(["seo_description"]);
        expect(screen.queryByText(said.rewriteField(["SEO description"], ["/mugs/"]))).toBeNull();
    });
});

describe("a run start a finding's relink option opens", () => {
    it("sends a relink of the page with no field scope", () => {
        seeded({ pageIds: ["p-mugs"], kind: "relink", fields: [], rewrite: [] });

        const request = priced();
        expect(request["kind"]).toBe("relink");
        expect(request["pageIds"]).toStrictEqual(["p-mugs"]);
        expect(request).not.toHaveProperty("fields");
        expect(screen.queryByRole("region", { name: said.scopeTitle })).toBeNull();
    });
});
