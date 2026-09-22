import { useMutation, useQueryClient } from "@tanstack/react-query";

import { flatten } from "../call.js";
import {
    cancelRun,
    estimateRun,
    getArtifact,
    getRun,
    listArtifacts,
    listRunItems,
    listRuns,
    pauseRun,
    resumeRun,
    retryStep,
    revertRun,
    startRun,
} from "../endpoints/runs.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import type { ItemProgress, LiveStats } from "../runs/derive.js";
import { itemProgress, liveStats } from "../runs/derive.js";
import type { RunEventsState } from "../runs/log.js";
import { catchUpNow, ensureLog } from "../runs/log.js";
import { useRunEvents } from "../runs/use-run-events.js";
import { useUsage } from "./models.js";
import type { RunSort } from "../sorts.js";
import type { Run, RunFilter, RunItem, RunTotals } from "../types.js";
import { terminalRunStatuses } from "../../generated/vocab.js";

export function useRuns(filter: RunFilter = {}, sort: RunSort | null = null, limit?: number) {
    return useUnlockedInfinite<RunFilter, Run>({
        queryKey: keys.runs.list(filter, sort, limit),
        fetch: listRuns,
        filters: filter,
        sort,
        limit,
    });
}

export function useRun(runId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.runs.detail(runId ?? ""),
        queryFn: ({ signal }) => getRun({ runId: runId ?? "" }, signal),
        enabled: runId !== null && runId !== "",
    });
}

export function useRunItems(runId: string | null, status?: string, limit?: number) {
    return useUnlockedInfinite<{ runId: string; status?: string }, RunItem>({
        queryKey: keys.runs.items(runId ?? "", status, limit),
        fetch: listRunItems,
        filters: { runId: runId ?? "", status },
        sort: null,
        limit,
        enabled: runId !== null && runId !== "",
    });
}

export function useArtifact(itemId: string | null, kind: string) {
    return useUnlockedQuery({
        queryKey: keys.runs.artifact(itemId ?? "", kind),
        queryFn: ({ signal }) => getArtifact({ itemId: itemId ?? "", kind }, signal),
        enabled: itemId !== null && itemId !== "",
        retry: false,
    });
}

export function useArtifacts(itemId: string | null) {
    return useUnlockedQuery({
        queryKey: keys.runs.artifactsOf(itemId ?? ""),
        queryFn: ({ signal }) => listArtifacts({ itemId: itemId ?? "" }, signal),
        enabled: itemId !== null && itemId !== "",
    });
}

export interface RunProgress {
    run: Run | undefined;
    items: RunItem[];
    events: RunEventsState;
    stats: RunTotals | LiveStats;
    progress: ReadonlyMap<string, ItemProgress>;
    terminal: boolean;
    gap: boolean;
}

export function useRunProgress(runId: string): RunProgress {
    const row = useRun(runId);
    const rows = useRunItems(runId);
    const events = useRunEvents(runId);

    const spent = useUsage({ runId });

    const run = row.data?.run;
    const terminal = run !== undefined && (terminalRunStatuses as readonly string[]).includes(run.status);
    const counted = liveStats(events.events);
    const summary = spent.data;
    const live: LiveStats =
        summary === undefined
            ? counted
            : { ...counted, tokens: summary.usage.total, usd: summary.usd, calls: summary.calls };

    return {
        run,
        items: flatten(rows.data?.pages),
        events,
        stats: terminal && run !== undefined ? run.stats : live,
        progress: itemProgress(events.events),
        terminal,
        gap: events.maxSeq > events.contiguousSeq,
    };
}

export function useStartRun() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof startRun>[0]) => startRun(request),
        onSuccess: (answered) => {
            ensureLog(answered.runId);
            void catchUpNow(answered.runId);
            void client.invalidateQueries({ queryKey: keys.runs.lists() });
        },
    });
}

export function useEstimateRun() {
    return useMutation({
        mutationFn: (request: Parameters<typeof estimateRun>[0]) => estimateRun(request),
        retry: false,
    });
}

function useRunControl<Request extends { runId: string }, Answer>(
    call: (request: Request) => Promise<Answer>,
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onSettled: (_answered, _thrown, request) => {
            void client.invalidateQueries({ queryKey: keys.runs.detail(request.runId) });
            void client.invalidateQueries({ queryKey: keys.runs.itemsOf(request.runId) });
            void client.invalidateQueries({ queryKey: keys.runs.lists() });
        },
    });
}

export function usePauseRun() {
    return useRunControl((request: Parameters<typeof pauseRun>[0]) => pauseRun(request));
}

export function useResumeRun() {
    return useRunControl((request: Parameters<typeof resumeRun>[0]) => resumeRun(request));
}

export function useCancelRun() {
    return useRunControl((request: Parameters<typeof cancelRun>[0]) => cancelRun(request));
}

export function useRevertRun() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof revertRun>[0]) => revertRun(request),
        onSuccess: (answered) => {
            ensureLog(answered.runId);
            void catchUpNow(answered.runId);
            void client.invalidateQueries({ queryKey: keys.runs.lists() });
        },
    });
}

export function useRetryStep() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof retryStep>[0]) => retryStep(request),
        onSettled: () => {
            void client.invalidateQueries({ queryKey: keys.runs.itemLists() });
            void client.invalidateQueries({ queryKey: keys.runs.details() });
        },
    });
}
