import type { RunStartRequest } from "../../data/types.js";
import { kindGenerate } from "./statuses.js";

export interface StartSeed {
    pageIds: readonly string[];
    kind: string;
    fields: readonly string[];
    rewrite: readonly string[];
}

export function seedOf(start: RunStartRequest): StartSeed {
    const kind = start.kind ?? "";
    return {
        pageIds: [...(start.pageIds ?? [])],
        kind: kind === "" ? kindGenerate : kind,
        fields: [...(start.fields ?? [])],
        rewrite: [...(start.rewrite ?? [])],
    };
}

export function seededFields(seed: StartSeed | null, kind: string): readonly string[] {
    return seed !== null && seed.kind === kind ? seed.fields : [];
}

export function seededRewrite(seed: StartSeed | null, kind: string): readonly string[] {
    return seed !== null && seed.kind === kind ? seed.rewrite : [];
}
