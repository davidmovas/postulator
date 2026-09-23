import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { settingGroups } from "../../../generated/vocab.js";
import { groupFallback, humanLabel, placements, sectionsOf, settingsTabKeys, tabLayout, unitOf } from "./layout.js";

const declared = Object.keys(copy.settings.keys);

function placed(keys: readonly string[]): string[] {
    return settingsTabKeys.flatMap((tab) => {
        const shaped = tabLayout(tab, keys);
        return [...shaped.plain, ...shaped.advanced].flatMap((section) => section.keys);
    });
}

describe("the placement of every declared setting", () => {
    it("places each key the copy declares exactly once", () => {
        const seen = placed(declared).sort();
        expect(seen).toEqual([...declared].sort());
    });

    it("keeps the promised settings out of the advanced cards", () => {
        const open = (tab: (typeof settingsTabKeys)[number]): number =>
            tabLayout(tab, declared).plain.flatMap((section) => section.keys).length;
        expect(open("models")).toBe(0);
        expect(open("runs")).toBe(7);
        expect(open("agent")).toBe(2);
        expect(open("browser")).toBe(1);
        expect(open("security")).toBe(0);
        expect(open("about")).toBe(0);
    });

    it("names no key the copy does not declare", () => {
        for (const placement of placements) {
            expect(declared).toContain(placement.key);
        }
    });

    it("gives a key the layout never names the advanced card of its own group", () => {
        const shaped = tabLayout("runs", [...declared, "runs.ghost"]);
        const advanced = shaped.advanced.flatMap((section) => section.keys);
        expect(advanced).toContain("runs.ghost");
        expect(shaped.plain.flatMap((section) => section.keys)).not.toContain("runs.ghost");
    });

    it("keeps a key of a group nothing has heard of, in the last advanced section", () => {
        const shaped = tabLayout("runs", [...declared, "future.thing"]);
        const held = shaped.advanced.find((section) => section.id === "other");
        expect(held?.keys).toEqual(["future.thing"]);
    });

    it("has a fallback section for every declared group, and it is advanced", () => {
        for (const group of settingGroups) {
            const section = groupFallback[group];
            const held = sectionsOf(section.tab).find((entry) => entry.id === section.id);
            expect(held?.advanced).toBe(true);
        }
    });

    it("draws a section only on the tab that owns it", () => {
        for (const placement of placements) {
            const owner = settingsTabKeys.filter((tab) =>
                sectionsOf(tab).some((section) => section.id === placement.section),
            );
            expect(owner).toHaveLength(1);
        }
    });

    it("draws the plain sections before the advanced ones", () => {
        for (const tab of settingsTabKeys) {
            const shaped = tabLayout(tab, declared);
            expect(shaped.plain.every((section) => !section.advanced)).toBe(true);
            expect(shaped.advanced.every((section) => section.advanced)).toBe(true);
        }
    });

    it("leaves out a section no declared key fills", () => {
        const shaped = tabLayout("agent", ["agent.loopLimit"]);
        expect(shaped.plain.map((section) => section.id)).toEqual(["agentLimits"]);
        expect(shaped.advanced).toHaveLength(0);
    });
});

describe("humanLabel", () => {
    it.each(["agent.historyBudgetChars", "wp.rateLimitPerSecond", "browser.torPath"])(
        "never prints the key %s",
        (key) => {
            const label = humanLabel(key);
            expect(label).not.toContain(".");
            expect(label[0]).toBe(label[0]?.toUpperCase());
        },
    );

    it("cuts a camel case leaf into words", () => {
        expect(humanLabel("runs.artifactRetentionDays")).toBe("Artifact retention days");
        expect(humanLabel("runs.workers")).toBe("Workers");
    });
});

describe("unitOf", () => {
    it("carries the unit of a counted setting", () => {
        expect(unitOf("runs.workers")).toBe("pages");
        expect(unitOf("runs.artifactRetentionDays")).toBe("days");
    });

    it("carries no unit where the value speaks for itself", () => {
        expect(unitOf("wp.proxyUrl")).toBeNull();
        expect(unitOf("agent.turnTimeout")).toBeNull();
    });
});
