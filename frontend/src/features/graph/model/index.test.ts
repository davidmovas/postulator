import { describe, expect, it } from "vitest";

import { edge, entity, potteryEdges, potteryEntities } from "./fixture.js";
import { buildGraphIndex, nodeStateOf } from "./index.js";

const index = buildGraphIndex(potteryEntities, potteryEdges);

describe("buildGraphIndex", () => {
    it("places every entity under its best approved parent, roots by score", () => {
        expect(index.roots).toStrictEqual(["pottery", "teapots", "care"]);
        expect(index.children.get("pottery")).toStrictEqual(["mugs", "stone", "glazing"]);
        expect(index.children.get("mugs")).toStrictEqual(["m350", "m500", "travel"]);
        expect(index.placementParent.get("stone")).toBe("pottery");
    });

    it("places an entity under a proposed parent when no approved one exists", () => {
        expect(index.children.get("glazing")).toStrictEqual(["kiln"]);
        expect(index.placementParent.get("kiln")).toBe("glazing");
        expect([...index.placementProposed]).toStrictEqual(["kiln"]);
    });

    it("counts descendants and measures depth in the placement tree", () => {
        expect(index.subtreeCount.get("pottery")).toBe(7);
        expect(index.subtreeCount.get("mugs")).toBe(3);
        expect(index.subtreeCount.get("kiln")).toBe(0);
        expect(index.depth.get("kiln")).toBe(2);
        expect(index.depth.get("teapots")).toBe(0);
    });

    it("keeps every approved and proposed parent of an entity", () => {
        expect(index.approvedParents.get("stone")).toStrictEqual(["pottery", "mugs"]);
        expect(index.proposedParents.get("kiln")).toStrictEqual(["glazing"]);
        expect(index.approvedParents.get("care") ?? []).toStrictEqual([]);
    });

    it("flags the problems a node carries", () => {
        expect(index.problems.get("travel")).toStrictEqual({ noPage: true, orphan: false, proposed: 0, multiParent: false });
        expect(index.problems.get("kiln")).toStrictEqual({ noPage: true, orphan: true, proposed: 2, multiParent: false });
        expect(index.problems.get("care")).toStrictEqual({ noPage: false, orphan: true, proposed: 0, multiParent: false });
        expect(index.problems.get("stone")).toStrictEqual({ noPage: false, orphan: false, proposed: 1, multiParent: true });
        expect(index.problems.get("mugs")).toStrictEqual({ noPage: false, orphan: false, proposed: 1, multiParent: false });
        expect(index.problems.get("pottery")).toStrictEqual({ noPage: false, orphan: false, proposed: 0, multiParent: false });
    });

    it("totals the problems for the site", () => {
        expect(index.counts).toStrictEqual({
            total: 10, noPage: 2, orphan: 2, proposedEdges: 3, ai: 2, multiParent: 1,
            states: { mismatch: 0, working: 0, published: 0, exists: 0, planned: 0, archived: 0, noPage: 10 },
        });
    });

    it("lists related links from both ends, strongest first", () => {
        expect(index.related.get("mugs")).toStrictEqual([
            { edgeId: "e10", otherId: "teapots", weight: 0.86, status: "proposed" },
            { edgeId: "e9", otherId: "glazing", weight: 0.62, status: "approved" },
        ]);
        expect(index.related.get("teapots")).toStrictEqual([{ edgeId: "e10", otherId: "mugs", weight: 0.86, status: "proposed" }]);
    });

    it("orders the proposed edges by weight", () => {
        expect(index.proposedEdges.map((held) => held.id)).toStrictEqual(["e8", "e10", "e11"]);
    });

    it("ignores rejected edges and edges with an unknown endpoint", () => {
        const stray = buildGraphIndex(potteryEntities, [...potteryEdges, edge("x", "ghost", "pottery", "parent")]);
        expect(stray.roots).toStrictEqual(index.roots);
        expect(stray.counts).toStrictEqual(index.counts);
        expect(index.edgeById.has("e12")).toBe(false);
    });

    it("drops a proposed placement that would close a cycle", () => {
        const looped = buildGraphIndex(
            [entity({ id: "a", name: "A" }), entity({ id: "b", name: "B" })],
            [edge("ab", "a", "b", "parent"), edge("ba", "b", "a", "parent", "proposed")],
        );
        expect(looped.roots).toStrictEqual(["b"]);
        expect(looped.children.get("b")).toStrictEqual(["a"]);
        expect(looped.placementProposed.size).toBe(0);
        expect(looped.depth.get("a")).toBe(1);
    });

    it("is empty for no entities", () => {
        const empty = buildGraphIndex([], []);
        expect(empty.roots).toStrictEqual([]);
        expect(empty.counts.total).toBe(0);
    });
});

describe("node state", () => {
    const states = buildGraphIndex(potteryEntities, potteryEdges, [
        { entityId: "pottery", pageId: "page-pottery", path: "/pottery/", status: "published", work: "", mismatch: false },
        { entityId: "mugs", pageId: "page-mugs", path: "/pottery/mugs/", status: "published", work: "", mismatch: true },
        { entityId: "stone", pageId: "page-stone", path: "/pottery/stone/", status: "exists", work: "running", mismatch: false },
        { entityId: "care", pageId: "page-care", path: "/care/", status: "planned", work: "", mismatch: false },
    ]);

    it("reads the state of a node from its page", () => {
        expect(nodeStateOf(states, "pottery")).toBe("published");
        expect(nodeStateOf(states, "care")).toBe("planned");
    });

    it("puts a disagreement with the site above everything else", () => {
        expect(nodeStateOf(states, "mugs")).toBe("mismatch");
    });

    it("shows work in flight above the stored status", () => {
        expect(nodeStateOf(states, "stone")).toBe("working");
    });

    it("has no page where the graph has no page", () => {
        expect(nodeStateOf(states, "travel")).toBe("noPage");
        expect(nodeStateOf(index, "pottery")).toBe("noPage");
    });

    it("counts the nodes of each state", () => {
        expect(states.counts.states).toStrictEqual({
            mismatch: 1, working: 1, published: 1, exists: 0, planned: 1, archived: 0, noPage: 6,
        });
    });
});
