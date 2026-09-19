import type { InfiniteData, QueryClient } from "@tanstack/react-query";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import type { Cursor, List } from "../../lib/paging.js";
import {
    createSchedule,
    deleteSchedule,
    disableSchedule,
    enableSchedule,
    getSchedule,
    listSchedules,
    runScheduleNow,
    updateSchedule,
} from "../endpoints/schedules.js";
import { keys } from "../keys.js";
import { useUnlockedInfinite, useUnlockedQuery } from "../query.js";
import { catchUpNow, ensureLog } from "../runs/log.js";
import type { Schedule, ScheduleFilter } from "../types.js";

type SchedulePages = InfiniteData<List<Schedule>, Cursor | undefined>;

export function useSchedules(filter: ScheduleFilter = {}, limit?: number) {
    return useUnlockedInfinite<ScheduleFilter, Schedule>({
        queryKey: keys.schedules.list(filter, limit),
        fetch: listSchedules,
        filters: filter,
        sort: null,
        limit,
    });
}

export function useSchedule(id: string | null) {
    return useUnlockedQuery({
        queryKey: keys.schedules.detail(id ?? ""),
        queryFn: ({ signal }) => getSchedule({ id: id ?? "" }, signal),
        enabled: id !== null && id !== "",
    });
}

function useScheduleWrite<Request, Answer extends { schedule: Schedule }>(
    call: (request: Request) => Promise<Answer>,
) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onSuccess: (answered) => {
            client.setQueryData(keys.schedules.detail(answered.schedule.id), answered);
            void client.invalidateQueries({ queryKey: keys.schedules.lists() });
        },
    });
}

export function useCreateSchedule() {
    return useScheduleWrite((request: Parameters<typeof createSchedule>[0]) => createSchedule(request));
}

export function useUpdateSchedule() {
    return useScheduleWrite((request: Parameters<typeof updateSchedule>[0]) => updateSchedule(request));
}

export function useDeleteSchedule() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteSchedule>[0]) => deleteSchedule(request),
        onSuccess: (_answered, request) => {
            client.removeQueries({ queryKey: keys.schedules.detail(request.id) });
            void client.invalidateQueries({ queryKey: keys.schedules.lists() });
        },
    });
}

function patchEnabled(client: QueryClient, id: string, enabled: boolean): void {
    client.setQueriesData<SchedulePages>({ queryKey: keys.schedules.lists() }, (held) => {
        if (held === undefined) {
            return held;
        }
        return {
            ...held,
            pages: held.pages.map((loaded) => ({
                ...loaded,
                items: loaded.items.map((row) => (row.id === id ? { ...row, enabled } : row)),
            })),
        };
    });
}

function useScheduleToggle(call: (request: { id: string }) => Promise<{ schedule: Schedule }>, enabled: boolean) {
    const client = useQueryClient();
    return useMutation({
        mutationFn: call,
        onMutate: async (request: { id: string }) => {
            await client.cancelQueries({ queryKey: keys.schedules.lists() });
            const snapshot = client.getQueriesData<SchedulePages>({ queryKey: keys.schedules.lists() });
            patchEnabled(client, request.id, enabled);
            return { snapshot };
        },
        onError: (_thrown, _request, context) => {
            for (const [key, held] of context?.snapshot ?? []) {
                client.setQueryData(key, held);
            }
        },
        onSettled: (_answered, _thrown, request) => {
            void client.invalidateQueries({ queryKey: keys.schedules.lists() });
            void client.invalidateQueries({ queryKey: keys.schedules.detail(request.id) });
        },
    });
}

export function useEnableSchedule() {
    return useScheduleToggle((request) => enableSchedule(request), true);
}

export function useDisableSchedule() {
    return useScheduleToggle((request) => disableSchedule(request), false);
}

export function useRunScheduleNow() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof runScheduleNow>[0]) => runScheduleNow(request),
        onSuccess: (answered) => {
            if (answered.runId !== "") {
                ensureLog(answered.runId);
                void catchUpNow(answered.runId);
            }
            void client.invalidateQueries({ queryKey: keys.runs.lists() });
            void client.invalidateQueries({ queryKey: keys.schedules.lists() });
        },
    });
}
