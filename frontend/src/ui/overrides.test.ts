import { describe, expect, it } from "vitest";

const sources = import.meta.glob("../**/*.tsx", { query: "?raw", import: "default", eager: true }) as Record<string, string>;

const sized = /<(Input|Select|Textarea)\b/;
const attribute = /className=(?:"([^"]*)"|\{cx\(([^)]*)\))/;
const conflicting = /(?:^|[\s"'`])[wh]-\S/;
const closing = /(?:\/>|^\s*>)\s*$/;

function offenders(path: string, source: string): string[] {
    const found: string[] = [];
    let inside = false;

    for (const [index, line] of source.split("\n").entries()) {
        if (sized.test(line)) {
            inside = true;
        }
        if (!inside) {
            continue;
        }
        const held = attribute.exec(line);
        if (held !== null && conflicting.test(`${held[1] ?? ""} ${held[2] ?? ""}`)) {
            found.push(`${path.replace("../", "")}:${index + 1}`);
        }
        if (closing.test(line.trimEnd())) {
            inside = false;
        }
    }
    return found;
}

describe("the control components", () => {
    it("reads every screen", () => {
        expect(Object.keys(sources).length).toBeGreaterThan(50);
    });

    it("are never sized through className, because cx cannot beat their own base classes", () => {
        const found = Object.entries(sources).flatMap(([path, source]) => offenders(path, source));
        expect(found).toEqual([]);
    });
});
