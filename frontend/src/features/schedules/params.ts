import { choiceParam, queryCodec, textParam } from "../../data/params.js";

const enabledFilters = ["all", "on", "off"] as const;

export type EnabledFilter = (typeof enabledFilters)[number];

export interface SchedulesQuery {
    show: EnabledFilter;
    id: string;
}

const codec = queryCodec<SchedulesQuery>({
    show: choiceParam("show", enabledFilters, "all"),
    id: textParam("id"),
});

export const readQuery = codec.read;

export const writeQuery = codec.write;

export function enabledOf(show: EnabledFilter): boolean | undefined {
    switch (show) {
        case "on":
            return true;
        case "off":
            return false;
        default:
            return undefined;
    }
}
