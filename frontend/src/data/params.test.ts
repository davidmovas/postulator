import { describe, expect, it } from "vitest";

import { choiceParam, flagParam, listParam, queryCodec, sortParam, textParam, wantsNew } from "./params.js";

describe("textParam", () => {
    it("reads what the address holds and writes nothing for an empty string", () => {
        const param = textParam("q");
        expect(param.key).toBe("q");
        expect(param.read(null)).toBe("");
        expect(param.read("kiln")).toBe("kiln");
        expect(param.write("")).toBe("");
        expect(param.write("mug kiln")).toBe("mug kiln");
    });
});

describe("flagParam", () => {
    const param = flagParam("open");

    it.each([
        ["1", true],
        ["0", false],
        ["yes", false],
        ["", false],
        [null, false],
    ] as const)("reads %s as %s", (raw, want) => {
        expect(param.read(raw)).toBe(want);
    });

    it("writes 1 for on and nothing for off", () => {
        expect(param.write(true)).toBe("1");
        expect(param.write(false)).toBe("");
    });
});

describe("choiceParam", () => {
    it("falls back on a value it does not know and writes nothing for the fallback", () => {
        const param = choiceParam("view", ["map", "outline"], "map");
        expect(param.read("outline")).toBe("outline");
        expect(param.read("canvas")).toBe("map");
        expect(param.read(null)).toBe("map");
        expect(param.write("map")).toBe("");
        expect(param.write("outline")).toBe("outline");
    });

    it("may fall back on an empty choice that is not one of the values", () => {
        const param = choiceParam("status", ["planned", "published"], "");
        expect(param.read("published")).toBe("published");
        expect(param.read("done")).toBe("");
        expect(param.write("")).toBe("");
        expect(param.write("planned")).toBe("planned");
    });
});

describe("listParam", () => {
    const param = listParam("kinds", ["hub", "topic", "product"]);

    it("keeps the known values in the order the address lists them", () => {
        expect(param.read("topic, bogus,,hub")).toStrictEqual(["topic", "hub"]);
        expect(param.read("")).toStrictEqual([]);
        expect(param.read(null)).toStrictEqual([]);
    });

    it("joins the values with a comma and writes nothing for none", () => {
        expect(param.write(["product", "hub"])).toBe("product,hub");
        expect(param.write([])).toBe("");
    });
});

describe("sortParam", () => {
    const param = sortParam("sort", ["path", "createdAt"]);

    it("reads a declared field and its direction", () => {
        expect(param.read("path:desc")).toStrictEqual({ field: "path", desc: true });
        expect(param.read("status:asc")).toBeNull();
        expect(param.read(null)).toBeNull();
    });

    it("writes nothing for no sort", () => {
        expect(param.write({ field: "createdAt", desc: false })).toBe("createdAt:asc");
        expect(param.write(null)).toBe("");
    });
});

interface SampleQuery {
    view: "map" | "outline";
    search: string;
    kinds: readonly string[];
    open: boolean;
    sort: { field: "path" | "createdAt"; desc: boolean } | null;
}

const codec = queryCodec<SampleQuery>({
    view: choiceParam("view", ["map", "outline"], "map"),
    search: textParam("q"),
    kinds: listParam("kinds", ["hub", "topic"]),
    open: flagParam("open"),
    sort: sortParam("sort", ["path", "createdAt"]),
});

const full: SampleQuery = {
    view: "outline",
    search: "mug kiln",
    kinds: ["topic", "hub"],
    open: true,
    sort: { field: "path", desc: true },
};

describe("queryCodec", () => {
    it("reads every field's fallback from an empty address", () => {
        expect(codec.defaults).toStrictEqual({ view: "map", search: "", kinds: [], open: false, sort: null });
        expect(codec.read(new URLSearchParams())).toStrictEqual(codec.defaults);
    });

    it("writes the fields in the order they are declared, each under its own key", () => {
        expect(codec.write(full).toString()).toBe("view=outline&q=mug+kiln&kinds=topic%2Chub&open=1&sort=path%3Adesc");
        expect(codec.search(full)).toBe("?view=outline&q=mug+kiln&kinds=topic%2Chub&open=1&sort=path%3Adesc");
    });

    it("writes nothing for the defaults", () => {
        expect(codec.write(codec.defaults).toString()).toBe("");
        expect(codec.search(codec.defaults)).toBe("");
    });

    it("round-trips a full query", () => {
        expect(codec.read(codec.write(full))).toStrictEqual(full);
    });

    it("ignores keys no field declares", () => {
        expect(codec.write(codec.read(new URLSearchParams("action=new&q=kiln"))).toString()).toBe("q=kiln");
    });

    it("says whether a query carries any of the named fields", () => {
        expect(codec.carries(codec.defaults, ["view", "search", "kinds", "open", "sort"])).toBe(false);
        expect(codec.carries({ ...codec.defaults, sort: { field: "path", desc: false } }, ["search", "kinds"])).toBe(false);
        expect(codec.carries({ ...codec.defaults, open: true }, ["search", "open"])).toBe(true);
        expect(codec.carries({ ...codec.defaults, kinds: ["hub"] }, ["kinds"])).toBe(true);
        expect(codec.carries(full, [])).toBe(false);
    });
});

describe("wantsNew", () => {
    it("reads only the create action the palette sends", () => {
        expect(wantsNew(new URLSearchParams("action=new"))).toBe(true);
        expect(wantsNew(new URLSearchParams("action=edit"))).toBe(false);
        expect(wantsNew(new URLSearchParams())).toBe(false);
    });
});
