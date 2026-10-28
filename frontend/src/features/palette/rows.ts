import { copy } from "../../copy/index.js";
import type { SyncProgress } from "../../data/types.js";
import type { Timestamp } from "../../data/wire.js";
import { relativeTime } from "../../domain/format.js";
import { runKindLabel } from "../runs/labels.js";
import { stagePhrase } from "../sync/model.js";

export function syncActionLabel(active: SyncProgress | null): string {
    return active === null ? copy.palette.syncSite : copy.sync.palette.open(stagePhrase(active));
}

export interface RunRow {
    kind: string;
    stats: { items: number };
    startedAt: Timestamp;
    createdAt: Timestamp;
}

export function runRowLabel(run: RunRow, now: Date = new Date()): string {
    const when = run.startedAt === null || run.startedAt === "" ? run.createdAt : run.startedAt;
    return copy.palette.run(runKindLabel(run.kind), run.stats.items, relativeTime(when, now));
}

export function pagePrefix(query: string): string {
    const trimmed = query.trim();
    if (trimmed === "") {
        return "";
    }
    return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}
