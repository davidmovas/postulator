import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import type { RunStartRequest } from "../../data/types.js";
import { runKindsWithTheirOwnRecipe, startableRunKinds } from "../../generated/vocab.js";

export const templateAuto = "";

export function capOf(raw: string): number {
    const parsed = Number.parseFloat(raw);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

export function tokenCapOf(raw: string): number {
    const parsed = Number.parseInt(raw, 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

export function kindDoes(kind: string): string {
    const said: Record<string, string> = copy.runs.start.kindDoes;
    return said[kind] ?? "";
}

export function takesATemplate(kind: string): boolean {
    return !(runKindsWithTheirOwnRecipe as readonly string[]).includes(kind);
}

export interface GradedFinding {
    severity: string;
    code?: string;
    message?: string;
    pageId?: string;
    path?: string;
}

export interface GradedFindings {
    findings?: readonly GradedFinding[] | null;
}

export function blockingFindings(estimate: GradedFindings | null): number {
    if (estimate === null) {
        return 0;
    }
    return (estimate.findings ?? []).filter((finding) => finding.severity === "error").length;
}

export const startableKinds: readonly string[] = startableRunKinds;

export interface StartRefusal {
    targets: string | null;
    banner: string | null;
}

export function startRefusal(thrown: unknown): StartRefusal {
    if (thrown === null || thrown === undefined) {
        return { targets: null, banner: null };
    }
    const reaction = react(thrown);
    if (reaction.kind === "silent" || reaction.kind === "unlock") {
        return { targets: null, banner: null };
    }
    if (reaction.kind === "field" && reaction.field === "pageIds") {
        return { targets: reaction.message, banner: null };
    }
    return { targets: null, banner: reaction.message };
}

export interface StartForm {
    siteId: string;
    pageIds: readonly string[];
    publishMode: string;
    kind: string;
    templateId: string;
    cap: string;
    tokenCap: string;
    rewrite: readonly string[];
    fields: readonly string[];
}

export function startRequestOf(form: StartForm): RunStartRequest {
    const built: RunStartRequest = {
        siteId: form.siteId,
        pageIds: [...form.pageIds],
        publishMode: form.publishMode,
        kind: form.kind,
        budget: { maxUsd: capOf(form.cap), maxTokens: tokenCapOf(form.tokenCap) },
    };
    if (form.templateId !== templateAuto) {
        built.templateId = form.templateId;
    }
    if (form.rewrite.length > 0) {
        built.rewrite = [...form.rewrite];
    }
    if (form.fields.length > 0) {
        built.fields = [...form.fields];
    }
    return built;
}
