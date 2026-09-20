import { describe, expect, it } from "vitest";

import type { ReadinessInput, ReadinessStepId } from "./readiness.js";
import { doneCount, readinessSteps } from "./readiness.js";

const bare: ReadinessInput = {
    siteId: null,
    siteName: null,
    anySite: false,
    configuredProviders: 0,
    missingRoles: "writer, editor",
    pluginInstalled: false,
    pluginVersion: "",
    pageCount: 0,
    syncedAt: null,
    entityCount: 0,
    approvedEdgeCount: 0,
    orphanEntityCount: 0,
    templateAvailable: false,
};

const ready: ReadinessInput = {
    siteId: "s1",
    siteName: "Crema Bench",
    anySite: true,
    configuredProviders: 2,
    missingRoles: "",
    pluginInstalled: true,
    pluginVersion: "1.1.0",
    pageCount: 62,
    syncedAt: "2026-09-20T09:00:00Z",
    entityCount: 41,
    approvedEdgeCount: 40,
    orphanEntityCount: 0,
    templateAvailable: true,
};

function step(input: ReadinessInput, id: ReadinessStepId) {
    const found = readinessSteps(input).find((held) => held.id === id);
    if (found === undefined) {
        throw new Error(`no step ${id}`);
    }
    return found;
}

describe("readiness over a bare install", () => {
    it("has eight steps and none of them done", () => {
        const steps = readinessSteps(bare);
        expect(steps).toHaveLength(8);
        expect(doneCount(steps)).toBe(0);
    });

    it("blocks every site-scoped step until a site exists", () => {
        const blocked = readinessSteps(bare).filter((held) => held.blocked).map((held) => held.id);
        expect(blocked).toStrictEqual(["plugin", "sync", "graph", "canonicals", "template"]);
    });

    it("sends a site-scoped step to the sites screen while there is no site", () => {
        expect(step(bare, "sync").to).toBe("/sites");
        expect(step(bare, "graph").to).toBe("/sites");
    });
});

describe("readiness over a working site", () => {
    it("is complete", () => {
        expect(doneCount(readinessSteps(ready))).toBe(8);
    });

    it("deep links each step at the site it was checked against", () => {
        expect(step(ready, "sync").to).toBe("/s/s1/pages");
        expect(step(ready, "canonicals").to).toBe("/s/s1/graph?lens=noPage");
        expect(step(ready, "template").to).toBe("/s/s1/templates");
    });

    it("names the version the plugin reported", () => {
        expect(step(ready, "plugin").hint).toContain("1.1.0");
    });
});

describe("the steps that read a half-built site", () => {
    it("tells no entities from no approved edges", () => {
        expect(step({ ...ready, entityCount: 0, approvedEdgeCount: 0 }, "graph").done).toBe(false);
        const noEdges = step({ ...ready, approvedEdgeCount: 0 }, "graph");
        expect(noEdges.done).toBe(false);
        expect(noEdges.hint).toBe("Entities exist, but no edge is approved yet.");
    });

    it("fails the canonicals step while any entity has no page", () => {
        expect(step({ ...ready, orphanEntityCount: 3 }, "canonicals").done).toBe(false);
    });

    it("counts a page map as synced even before a sync stamped a row", () => {
        const step1 = step({ ...ready, syncedAt: null, pageCount: 12 }, "sync");
        expect(step1.done).toBe(true);
        expect(step1.hint).toBe("12 pages in the map.");
    });
});
