import { copy } from "../../copy/index.js";
import type { Timestamp } from "../../data/wire.js";
import { relativeTime } from "../../domain/format.js";
import { kindLabel } from "../runs/labels.js";

export interface RunRow {
    kind: string;
    stats: { items: number };
    startedAt: Timestamp;
    createdAt: Timestamp;
}

export function runRowLabel(run: RunRow, now: Date = new Date()): string {
    const when = run.startedAt === null || run.startedAt === "" ? run.createdAt : run.startedAt;
    return copy.palette.run(kindLabel(run.kind), run.stats.items, relativeTime(when, now));
}

export function pagePrefix(query: string): string {
    const trimmed = query.trim();
    if (trimmed === "") {
        return "";
    }
    return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}
