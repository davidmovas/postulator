import { useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import type { Envelope, EventType } from "../generated/events.js";
import { on } from "../lib/events.js";
import {
    applyConfirmRequested,
    applyConfirmResolved,
    applyDelta,
    applyDone,
    applyToolFinished,
    applyToolStarted,
    onStall,
    sweepStalled,
} from "./agent/turn.js";
import { coalesce, invalidate, invalidateAll, invalidateBySite } from "./invalidate.js";
import { keys } from "./keys.js";
import { markLocked, markUnlocked } from "./lock.js";
import type { RawRunEvent } from "./runs/decode.js";
import { ingestLive, liveRunIds, scheduleCatchUp } from "./runs/log.js";
import type { Run } from "./types.js";

export const itemCoalesceMs = 400;
export const usageCoalesceMs = 2000;
export const stallSweepMs = 15_000;

function record(envelope: Envelope<EventType>): RawRunEvent {
    return { seq: envelope.seq, type: envelope.type, at: envelope.at, payload: envelope.payload };
}

function runIdOf(envelope: Envelope<EventType>): string {
    if (typeof envelope.runId === "string" && envelope.runId !== "") {
        return envelope.runId;
    }
    const payload = envelope.payload as { runId?: unknown };
    return typeof payload.runId === "string" ? payload.runId : "";
}

function siteOfRun(client: QueryClient, runId: string): string | null {
    const held = client.getQueryData<{ run: Run }>(keys.runs.detail(runId));
    const siteId = held?.run.siteId;
    return typeof siteId === "string" && siteId !== "" ? siteId : null;
}

function fanOutSite(client: QueryClient, siteId: string | null): void {
    if (siteId === null) {
        invalidateAll(client, keys.pages.root(), keys.graph.root(), keys.reports.root());
        return;
    }
    invalidateBySite(client, keys.pages.lists(), siteId);
    invalidateBySite(client, keys.graph.entityLists(), siteId);
    invalidateBySite(client, keys.graph.edgeLists(), siteId);
    invalidateAll(client, keys.pages.tree(siteId), keys.graph.full(siteId), keys.reports.site(siteId));
}

function install(client: QueryClient): (() => void)[] {
    return [
        on("run.queued", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(`run.queued:${runId}`, () => {
                invalidateAll(client, keys.runs.lists(), keys.runs.detail(runId), keys.schedules.lists());
            });
        }),
        on("run.started", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(`run.started:${runId}`, () => {
                invalidateAll(client, keys.runs.detail(runId), keys.runs.lists());
            });
        }),
        on("run.paused", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            invalidateAll(client, keys.runs.detail(runId), keys.runs.lists());
        }),
        on("run.resumed", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            invalidateAll(client, keys.runs.detail(runId), keys.runs.lists());
        }),
        on("run.cancelled", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            invalidateAll(client, keys.runs.detail(runId), keys.runs.lists(), keys.runs.itemsOf(runId));
        }),
        on("run.completed", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            invalidateAll(
                client,
                keys.runs.detail(runId),
                keys.runs.lists(),
                keys.runs.itemsOf(runId),
                keys.reports.run(runId),
                keys.models.usageAll(),
            );
            fanOutSite(client, siteOfRun(client, runId));
        }),
        on("run.failed", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            invalidateAll(client, keys.runs.detail(runId), keys.runs.lists(), keys.runs.itemsOf(runId));
        }),
        on("run.budget_exceeded", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            invalidateAll(client, keys.runs.detail(runId), keys.runs.lists(), keys.models.usageAll());
        }),
        on("item.started", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(
                `item:${runId}`,
                () => {
                    invalidate(client, keys.runs.itemsOf(runId));
                },
                itemCoalesceMs,
            );
        }),
        on("item.done", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(
                `item:${runId}`,
                () => {
                    invalidateAll(client, keys.runs.itemsOf(runId), keys.runs.detail(runId));
                },
                itemCoalesceMs,
            );
        }),
        on("item.failed", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(
                `item:${runId}`,
                () => {
                    invalidateAll(client, keys.runs.itemsOf(runId), keys.runs.detail(runId));
                },
                itemCoalesceMs,
            );
        }),
        on("item.needs_human", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(
                `item:${runId}`,
                () => {
                    invalidateAll(client, keys.runs.itemsOf(runId), keys.runs.detail(runId));
                },
                itemCoalesceMs,
            );
        }),
        on("step.started", (envelope) => {
            ingestLive(runIdOf(envelope), record(envelope));
        }),
        on("step.done", (envelope) => {
            ingestLive(runIdOf(envelope), record(envelope));
        }),
        on("step.failed", (envelope) => {
            ingestLive(runIdOf(envelope), record(envelope));
        }),
        on("step.retrying", (envelope) => {
            ingestLive(runIdOf(envelope), record(envelope));
        }),
        on("llm.usage", (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(
                `usage:${runId}`,
                () => {
                    invalidate(client, keys.models.usage({ runId }));
                },
                usageCoalesceMs,
            );
        }),
        on("graph.changed", (envelope) => {
            const siteId = envelope.payload.siteId;
            coalesce(`graph:${siteId}`, () => {
                invalidateBySite(client, keys.graph.entityLists(), siteId);
                invalidateBySite(client, keys.graph.edgeLists(), siteId);
                invalidateBySite(client, keys.pages.lists(), siteId);
                invalidateAll(
                    client,
                    keys.graph.entityAll(),
                    keys.graph.full(siteId),
                    keys.reports.site(siteId),
                );
            });
        }),
        on("pages.changed", (envelope) => {
            const siteId = envelope.payload.siteId;
            coalesce(`pages:${siteId}`, () => {
                invalidateBySite(client, keys.pages.lists(), siteId);
                invalidateAll(
                    client,
                    keys.pages.details(),
                    keys.pages.tree(siteId),
                    keys.reports.site(siteId),
                    keys.graph.entityAll(),
                );
            });
        }),
        on("templates.changed", () => {
            coalesce("templates", () => {
                invalidateAll(client, keys.templates.root(), keys.policies.root());
            });
        }),
        on("app.locked", () => {
            markLocked(client);
        }),
        on("app.unlocked", () => {
            markUnlocked(client, { locked: false, protected: true });
        }),
        on("agent.delta", (envelope) => {
            applyDelta(envelope.payload);
        }),
        on("agent.tool.started", (envelope) => {
            applyToolStarted(envelope.payload);
        }),
        on("agent.tool.finished", (envelope) => {
            applyToolFinished(envelope.payload);
        }),
        on("agent.confirm.requested", (envelope) => {
            applyConfirmRequested(envelope.payload);
            invalidate(client, keys.agent.pendingAll());
        }),
        on("agent.confirm.resolved", (envelope) => {
            applyConfirmResolved(envelope.payload);
            invalidateAll(
                client,
                keys.agent.pendingAll(),
                keys.agent.messagesOf(envelope.payload.conversationId),
            );
        }),
        on("agent.done", (envelope) => {
            applyDone(envelope.payload);
            invalidateAll(
                client,
                keys.agent.messagesOf(envelope.payload.conversationId),
                keys.agent.conversationsAll(),
                keys.models.usage({ conversationId: envelope.payload.conversationId }),
                keys.sites.root(),
                keys.pages.root(),
                keys.graph.root(),
            );
        }),
        on("agent.titled", () => {
            invalidateAll(client, keys.agent.conversationsAll());
        }),
    ];
}

export function EventBridge(): null {
    const client = useQueryClient();

    useEffect(() => {
        const stops = install(client);
        const stopStallWatch = onStall((conversationId) => {
            invalidate(client, keys.agent.messagesOf(conversationId));
        });

        const wake = (): void => {
            for (const runId of liveRunIds()) {
                scheduleCatchUp(runId);
            }
            sweepStalled(Date.now());
        };
        const visibility = (): void => {
            if (document.visibilityState === "visible") {
                wake();
            }
        };
        window.addEventListener("focus", wake);
        document.addEventListener("visibilitychange", visibility);
        const sweep = setInterval(() => {
            sweepStalled(Date.now());
        }, stallSweepMs);

        return () => {
            for (const stop of stops) {
                stop();
            }
            stopStallWatch();
            window.removeEventListener("focus", wake);
            document.removeEventListener("visibilitychange", visibility);
            clearInterval(sweep);
        };
    }, [client]);

    return null;
}
