export type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

export type JsonObject = { [key: string]: JsonValue };

function isObject(candidate: unknown): candidate is JsonObject {
    return typeof candidate === "object" && candidate !== null && !Array.isArray(candidate);
}

function equal(left: unknown, right: unknown): boolean {
    if (left === right) {
        return true;
    }
    if (Array.isArray(left) && Array.isArray(right)) {
        if (left.length !== right.length) {
            return false;
        }
        return left.every((held, index) => equal(held, right[index]));
    }
    if (isObject(left) && isObject(right)) {
        const leftKeys = Object.keys(left);
        const rightKeys = Object.keys(right);
        if (leftKeys.length !== rightKeys.length) {
            return false;
        }
        return leftKeys.every(
            (key) => Object.prototype.hasOwnProperty.call(right, key) && equal(left[key], right[key]),
        );
    }
    return false;
}

function withoutNulls(value: JsonValue): JsonValue {
    if (!isObject(value)) {
        return value;
    }
    const out: JsonObject = {};
    for (const [key, held] of Object.entries(value)) {
        if (held === null) {
            continue;
        }
        out[key] = withoutNulls(held);
    }
    return out;
}

export function mergePatch(base: unknown, edited: JsonValue): JsonValue | undefined {
    if (!isObject(edited)) {
        return equal(base, edited) ? undefined : edited;
    }
    if (!isObject(base)) {
        return withoutNulls(edited);
    }

    const patch: JsonObject = {};
    for (const [key, held] of Object.entries(edited)) {
        if (!Object.prototype.hasOwnProperty.call(base, key)) {
            if (held === null) {
                continue;
            }
            patch[key] = withoutNulls(held);
            continue;
        }
        const nested = mergePatch(base[key], held);
        if (nested !== undefined) {
            patch[key] = nested;
        }
    }
    for (const key of Object.keys(base)) {
        if (!Object.prototype.hasOwnProperty.call(edited, key)) {
            patch[key] = null;
        }
    }

    return Object.keys(patch).length === 0 ? undefined : patch;
}
