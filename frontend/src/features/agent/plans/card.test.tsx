import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import type { Plan, PlanStepChoice } from "../../../data/types.js";
import { renderScreen } from "../../../testing/render.js";
import type { PlanBusy } from "./card.js";
import { PlanCard } from "./card.js";
import { aPlan, aStep, withStatus } from "./fixture.js";

const said = copy.plans;

interface Sent {
    choices: PlanStepChoice[];
    busy: Exclude<PlanBusy, null>;
}

function card(plan: Plan, sent: Sent[] = [], refusal: string | null = null) {
    return renderScreen(
        <PlanCard
            plan={plan}
            siteId="site1"
            busy={null}
            refusal={refusal}
            onConfirm={(choices, busy) => {
                sent.push({ choices, busy });
            }}
        />,
    );
}

const proposed = aPlan([
    aStep(1, { summary: "Add a related edge from Kits to Peptides", estimateUsd: 0 }),
    aStep(2, {
        summary: "Start a run on /kits/",
        dependsOn: [1],
        estimateUsd: 0.42,
        estimateTokens: 12000,
        edits: [{ name: "anchor", value: "peptide kits" }],
    }),
    aStep(3, { summary: "Relink /peptides/", dependsOn: [2] }),
]);

function box(number: number): HTMLInputElement {
    return screen.getByRole("checkbox", { name: said.tick(number) });
}

describe("a proposed plan", () => {
    it("lists every step with its reason, ticked by default", () => {
        card(proposed);

        expect(screen.getByText("Link the kits")).toBeDefined();
        expect(screen.getByText("Because of 2")).toBeDefined();
        expect(screen.getByText(said.after([1]))).toBeDefined();
        expect([1, 2, 3].every((number) => box(number).checked)).toBe(true);
    });

    it("unticks every step that waits on an unticked one before anything is sent", () => {
        card(proposed);

        fireEvent.click(box(1));

        expect([1, 2, 3].map((number) => box(number).checked)).toStrictEqual([false, false, false]);
        expect(screen.getByText(said.nothingChosen)).toBeDefined();
    });

    it("approves only the chosen steps", () => {
        const sent: Sent[] = [];
        card(proposed, sent);

        fireEvent.click(box(3));
        fireEvent.click(screen.getByRole("button", { name: said.approveChosen(2) }));

        expect(sent).toStrictEqual([
            {
                busy: "chosen",
                choices: [
                    { stepId: "s1", approve: true },
                    { stepId: "s2", approve: true },
                    { stepId: "s3", approve: false },
                ],
            },
        ]);
    });

    it("sends a changed edit with the step it belongs to", () => {
        const sent: Sent[] = [];
        card(proposed, sent);

        fireEvent.change(screen.getByDisplayValue("peptide kits"), { target: { value: "kits for peptides" } });
        fireEvent.click(screen.getByRole("button", { name: said.approveAll }));

        expect(sent[0]?.choices[1]).toStrictEqual({
            stepId: "s2",
            approve: true,
            edits: { anchor: "kits for peptides" },
        });
    });

    it("declines every step", () => {
        const sent: Sent[] = [];
        card(proposed, sent);

        fireEvent.click(screen.getByRole("button", { name: said.decline }));

        expect(sent[0]?.busy).toBe("decline");
        expect(sent[0]?.choices.every((choice) => !choice.approve)).toBe(true);
    });

    it("says why a confirmation was refused", () => {
        card(proposed, [], said.settled);

        expect(screen.getByText(said.settled)).toBeDefined();
    });
});

describe("a plan after approval", () => {
    const running = withStatus(
        aPlan([
            withStatus(aStep(1), "done"),
            withStatus(aStep(2, { runId: "run9" }), "waiting"),
            withStatus(aStep(3, { error: "the run did not land: budget" }), "failed"),
            withStatus(aStep(4, { error: "step 3 it depends on did not land: budget" }), "skipped"),
        ]),
        "running",
    );

    it("shows each step's live state and offers no controls", () => {
        card(running);

        expect(screen.queryAllByRole("checkbox")).toHaveLength(0);
        expect(screen.queryByRole("button", { name: said.approveAll })).toBeNull();
        expect(screen.getByText(said.stepStatuses.waiting)).toBeDefined();
        expect(screen.getByText(said.stepStatuses.failed)).toBeDefined();
        expect(screen.getByText("the run did not land: budget")).toBeDefined();
        expect(screen.getByText("step 3 it depends on did not land: budget")).toBeDefined();
    });

    it("links a waiting step to its run", () => {
        card(running);

        expect(screen.getByRole("link", { name: said.openRun }).getAttribute("href")).toBe("/s/site1/runs/run9");
    });
});
