import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { CatalogModel } from "../../../data/types.js";

const said = copy.settings.models.catalog;

function model(overrides: Partial<CatalogModel>): CatalogModel {
    return {
        provider: "openai",
        model: "gpt-5.6-terra",
        contextTokens: 1_050_000,
        maxOutputTokens: 128_000,
        inputUsdPerM: 2,
        cachedInputUsdPerM: 0.2,
        cacheWriteUsdPerM: 2.5,
        outputUsdPerM: 12,
        flexInputUsdPerM: 1,
        flexCachedInputUsdPerM: 0.1,
        flexCacheWriteUsdPerM: 1.25,
        flexOutputUsdPerM: 6,
        rpm: 60,
        tpm: 120_000,
        supportsStructured: true,
        supportsImages: true,
        reasoning: true,
        ...overrides,
    } as CatalogModel;
}

const listed: CatalogModel[] = [
    model({}),
    model({
        model: "gpt-image-2",
        inputUsdPerM: 8,
        cachedInputUsdPerM: 0,
        cacheWriteUsdPerM: 0,
        outputUsdPerM: 30,
        flexInputUsdPerM: 0,
        flexCachedInputUsdPerM: 0,
        flexCacheWriteUsdPerM: 0,
        flexOutputUsdPerM: 0,
        reasoning: false,
    }),
];

vi.mock("../../../data/hooks/models.js", () => ({
    useModelCatalog: () => ({ data: { models: listed }, isPending: false, error: null }),
    useDisableModel: () => ({ mutate: () => undefined, isPending: false, error: null }),
    useUpsertModel: () => ({ mutate: () => undefined, isPending: false, error: null }),
}));

const { CatalogDrawer } = await import("./catalog.js");

function rows(): HTMLElement[] {
    return within(screen.getByRole("table", { name: said.title })).getAllByRole("row").slice(1);
}

describe("the model catalog", () => {
    it("shows each model by its name with its standard and flex prices per million tokens", () => {
        render(<CatalogDrawer onClose={() => undefined} />);

        const [terra, image] = rows();
        const terraCells = within(terra).getAllByRole("cell").map((cell) => cell.textContent);
        expect(terraCells.slice(0, 3)).toStrictEqual(["gpt-5.6-terra", "2 · 0.2 · 12", "1 · 6"]);

        const imageCells = within(image).getAllByRole("cell").map((cell) => cell.textContent);
        expect(imageCells.slice(0, 3)).toStrictEqual(["gpt-image-2", "8 · 8 · 30", said.noFlex]);
    });

    it("never names the provider beside a model", () => {
        render(<CatalogDrawer onClose={() => undefined} />);

        for (const row of rows()) {
            expect(row.textContent).not.toContain("openai/");
        }
    });
});
