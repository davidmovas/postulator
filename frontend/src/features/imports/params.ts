export const importTabs = ["import", "export"] as const;

export type ImportTab = (typeof importTabs)[number];

export const importSteps = ["file", "columns", "preview", "apply"] as const;

export type ImportStep = (typeof importSteps)[number];

export interface ImportQuery {
    tab: ImportTab;
    step: ImportStep;
    path: string;
    mappingId: string;
}

export const defaultQuery: ImportQuery = { tab: "import", step: "file", path: "", mappingId: "" };

function isTab(value: string): value is ImportTab {
    return (importTabs as readonly string[]).includes(value);
}

function isStep(value: string): value is ImportStep {
    return (importSteps as readonly string[]).includes(value);
}

export function stepIndex(step: ImportStep): number {
    return importSteps.indexOf(step);
}

export function reachable(step: ImportStep, path: string): boolean {
    return step === "file" || path !== "";
}

export function stepAfter(step: ImportStep): ImportStep {
    return importSteps[Math.min(stepIndex(step) + 1, importSteps.length - 1)] ?? step;
}

export function stepBefore(step: ImportStep): ImportStep {
    return importSteps[Math.max(stepIndex(step) - 1, 0)] ?? step;
}

export function readQuery(params: URLSearchParams): ImportQuery {
    const tab = params.get("tab") ?? "";
    const step = params.get("step") ?? "";
    const path = params.get("path") ?? "";
    const wanted = isStep(step) ? step : "file";
    return {
        tab: isTab(tab) ? tab : "import",
        step: reachable(wanted, path) ? wanted : "file",
        path,
        mappingId: params.get("mapping") ?? "",
    };
}

export function writeQuery(query: ImportQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.tab !== defaultQuery.tab) {
        params.set("tab", query.tab);
    }
    if (query.step !== defaultQuery.step && query.path !== "") {
        params.set("step", query.step);
    }
    if (query.path !== "") {
        params.set("path", query.path);
    }
    if (query.mappingId !== "") {
        params.set("mapping", query.mappingId);
    }
    return params;
}
