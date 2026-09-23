import { copy } from "../../../copy/index.js";
import { relativeTime } from "../../../domain/format.js";

export type ReadinessStepId =
    | "providerKey"
    | "profiles"
    | "site"
    | "plugin"
    | "sync"
    | "graph"
    | "canonicals"
    | "template";

export interface ReadinessStep {
    id: ReadinessStepId;
    name: string;
    instruction: string;
    hint: string | null;
    done: boolean;
    blocked: boolean;
    to: string;
    action: string;
}

export interface ReadinessInput {
    siteId: string | null;
    siteName: string | null;
    anySite: boolean;
    configuredProviders: number;
    missingRoles: string;
    pluginInstalled: boolean;
    pluginVersion: string;
    pageCount: number;
    syncedAt: string | null;
    entityCount: number;
    approvedEdgeCount: number;
    orphanEntityCount: number;
    templateAvailable: boolean;
}

function sitePath(siteId: string | null, section: string): string {
    return siteId === null ? "/sites" : `/s/${siteId}/${section}`;
}

export function readinessSteps(input: ReadinessInput): readonly ReadinessStep[] {
    const blocked = !input.anySite;
    const graphDone = input.entityCount > 0 && input.approvedEdgeCount > 0;

    return [
        {
            id: "providerKey",
            name: copy.readiness.names.providerKey,
            instruction: copy.readiness.instructions.providerKey,
            hint:
                input.configuredProviders > 0
                    ? copy.readiness.done.providerKey(input.configuredProviders)
                    : copy.empty.providerKeys,
            done: input.configuredProviders > 0,
            blocked: false,
            to: "/settings/models",
            action: copy.readiness.actions.providerKey,
        },
        {
            id: "profiles",
            name: copy.readiness.names.profiles,
            instruction: copy.readiness.instructions.profiles,
            hint:
                input.missingRoles === ""
                    ? copy.readiness.done.profiles
                    : copy.readiness.todo.profiles(input.missingRoles),
            done: input.missingRoles === "",
            blocked: false,
            to: "/settings/models",
            action: copy.readiness.actions.profiles,
        },
        {
            id: "site",
            name: copy.readiness.names.site,
            instruction: copy.readiness.instructions.site,
            hint: input.anySite ? input.siteName : copy.empty.sites,
            done: input.anySite,
            blocked: false,
            to: "/sites",
            action: copy.readiness.actions.site,
        },
        {
            id: "plugin",
            name: copy.readiness.names.plugin,
            instruction: copy.readiness.instructions.plugin,
            hint: blocked
                ? copy.readiness.blocked
                : input.pluginInstalled
                  ? copy.readiness.done.plugin(input.pluginVersion)
                  : null,
            done: input.pluginInstalled,
            blocked,
            to: "/sites",
            action: copy.readiness.actions.plugin,
        },
        {
            id: "sync",
            name: copy.readiness.names.sync,
            instruction: copy.readiness.instructions.sync,
            hint: blocked
                ? copy.readiness.blocked
                : input.syncedAt !== null
                  ? copy.readiness.done.syncAt(relativeTime(input.syncedAt))
                  : input.pageCount > 0
                    ? copy.readiness.done.sync(input.pageCount)
                    : null,
            done: !blocked && input.pageCount > 0,
            blocked,
            to: sitePath(input.siteId, "pages"),
            action: copy.readiness.actions.sync,
        },
        {
            id: "graph",
            name: copy.readiness.names.graph,
            instruction: copy.readiness.instructions.graph,
            hint: blocked
                ? copy.readiness.blocked
                : graphDone
                  ? copy.readiness.done.graph
                  : input.entityCount === 0
                    ? copy.readiness.todo.graphNoEntities
                    : copy.readiness.todo.graphNoEdges,
            done: !blocked && graphDone,
            blocked,
            to: sitePath(input.siteId, "graph"),
            action: copy.readiness.actions.graph,
        },
        {
            id: "canonicals",
            name: copy.readiness.names.canonicals,
            instruction: copy.readiness.instructions.canonicals,
            hint: blocked
                ? copy.readiness.blocked
                : input.entityCount === 0
                  ? copy.readiness.todo.graphNoEntities
                  : input.orphanEntityCount === 0
                    ? copy.readiness.done.canonicals
                    : copy.readiness.todo.canonicals,
            done: !blocked && input.entityCount > 0 && input.orphanEntityCount === 0,
            blocked,
            to: `${sitePath(input.siteId, "graph")}?lens=noPage`,
            action: copy.readiness.actions.canonicals,
        },
        {
            id: "template",
            name: copy.readiness.names.template,
            instruction: copy.readiness.instructions.template,
            hint: blocked
                ? copy.readiness.blocked
                : input.templateAvailable
                  ? copy.readiness.done.template
                  : copy.empty.templates,
            done: !blocked && input.templateAvailable,
            blocked,
            to: sitePath(input.siteId, "templates"),
            action: copy.readiness.actions.template,
        },
    ];
}

export function doneCount(steps: readonly ReadinessStep[]): number {
    return steps.filter((step) => step.done).length;
}
