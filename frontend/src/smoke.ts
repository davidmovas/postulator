import type { Site } from "../bindings/github.com/davidmovas/postulator/internal/application/sites/models.js";
import type { Event as RunEvent } from "../bindings/github.com/davidmovas/postulator/internal/application/runs/models.js";

import { Agent, Runs, Sites } from "./lib/api.js";
import { parseError } from "./lib/errors.js";
import { on } from "./lib/events.js";
import { listOf, page } from "./lib/paging.js";

const catchUpLimit = 200;

export async function firstSite(): Promise<Site | undefined> {
    const listed = listOf<Site>(await Sites.List(page(25)));
    return listed.items[0];
}

export async function generate(siteId: string, pageIds: string[]): Promise<string> {
    const started = await Runs.Start({
        siteId,
        pageIds,
        publishMode: "draft",
        budget: { maxUsd: 5, maxTokens: 0 },
    });
    return started.runId;
}

export async function catchUp(runId: string, sinceSeq: number): Promise<RunEvent[]> {
    const logged = await Runs.ListEvents({ runId, sinceSeq, limit: catchUpLimit });
    return logged.events ?? [];
}

export function follow(runId: string, onSeq: (seq: number) => void): () => void {
    const unsubscribe = [
        on("run.queued", (envelope) => onSeq(envelope.seq)),
        on("run.started", (envelope) => onSeq(envelope.seq)),
        on("step.done", (envelope) => {
            if (envelope.payload.runId === runId) {
                onSeq(envelope.seq);
            }
        }),
        on("run.completed", (envelope) => onSeq(envelope.seq)),
        on("run.failed", (envelope) => {
            console.error(`${envelope.payload.code}: ${envelope.payload.message}`);
            onSeq(envelope.seq);
        }),
    ];
    return () => {
        for (const stop of unsubscribe) {
            stop();
        }
    };
}

export async function ask(conversationId: string, text: string): Promise<string> {
    const sent = await Agent.Send({ conversationId, text });
    return sent.messageId;
}

export async function smoke(): Promise<void> {
    const site = await firstSite();
    if (site === undefined) {
        return;
    }

    const conversation = await Agent.CreateConversation({ siteId: site.id, mode: "confirm" });
    await ask(conversation.conversation.id, "plan the pages under the running shoes hub");

    const runId = await generate(site.id, []);

    let seen = 0;
    const stop = follow(runId, (seq) => {
        seen = Math.max(seen, seq);
    });

    try {
        for (const recorded of await catchUp(runId, seen)) {
            seen = Math.max(seen, recorded.seq);
        }
    } catch (thrown: unknown) {
        const failure = parseError(thrown);
        console.error(`${failure.code}: ${failure.message}`);
    } finally {
        stop();
    }
}
