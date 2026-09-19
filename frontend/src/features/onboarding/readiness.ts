import { useCallback, useMemo } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useEdges, useEntities } from "../../data/hooks/graph.js";
import { useRoleProfiles } from "../../data/hooks/models.js";
import { usePages } from "../../data/hooks/pages.js";
import { useProviderKeys } from "../../data/hooks/settings.js";
import { useSite, useSites } from "../../data/hooks/sites.js";
import { usePluginState } from "../../data/hooks/sync.js";
import { useTemplates } from "../../data/hooks/templates.js";
import { relativeTime } from "../../domain/format.js";
import type { EdgeStatus, ModelRole } from "../../generated/vocab.js";

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

export interface Readiness {
    steps: readonly ReadinessStep[];
    done: number;
    total: number;
    loading: boolean;
    ready: boolean;
    siteId: string | null;
    siteName: string | null;
    refresh: () => void;
}

const generateRoles: readonly ModelRole[] = ["writer", "editor", "linker", "judge"];

const approved: EdgeStatus = "approved";

function sitePath(siteId: string | null, section: string): string {
    return siteId === null ? "/sites" : `/s/${siteId}/${section}`;
}

export function useReadiness(siteId: string | null): Readiness {
    const keys = useProviderKeys();
    const sites = useSites();

    const siteRows = flatten(sites.data?.pages);
    const firstSite = siteRows.length > 0 ? siteRows[0] : null;
    const resolvedSiteId = siteId ?? firstSite?.id ?? null;
    const scoped = resolvedSiteId ?? "";

    const profiles = useRoleProfiles(resolvedSiteId ?? undefined);
    const site = useSite(resolvedSiteId);
    const plugin = usePluginState(resolvedSiteId);
    const pages = usePages({ siteId: scoped });
    const entities = useEntities({ siteId: scoped });
    const orphanEntities = useEntities({ siteId: scoped, hasCanonicalPage: false });
    const approvedEdges = useEdges({ siteId: scoped, status: approved });
    const templates = useTemplates();

    const refresh = useCallback(() => {
        void keys.refetch();
        void profiles.refetch();
        void sites.refetch();
        void site.refetch();
        void plugin.refetch();
        void pages.refetch();
        void entities.refetch();
        void orphanEntities.refetch();
        void approvedEdges.refetch();
        void templates.refetch();
    }, [keys, profiles, sites, site, plugin, pages, entities, orphanEntities, approvedEdges, templates]);

    const loading =
        keys.isLoading ||
        profiles.isLoading ||
        sites.isLoading ||
        site.isLoading ||
        plugin.isLoading ||
        pages.isLoading ||
        entities.isLoading ||
        orphanEntities.isLoading ||
        approvedEdges.isLoading ||
        templates.isLoading;

    const providerRows = keys.data?.providers ?? [];
    const configured = providerRows.filter((row) => row.configured);

    const profileRows = profiles.data?.profiles ?? [];
    const missingRoles = generateRoles
        .filter((role) => {
            const held = profileRows.find((row) => row.role === role);
            const effective = held?.effective ?? null;
            return effective === null || effective.model === "";
        })
        .join(", ");

    const selected = site.data?.site ?? (resolvedSiteId === (firstSite?.id ?? null) ? firstSite : null);
    const anySite = siteRows.length > 0 || selected !== null;

    const pageRows = flatten(pages.data?.pages);
    const syncedAt = pageRows.find((row) => row.lastSyncedAt !== null)?.lastSyncedAt ?? null;
    const entityRows = flatten(entities.data?.pages);
    const orphanRows = flatten(orphanEntities.data?.pages);
    const edgeRows = flatten(approvedEdges.data?.pages);
    const templateRows = flatten(templates.data?.pages);

    const steps = useMemo<readonly ReadinessStep[]>(() => {
        const blocked = !anySite;
        const pluginState = plugin.data?.plugin ?? null;
        const graphDone = entityRows.length > 0 && edgeRows.length > 0;
        const templateDone = (selected?.defaults.templateId ?? null) !== null || templateRows.length > 0;

        return [
            {
                id: "providerKey",
                name: copy.onboarding.names.providerKey,
                instruction: copy.readiness.providerKey,
                hint:
                    configured.length > 0
                        ? copy.onboarding.done.providerKey(configured.length)
                        : copy.empty.providerKeys,
                done: configured.length > 0,
                blocked: false,
                to: "/settings/models",
                action: copy.onboarding.actions.providerKey,
            },
            {
                id: "profiles",
                name: copy.onboarding.names.profiles,
                instruction: copy.readiness.profiles,
                hint:
                    missingRoles === ""
                        ? copy.onboarding.done.profiles
                        : copy.onboarding.todo.profiles(missingRoles),
                done: missingRoles === "",
                blocked: false,
                to: "/settings/models",
                action: copy.onboarding.actions.profiles,
            },
            {
                id: "site",
                name: copy.onboarding.names.site,
                instruction: copy.readiness.site,
                hint: anySite ? (selected?.name ?? firstSite?.name ?? null) : copy.empty.sites,
                done: anySite,
                blocked: false,
                to: "/sites",
                action: copy.onboarding.actions.site,
            },
            {
                id: "plugin",
                name: copy.onboarding.names.plugin,
                instruction: copy.readiness.plugin,
                hint: blocked
                    ? copy.onboarding.blocked
                    : pluginState?.installed === true
                      ? copy.onboarding.done.plugin(pluginState.version)
                      : null,
                done: pluginState?.installed === true,
                blocked,
                to: "/sites",
                action: copy.onboarding.actions.plugin,
            },
            {
                id: "sync",
                name: copy.onboarding.names.sync,
                instruction: copy.readiness.sync,
                hint: blocked
                    ? copy.onboarding.blocked
                    : syncedAt !== null
                      ? copy.onboarding.done.syncAt(relativeTime(syncedAt))
                      : pageRows.length > 0
                        ? copy.onboarding.done.sync(pageRows.length)
                        : null,
                done: !blocked && pageRows.length > 0,
                blocked,
                to: sitePath(resolvedSiteId, "pages"),
                action: copy.onboarding.actions.sync,
            },
            {
                id: "graph",
                name: copy.onboarding.names.graph,
                instruction: copy.readiness.graph,
                hint: blocked
                    ? copy.onboarding.blocked
                    : graphDone
                      ? copy.onboarding.done.graph
                      : entityRows.length === 0
                        ? copy.onboarding.todo.graphNoEntities
                        : copy.onboarding.todo.graphNoEdges,
                done: !blocked && graphDone,
                blocked,
                to: sitePath(resolvedSiteId, "graph"),
                action: copy.onboarding.actions.graph,
            },
            {
                id: "canonicals",
                name: copy.onboarding.names.canonicals,
                instruction: copy.readiness.canonicals,
                hint: blocked
                    ? copy.onboarding.blocked
                    : entityRows.length === 0
                      ? copy.onboarding.todo.graphNoEntities
                      : orphanRows.length === 0
                        ? copy.onboarding.done.canonicals
                        : copy.onboarding.todo.canonicals,
                done: !blocked && entityRows.length > 0 && orphanRows.length === 0,
                blocked,
                to: sitePath(resolvedSiteId, "pages"),
                action: copy.onboarding.actions.canonicals,
            },
            {
                id: "template",
                name: copy.onboarding.names.template,
                instruction: copy.readiness.template,
                hint: blocked
                    ? copy.onboarding.blocked
                    : templateDone
                      ? copy.onboarding.done.template
                      : copy.empty.templates,
                done: !blocked && templateDone,
                blocked,
                to: sitePath(resolvedSiteId, "templates"),
                action: copy.onboarding.actions.template,
            },
        ];
    }, [
        anySite,
        configured.length,
        edgeRows.length,
        entityRows.length,
        firstSite,
        missingRoles,
        orphanRows.length,
        pageRows.length,
        plugin.data,
        resolvedSiteId,
        selected,
        syncedAt,
        templateRows.length,
    ]);

    const done = steps.filter((step) => step.done).length;

    return {
        steps,
        done,
        total: steps.length,
        loading,
        ready: done === steps.length,
        siteId: resolvedSiteId,
        siteName: selected?.name ?? null,
        refresh,
    };
}
