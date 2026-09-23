export const enabledFilters = ["all", "on", "off"] as const;

export type EnabledFilter = (typeof enabledFilters)[number];

export interface SchedulesQuery {
    show: EnabledFilter;
    id: string;
}

export const defaultQuery: SchedulesQuery = { show: "all", id: "" };

export const actionParam = "action";

export const actionNew = "new";

function isFilter(value: string): value is EnabledFilter {
    return (enabledFilters as readonly string[]).includes(value);
}

export function readQuery(params: URLSearchParams): SchedulesQuery {
    const show = params.get("show") ?? "";
    return { show: isFilter(show) ? show : "all", id: params.get("id") ?? "" };
}

export function writeQuery(query: SchedulesQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.show !== defaultQuery.show) {
        params.set("show", query.show);
    }
    if (query.id !== "") {
        params.set("id", query.id);
    }
    return params;
}

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

export function wantsNew(params: URLSearchParams): boolean {
    return params.get(actionParam) === actionNew;
}
