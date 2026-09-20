import type { SettingDescriptor } from "../../../data/types.js";

export type SettingKind = "bool" | "int" | "string" | "enum" | "duration";

export type SettingProblem =
    | { kind: "empty" }
    | { kind: "shape" }
    | { kind: "range"; min: string | null; max: string | null };

const durationUnits: Readonly<Record<string, number>> = {
    ns: 1,
    us: 1000,
    "\u00b5s": 1000,
    ms: 1000000,
    s: 1000000000,
    m: 60000000000,
    h: 3600000000000,
};

const durationPart = /^(\d+(?:\.\d+)?)(ns|us|\u00b5s|ms|s|m|h)/;

export function parseDuration(text: string): number | null {
    let rest = text.trim();
    if (rest === "") {
        return null;
    }

    let sign = 1;
    if (rest.startsWith("-") || rest.startsWith("+")) {
        sign = rest.startsWith("-") ? -1 : 1;
        rest = rest.slice(1);
    }
    if (rest === "0") {
        return 0;
    }

    let total = 0;
    while (rest !== "") {
        const matched = durationPart.exec(rest);
        if (matched === null) {
            return null;
        }
        total += Number(matched[1]) * (durationUnits[matched[2] ?? ""] ?? 0);
        rest = rest.slice(matched[0].length);
    }
    return sign * total;
}

export function boundNumber(bound: unknown): number | null {
    return typeof bound === "number" && Number.isFinite(bound) ? bound : null;
}

export function boundText(bound: unknown): string | null {
    return typeof bound === "string" && bound !== "" ? bound : null;
}

export function kindOf(descriptor: SettingDescriptor): SettingKind {
    switch (descriptor.type) {
        case "bool":
        case "int":
        case "enum":
        case "duration":
            return descriptor.type;
        default:
            return "string";
    }
}

export function renderValue(kind: SettingKind, raw: unknown): string {
    if (kind === "bool") {
        return raw === true ? "true" : "false";
    }
    if (kind === "int") {
        const held = boundNumber(raw);
        return held === null ? "" : String(held);
    }
    return typeof raw === "string" ? raw : "";
}

export function parseValue(kind: SettingKind, text: string): unknown {
    if (kind === "bool") {
        return text === "true";
    }
    if (kind === "int") {
        return Number(text.trim());
    }
    return text.trim();
}

export function sameValue(kind: SettingKind, left: unknown, right: unknown): boolean {
    if (kind === "duration") {
        const first = typeof left === "string" ? parseDuration(left) : null;
        const second = typeof right === "string" ? parseDuration(right) : null;
        return first !== null && second !== null ? first === second : left === right;
    }
    return left === right;
}

function rangeProblem(min: string | null, max: string | null): SettingProblem {
    return { kind: "range", min, max };
}

export function checkSetting(descriptor: SettingDescriptor, text: string): SettingProblem | null {
    const kind = kindOf(descriptor);
    const trimmed = text.trim();

    if (kind === "int") {
        if (!/^-?\d+$/.test(trimmed)) {
            return { kind: "shape" };
        }
        const held = Number(trimmed);
        const min = boundNumber(descriptor.min);
        const max = boundNumber(descriptor.max);
        if ((min !== null && held < min) || (max !== null && held > max)) {
            return rangeProblem(min === null ? null : String(min), max === null ? null : String(max));
        }
        return null;
    }

    if (kind === "duration") {
        const held = parseDuration(trimmed);
        if (held === null) {
            return { kind: "shape" };
        }
        const min = boundText(descriptor.min);
        const max = boundText(descriptor.max);
        const floor = min === null ? null : parseDuration(min);
        const ceiling = max === null ? null : parseDuration(max);
        if ((floor !== null && held < floor) || (ceiling !== null && held > ceiling)) {
            return rangeProblem(min, max);
        }
        return null;
    }

    if (kind === "string" && descriptor.nonEmpty === true && trimmed === "") {
        return { kind: "empty" };
    }
    return null;
}
