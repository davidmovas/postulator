import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../../copy/index.js";
import type { CatalogModel } from "../../../data/types.js";

const said = copy.settings.models.catalog;

interface Held {
    sent: unknown[];
}

const held: Held = { sent: [] };

vi.mock("../../../data/hooks/models.js", () => ({
    useUpsertModel: () => ({
        mutate: (request: unknown) => {
            held.sent.push(request);
        },
        isPending: false,
        error: null,
    }),
}));

const { ModelForm } = await import("./catalog-form.js");

function terra(): CatalogModel {
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
    } as CatalogModel;
}

function escaped(text: string): string {
    return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function input(label: string): HTMLInputElement {
    return screen.getByLabelText(new RegExp(`^${escaped(label)}\\*?$`));
}

function type(label: string, value: string): void {
    fireEvent.change(input(label), { target: { value } });
}

function save(): void {
    fireEvent.click(screen.getByRole("button", { name: copy.app.save }));
}

beforeEach(() => {
    held.sent = [];
});

describe("the model form", () => {
    it("names OpenAI as the provider and asks for no reasoning effort", () => {
        render(<ModelForm editing={null} onDone={() => undefined} />);

        expect(screen.queryByLabelText(new RegExp(`^${escaped(said.field.provider)}`))).toBeNull();
        expect(screen.getByText(said.providerFixed)).toBeDefined();
        expect(screen.queryByRole("combobox")).toBeNull();
    });

    it("sends every price, OpenAI as the provider and a zero for a price left empty", () => {
        render(<ModelForm editing={null} onDone={() => undefined} />);

        type(said.field.model, " gpt-house-blend ");
        type(said.field.contextTokens, "400000");
        type(said.field.maxOutputTokens, "64000");
        type(said.field.rpm, "60");
        type(said.field.tpm, "120000");
        type(said.field.inputUsdPerM, "2");
        type(said.field.cachedInputUsdPerM, "0.2");
        type(said.field.outputUsdPerM, "12");
        type(said.field.flexInputUsdPerM, "1");
        type(said.field.flexOutputUsdPerM, "6");
        save();

        expect(held.sent).toStrictEqual([
            {
                provider: "openai",
                model: "gpt-house-blend",
                contextTokens: 400_000,
                maxOutputTokens: 64_000,
                inputUsdPerM: 2,
                cachedInputUsdPerM: 0.2,
                cacheWriteUsdPerM: 0,
                outputUsdPerM: 12,
                flexInputUsdPerM: 1,
                flexCachedInputUsdPerM: 0,
                flexCacheWriteUsdPerM: 0,
                flexOutputUsdPerM: 6,
                rpm: 60,
                tpm: 120_000,
                supportsStructured: false,
                supportsImages: false,
                reasoning: false,
            },
        ]);
    });

    it("opens a model with its standard and flex prices and keeps them on save", () => {
        render(<ModelForm editing={terra()} onDone={() => undefined} />);

        expect(input(said.field.model).disabled).toBe(true);
        expect(input(said.field.cacheWriteUsdPerM).value).toBe("2.5");
        expect(input(said.field.flexCachedInputUsdPerM).value).toBe("0.1");
        save();

        expect(held.sent).toHaveLength(1);
        expect(held.sent[0]).toMatchObject({
            provider: "openai",
            model: "gpt-5.6-terra",
            cacheWriteUsdPerM: 2.5,
            flexInputUsdPerM: 1,
            flexCachedInputUsdPerM: 0.1,
            flexCacheWriteUsdPerM: 1.25,
            flexOutputUsdPerM: 6,
            reasoning: true,
        });
        expect(held.sent[0]).not.toHaveProperty("reasoningEffort");
    });

    it("shows a model without flex with its flex prices empty", () => {
        const plain = {
            ...terra(),
            flexInputUsdPerM: 0,
            flexCachedInputUsdPerM: 0,
            flexCacheWriteUsdPerM: 0,
            flexOutputUsdPerM: 0,
        };
        render(<ModelForm editing={plain} onDone={() => undefined} />);

        for (const label of [said.field.flexInputUsdPerM, said.field.flexOutputUsdPerM]) {
            expect(input(label).value).toBe("");
        }
        expect(screen.getByText(said.flexHelp)).toBeDefined();
    });

    it("saves nothing while a required number is missing or a price is not a number", () => {
        render(<ModelForm editing={terra()} onDone={() => undefined} />);

        type(said.field.inputUsdPerM, "");
        type(said.field.flexOutputUsdPerM, "six");
        save();

        expect(held.sent).toStrictEqual([]);
        expect(screen.getByText(copy.settings.problem.empty)).toBeDefined();
        expect(screen.getByText(copy.settings.problem.number)).toBeDefined();
    });
});
