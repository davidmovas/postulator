import { useCallback, useMemo } from "react";

import { flatten } from "../../data/call.js";
import { useEdges, useEntities } from "../../data/hooks/graph.js";
import { useRoleProfiles } from "../../data/hooks/models.js";
import { usePages } from "../../data/hooks/pages.js";
import { useProviderKeys } from "../../data/hooks/settings.js";
import { useSite, useSites } from "../../data/hooks/sites.js";
import { usePluginState } from "../../data/hooks/sync.js";
import { useTemplates } from "../../data/hooks/templates.js";
import type { EdgeStatus, ModelRole } from "../../generated/vocab.js";
import type { ReadinessStep } from "./model/readiness.js";
import { doneCount, readinessSteps } from "./model/readiness.js";

export type { ReadinessStep, ReadinessStepId } from "./model/readiness.js";

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

    const configured = (keys.data?.providers ?? []).filter((row) => row.configured).length;

    const profileRows = profiles.data?.profiles ?? [];
    const missingRoles = generateRoles
        .filter((role) => {
            const held = profileRows.find((row) => row.role === role);
            const effective = held?.effective ?? null;
            return effective === null || effective.model === "";
        })
        .join(", ");

    const selected = site.data?.site ?? (resolvedSiteId === (firstSite?.id ?? null) ? firstSite : null);
    const pageRows = flatten(pages.data?.pages);
    const templateRows = flatten(templates.data?.pages);
    const pluginState = plugin.data?.plugin ?? null;

    const steps = useMemo(
        () =>
            readinessSteps({
                siteId: resolvedSiteId,
                siteName: selected?.name ?? null,
                anySite: siteRows.length > 0 || selected !== null,
                configuredProviders: configured,
                missingRoles,
                pluginInstalled: pluginState?.installed === true,
                pluginVersion: pluginState?.version ?? "",
                pageCount: pageRows.length,
                syncedAt: pageRows.find((row) => row.lastSyncedAt !== null)?.lastSyncedAt ?? null,
                entityCount: flatten(entities.data?.pages).length,
                approvedEdgeCount: flatten(approvedEdges.data?.pages).length,
                orphanEntityCount: flatten(orphanEntities.data?.pages).length,
                templateAvailable: (selected?.defaults.templateId ?? null) !== null || templateRows.length > 0,
            }),
        [
            approvedEdges.data,
            configured,
            entities.data,
            missingRoles,
            orphanEntities.data,
            pageRows,
            pluginState,
            resolvedSiteId,
            selected,
            siteRows.length,
            templateRows.length,
        ],
    );

    const done = doneCount(steps);

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
