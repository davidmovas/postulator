import { useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

import type { Envelope, EventType } from "../generated/events.js";
import { eventTypes } from "../generated/events.js";
import { on } from "../lib/events.js";
import { reconcileConversation } from "./agent/reconcile.js";
import {
    applyConfirmRequested,
    applyConfirmResolved,
    applyDelta,
    applyDone,
    applyToolFinished,
    applyToolStarted,
    applyUsage,
    silentConversationIds,
} from "./agent/turn.js";
import { publishDrop } from "./drops.js";
import { coalesce, invalidate, invalidateAll, invalidateBySite } from "./invalidate.js";
import { keys } from "./keys.js";
import { markLocked, markUnlocked } from "./lock.js";
import type { RawRunEvent } from "./runs/decode.js";
import { ingestLive, liveRunIds, scheduleCatchUp } from "./runs/log.js";
import type { Run } from "./types.js";

export const itemCoalesceMs = 400;
export const usageCoalesceMs = 2000;
export const silenceSweepMs = 10_000;

type Handlers = { [T in EventType]: (envelope: Envelope<T>) => void };

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

function runLifecycle(client: QueryClient, envelope: Envelope<EventType>, ...extra: readonly (readonly unknown[])[]): void {
    const runId = runIdOf(envelope);
    ingestLive(runId, record(envelope));
    invalidateAll(client, keys.runs.detail(runId), keys.runs.lists(), ...extra);
}

function itemChanged(client: QueryClient, envelope: Envelope<EventType>, withDetail: boolean): void {
    const runId = runIdOf(envelope);
    ingestLive(runId, record(envelope));
    coalesce(
        `item:${runId}`,
        () => {
            if (withDetail) {
                invalidateAll(client, keys.runs.itemsOf(runId), keys.runs.detail(runId));
                return;
            }
            invalidate(client, keys.runs.itemsOf(runId));
        },
        itemCoalesceMs,
    );
}

function stepOnly(envelope: Envelope<EventType>): void {
    ingestLive(runIdOf(envelope), record(envelope));
}

function handlersFor(client: QueryClient): Handlers {
    return {
        "run.queued": (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(`run.queued:${runId}`, () => {
                invalidateAll(client, keys.runs.lists(), keys.runs.detail(runId), keys.schedules.lists());
            });
        },
        "run.started": (envelope) => {
            const runId = runIdOf(envelope);
            ingestLive(runId, record(envelope));
            coalesce(`run.started:${runId}`, () => {
                invalidateAll(client, keys.runs.detail(runId), keys.runs.lists());
            });
        },
        "run.paused": (envelope) => {
            runLifecycle(client, envelope);
        },
        "run.resumed": (envelope) => {
            runLifecycle(client, envelope);
        },
        "run.cancelled": (envelope) => {
            runLifecycle(client, envelope, keys.runs.itemsOf(runIdOf(envelope)));
        },
        "run.completed": (envelope) => {
            const runId = runIdOf(envelope);
            runLifecycle(
                client,
                envelope,
                keys.runs.itemsOf(runId),
                keys.reports.run(runId),
                keys.models.usageAll(),
            );
            fanOutSite(client, siteOfRun(client, runId));
        },
        "run.failed": (envelope) => {
            runLifecycle(client, envelope, keys.runs.itemsOf(runIdOf(envelope)));
        },
        "run.budget_exceeded": (envelope) => {
            runLifecycle(client, envelope, keys.models.usageAll());
        },
        "item.started": (envelope) => {
            itemChanged(client, envelope, false);
        },
        "item.done": (envelope) => {
            itemChanged(client, envelope, true);
        },
        "item.failed": (envelope) => {
            itemChanged(client, envelope, true);
        },
        "item.needs_human": (envelope) => {
            itemChanged(client, envelope, true);
        },
        "step.started": stepOnly,
        "step.done": stepOnly,
        "step.failed": stepOnly,
        "step.retrying": stepOnly,
        "llm.usage": (envelope) => {
            const runId = runIdOf(envelope);
            coalesce(
                `usage:${runId}`,
                () => {
                    invalidate(client, keys.models.usage({ runId }));
                    invalidate(client, keys.models.usageAll());
                },
                usageCoalesceMs,
            );
        },
        "graph.changed": (envelope) => {
            const siteId = envelope.payload.siteId;
            coalesce(`graph:${siteId}`, () => {
                invalidateBySite(client, keys.graph.entityLists(), siteId);
                invalidateBySite(client, keys.graph.edgeLists(), siteId);
                invalidateBySite(client, keys.pages.lists(), siteId);
                invalidateAll(client, keys.graph.entityAll(), keys.graph.full(siteId), keys.reports.site(siteId));
            });
        },
        "pages.changed": (envelope) => {
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
        },
        "templates.changed": () => {
            coalesce("templates", () => {
                invalidateAll(client, keys.templates.root(), keys.policies.root());
            });
        },
        "sites.changed": (envelope) => {
            const siteId = envelope.payload.siteId;
            coalesce(`sites:${siteId}`, () => {
                invalidateAll(client, keys.sites.root(), keys.sync.plugin(siteId), keys.reports.site(siteId));
            });
        },
        "settings.changed": () => {
            coalesce("settings", () => {
                invalidateAll(client, keys.settings.values(), keys.models.root());
            });
        },
        "schedules.changed": () => {
            coalesce("schedules", () => {
                invalidate(client, keys.schedules.root());
            });
        },
        "app.locked": () => {
            markLocked(client);
        },
        "app.unlocked": () => {
            markUnlocked(client, { locked: false, protected: true });
        },
        "agent.delta": (envelope) => {
            applyDelta(envelope.payload);
        },
        "agent.tool.started": (envelope) => {
            applyToolStarted(envelope.payload);
        },
        "agent.tool.finished": (envelope) => {
            applyToolFinished(envelope.payload);
        },
        "agent.confirm.requested": (envelope) => {
            applyConfirmRequested(envelope.payload);
            invalidate(client, keys.agent.pendingAll());
        },
        "agent.confirm.resolved": (envelope) => {
            applyConfirmResolved(envelope.payload);
            invalidateAll(
                client,
                keys.agent.pendingAll(),
                keys.agent.messagesOf(envelope.payload.conversationId),
            );
            void reconcileConversation(client, envelope.payload.conversationId);
        },
        "agent.usage": (envelope) => {
            const conversationId = envelope.payload.conversationId;
            applyUsage(envelope.payload);
            coalesce(
                `agent-usage:${conversationId}`,
                () => {
                    invalidate(client, keys.models.usage({ conversationId }));
                    invalidate(client, keys.models.usageAll());
                },
                usageCoalesceMs,
            );
        },
        "agent.done": (envelope) => {
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
        },
        "files.dropped": (envelope) => {
            publishDrop(envelope.payload.paths);
        },
        "agent.titled": () => {
            invalidateAll(client, keys.agent.conversationsAll());
        },
    };
}

function bind<T extends EventType>(type: T, handlers: Handlers): () => void {
    return on(type, handlers[type]);
}

export function EventBridge(): null {
    const client = useQueryClient();

    useEffect(() => {
        const handlers = handlersFor(client);
        const stops = eventTypes.map((type) => bind(type, handlers));

        const sweep = (): void => {
            for (const conversationId of silentConversationIds(Date.now())) {
                void reconcileConversation(client, conversationId);
            }
        };
        const wake = (): void => {
            for (const runId of liveRunIds()) {
                scheduleCatchUp(runId);
            }
            sweep();
        };
        const visibility = (): void => {
            if (document.visibilityState === "visible") {
                wake();
            }
        };
        window.addEventListener("focus", wake);
        document.addEventListener("visibilitychange", visibility);
        const timer = setInterval(sweep, silenceSweepMs);

        return () => {
            for (const stop of stops) {
                stop();
            }
            window.removeEventListener("focus", wake);
            document.removeEventListener("visibilitychange", visibility);
            clearInterval(timer);
        };
    }, [client]);

    return null;
}
