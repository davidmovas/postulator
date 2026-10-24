import { describe, expect, it } from "vitest";

import { mapLayout } from "./columns.js";

const everyColumn = ["page", "type", "parent", "body", "links", "findings", "children", "under"];
const withParent = ["page", "type", "parent", "body", "findings", "under"];
const keyFacts = ["page", "type", "body", "findings", "under"];

const leastMainBesideTheWidestDock = 960 - 600 - 68;

describe("the columns the site map shows at a width", () => {
    it("shows every column while the table gives them their least widths", () => {
        expect(mapLayout(1400).columns).toStrictEqual(everyColumn);
        expect(mapLayout(854).columns).toStrictEqual(everyColumn);
        expect(mapLayout(854).template).toBe("minmax(260px,3fr) minmax(72px,1fr) minmax(120px,1.4fr) 76px 64px 64px 52px 28px");
        expect(mapLayout(854).least).toBe(854);
    });

    it("drops the links and the children counts first, keeping where the parent came from", () => {
        expect(mapLayout(853).columns).toStrictEqual(withParent);
        expect(mapLayout(714).columns).toStrictEqual(withParent);
        expect(mapLayout(714).template).toBe("minmax(260px,3fr) minmax(72px,1fr) minmax(120px,1.4fr) 76px 64px 28px");
    });

    it("keeps the tree, the type, the body and the findings beside the dock, the page and the type shrinking first", () => {
        for (const width of [713, 500, 412, leastMainBesideTheWidestDock]) {
            expect(mapLayout(width).columns).toStrictEqual(keyFacts);
            expect(mapLayout(width).template).toBe("minmax(0,3fr) minmax(0,1fr) 76px 64px 28px");
        }
    });

    it("never asks for more than the window leaves it beside the widest dock", () => {
        expect(mapLayout(leastMainBesideTheWidestDock).least).toBeLessThanOrEqual(leastMainBesideTheWidestDock);
    });

    it("shows every column before the table has been measured", () => {
        expect(mapLayout(0).columns).toStrictEqual(everyColumn);
    });
});
