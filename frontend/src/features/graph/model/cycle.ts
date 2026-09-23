import { failure } from "../../../data/errors.js";

export function cycleOf(thrown: unknown): string[] | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reported = failure(thrown);
    if (reported.code !== "INVALID") {
        return null;
    }
    const cycle = (reported.details ?? {})["cycle"];
    if (!Array.isArray(cycle)) {
        return null;
    }
    const ids = cycle.filter((entry): entry is string => typeof entry === "string");
    return ids.length === 0 ? null : ids;
}
