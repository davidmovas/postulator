import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { copy } from "../../copy/index.js";
import { renderScreen } from "../../testing/render.js";
import { aSite, siteSettings } from "../../testing/sites.js";

const sent: unknown[] = [];

vi.mock("../../data/hooks/sites.js", () => ({
    useUpdateSite: () => ({
        mutate: (request: unknown) => {
            sent.push(request);
        },
        reset: () => {},
        isPending: false,
        error: null,
    }),
}));

const { SiteSettingsForm } = await import("./settings-form.js");

const said = copy.sites.settings;

function form() {
    return renderScreen(<SiteSettingsForm site={aSite()} />);
}

function field(label: string): HTMLInputElement {
    return screen.getByLabelText(label);
}

function save(): HTMLElement {
    return screen.getByRole("button", { name: said.save });
}

beforeEach(() => {
    sent.length = 0;
});

describe("the site's settings form", () => {
    it("shows what the site has stored, the neutral default as a blank", () => {
        form();

        expect(field(said.userAgent).value).toBe("");
        expect(field(said.userAgent).placeholder).toBe(said.userAgentDefault);
        expect(field(said.concurrency).value).toBe("4");
        expect(field(said.cadence).value).toBe("10");
        expect(save().hasAttribute("disabled")).toBe(true);
    });

    it("sends the whole stored settings back with only the edited fields changed", () => {
        form();

        fireEvent.change(field(said.userAgent), { target: { value: "  Agent/1  " } });
        fireEvent.change(field(said.concurrency), { target: { value: "8" } });
        fireEvent.click(save());

        expect(sent).toStrictEqual([
            {
                id: "s1",
                settings: {
                    ...siteSettings(),
                    userAgent: "Agent/1",
                    observe: { renderConcurrency: 8, fullWalkEvery: 10 },
                },
            },
        ]);
    });

    it.each([
        [said.concurrency, "17", said.problem.concurrency],
        [said.concurrency, "0", said.problem.concurrency],
        [said.concurrency, "two", said.problem.whole],
        [said.cadence, "0", said.problem.cadence],
        [said.cadence, "1.5", said.problem.whole],
    ])("refuses %s of %s inline and sends nothing", (label, value, problem) => {
        form();

        fireEvent.change(field(label), { target: { value } });

        expect(screen.getByText(problem)).toBeDefined();
        expect(save().hasAttribute("disabled")).toBe(true);
        fireEvent.click(save());
        expect(sent).toStrictEqual([]);
    });

    it("takes the edges of the ranges", () => {
        form();

        fireEvent.change(field(said.concurrency), { target: { value: "16" } });
        fireEvent.change(field(said.cadence), { target: { value: "1" } });
        fireEvent.click(save());

        expect(sent).toHaveLength(1);
        expect(screen.queryByText(said.problem.concurrency)).toBeNull();
    });

    it("puts back what is stored", () => {
        form();

        fireEvent.change(field(said.cadence), { target: { value: "3" } });
        fireEvent.click(screen.getByRole("button", { name: said.revert }));

        expect(field(said.cadence).value).toBe("10");
        expect(save().hasAttribute("disabled")).toBe(true);
    });
});
