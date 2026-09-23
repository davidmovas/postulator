import { describe, expect, it } from "vitest";

import { opensPalette, togglesDock } from "./keys.js";

const bare = { ctrl: false, meta: false, alt: false };
const free = { editable: false, overlayOpen: false };

describe("opensPalette", () => {
    const cases: readonly {
        name: string;
        stroke: { key: string; ctrl?: boolean; meta?: boolean; alt?: boolean };
        where?: { editable?: boolean; overlayOpen?: boolean };
        want: boolean;
    }[] = [
        { name: "f opens it", stroke: { key: "f" }, want: true },
        { name: "F opens it", stroke: { key: "F" }, want: true },
        { name: "f in a field types a letter", stroke: { key: "f" }, where: { editable: true }, want: false },
        { name: "f under an overlay does nothing", stroke: { key: "f" }, where: { overlayOpen: true }, want: false },
        { name: "ctrl+f is the browser's, not ours", stroke: { key: "f", ctrl: true }, want: false },
        { name: "alt+f is a menu", stroke: { key: "f", alt: true }, want: false },
        { name: "ctrl+k opens it", stroke: { key: "k", ctrl: true }, want: true },
        { name: "cmd+k opens it", stroke: { key: "k", meta: true }, want: true },
        {
            name: "ctrl+k opens it from inside a field",
            stroke: { key: "k", ctrl: true },
            where: { editable: true },
            want: true,
        },
        { name: "a bare k types a letter", stroke: { key: "k" }, want: false },
        { name: "ctrl+alt+k is neither", stroke: { key: "k", ctrl: true, alt: true }, want: false },
        { name: "another letter is not it", stroke: { key: "g" }, want: false },
    ];

    it.each(cases)("$name", ({ stroke, where, want }) => {
        expect(opensPalette({ ...bare, ...stroke }, { ...free, ...where })).toBe(want);
    });
});

describe("togglesDock", () => {
    it("takes ctrl+j and cmd+j", () => {
        expect(togglesDock({ ...bare, key: "j", ctrl: true })).toBe(true);
        expect(togglesDock({ ...bare, key: "J", meta: true })).toBe(true);
    });

    it("leaves a bare j and alt+j alone", () => {
        expect(togglesDock({ ...bare, key: "j" })).toBe(false);
        expect(togglesDock({ ...bare, key: "j", ctrl: true, alt: true })).toBe(false);
    });
});
