import { useMemo } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useEntities } from "../../data/hooks/graph.js";
import { usePages } from "../../data/hooks/pages.js";
import { useRuns } from "../../data/hooks/runs.js";
import { useSettingsSchema } from "../../data/hooks/settings.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import { pushToast } from "../../data/toasts.js";
import { AddIcon, BoltIcon, DescriptionIcon, HubIcon, PlayCircleIcon, SmartToyIcon, SyncIcon, TuneIcon } from "../../ui/index.js";
import type { IconComponent } from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import { humanLabel, tabOf } from "../settings/model/layout.js";
import { rankBy } from "./rank.js";
import { pagePrefix, runRowLabel } from "./rows.js";
import { closePalette } from "./state.js";

const described = copy.settings.keys as Readonly<Record<string, { label: string; help: string }>>;

const perSection = 6;

export interface Row {
    key: string;
    label: string;
    kind: string;
    Icon: IconComponent;
    run: () => void;
}

export interface Section {
    key: string;
    label: string;
    rows: readonly Row[];
}

export interface Destination {
    key: string;
    label: string;
    to: string;
    Icon: IconComponent;
}

export function usePaletteSections(siteId: string | null, destinations: readonly Destination[], settled: string): readonly Section[] {
    const navigate = useNavigate();
    const sync = useSyncSite();

    const entities = useEntities(
        { siteId: siteId ?? "", namePrefix: settled },
        { field: "name", desc: false },
        perSection,
    );
    const pages = usePages(
        { siteId: siteId ?? "", pathPrefix: pagePrefix(settled) },
        { field: "path", desc: false },
        perSection,
    );
    const runs = useRuns(siteId === null ? {} : { siteId }, null, perSection);
    const schema = useSettingsSchema();

    const go = (to: string): (() => void) => {
        return () => {
            closePalette();
            void navigate(to);
        };
    };

    return useMemo<readonly Section[]>(() => {
        const said = copy.palette;
        const screens: Row[] = rankBy(settled, destinations, (entry) => entry.label, perSection).map(
            (entry) => ({
                key: `screen:${entry.key}`,
                label: entry.label,
                kind: said.kinds.screen,
                Icon: entry.Icon,
                run: go(entry.to),
            }),
        );

        const actions: Row[] = [];
        if (siteId !== null) {
            actions.push({
                key: "action:run",
                label: said.newRun,
                kind: said.kinds.action,
                Icon: BoltIcon,
                run: go(`/s/${siteId}/runs?action=new`),
            });
            actions.push({
                key: "action:entity",
                label: said.newEntity,
                kind: said.kinds.action,
                Icon: AddIcon,
                run: go(`/s/${siteId}/graph?action=new`),
            });
            actions.push({
                key: "action:sync",
                label: said.syncSite,
                kind: said.kinds.action,
                Icon: SyncIcon,
                run: () => {
                    closePalette();
                    sync.mutate(
                        { siteId },
                        {
                            onSuccess: () => {
                                pushToast("info", said.syncing);
                            },
                        },
                    );
                },
            });
        }
        actions.push({
            key: "action:agent",
            label: said.askAgent,
            kind: said.kinds.action,
            Icon: SmartToyIcon,
            run: () => {
                closePalette();
                askAgent("");
            },
        });

        const entityRows: Row[] = flatten(entities.data?.pages).map((entity) => ({
            key: `entity:${entity.id}`,
            label: entity.name,
            kind: said.kinds.entity,
            Icon: HubIcon,
            run: go(`/s/${entity.siteId}/graph/${entity.id}`),
        }));

        const pageRows: Row[] = flatten(pages.data?.pages).map((page) => ({
            key: `page:${page.id}`,
            label: page.path,
            kind: said.kinds.page,
            Icon: DescriptionIcon,
            run: go(`/s/${page.siteId}/pages/${page.id}`),
        }));

        const runRows: Row[] = rankBy(
            settled,
            flatten(runs.data?.pages),
            (held) => `${held.kind} ${held.status}`,
            perSection,
        ).map((held) => ({
            key: `run:${held.id}`,
            label: runRowLabel(held),
            kind: said.kinds.run,
            Icon: PlayCircleIcon,
            run: go(`/s/${held.siteId}/runs/${held.id}`),
        }));

        const settingRows: Row[] = rankBy(
            settled,
            (schema.data?.settings ?? []).map((descriptor) => ({
                key: descriptor.key,
                label: described[descriptor.key]?.label ?? humanLabel(descriptor.key),
                matched: `${described[descriptor.key]?.label ?? humanLabel(descriptor.key)} ${descriptor.key.split(".").join(" ")}`,
            })),
            (entry) => entry.matched,
            perSection,
        ).map((entry) => ({
            key: `setting:${entry.key}`,
            label: entry.label,
            kind: said.kinds.setting,
            Icon: TuneIcon,
            run: go(`/settings/${tabOf(entry.key)}`),
        }));

        return [
            { key: "goTo", label: said.goTo, rows: screens },
            { key: "actions", label: said.actions, rows: rankBy(settled, actions, (row) => row.label, perSection) },
            { key: "entities", label: said.entities, rows: entityRows },
            { key: "pages", label: said.pages, rows: pageRows },
            { key: "runs", label: said.runs, rows: runRows },
            { key: "settings", label: said.settings, rows: settingRows },
        ].filter((section) => section.rows.length > 0);
    }, [settled, siteId, destinations, entities.data, pages.data, runs.data, schema.data, navigate, sync]);
}
