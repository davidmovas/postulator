import { useMemo } from "react";

import { useSettingsSchema, useSettingValues } from "../../data/hooks/settings.js";
import type { SettingDescriptor } from "../../data/types.js";
import type { SectionId, SettingsTabKey } from "./model/layout.js";
import { tabLayout } from "./model/layout.js";

export interface SettingEntry {
    descriptor: SettingDescriptor;
    value: unknown;
    loading: boolean;
}

export interface SettingSectionView {
    id: SectionId;
    entries: readonly SettingEntry[];
}

export interface TabSettings {
    plain: readonly SettingSectionView[];
    advanced: readonly SettingSectionView[];
    pending: boolean;
    failed: boolean;
    retry: () => void;
}

export function useTabSettings(tab: SettingsTabKey): TabSettings {
    const schema = useSettingsSchema();
    const declared = useMemo(() => schema.data?.settings ?? [], [schema.data]);

    const shaped = useMemo(
        () =>
            tabLayout(
                tab,
                declared.map((descriptor) => descriptor.key),
            ),
        [tab, declared],
    );

    const ordered = useMemo(
        () => [...shaped.plain, ...shaped.advanced].flatMap((section) => section.keys),
        [shaped],
    );

    const answered = useSettingValues(ordered);
    const byKey = new Map<string, SettingDescriptor>(declared.map((descriptor) => [descriptor.key, descriptor]));

    const held = new Map<string, SettingEntry>();
    ordered.forEach((key, index) => {
        const descriptor = byKey.get(key);
        if (descriptor === undefined) {
            return;
        }
        const state = answered[index];
        held.set(key, {
            descriptor,
            value: state?.data === undefined ? descriptor.default : state.data.value,
            loading: state?.isPending ?? true,
        });
    });

    const view = (sections: readonly { id: SectionId; keys: readonly string[] }[]): SettingSectionView[] =>
        sections
            .map((section) => ({
                id: section.id,
                entries: section.keys.flatMap((key) => {
                    const entry = held.get(key);
                    return entry === undefined ? [] : [entry];
                }),
            }))
            .filter((section) => section.entries.length > 0);

    return {
        plain: view(shaped.plain),
        advanced: view(shaped.advanced),
        pending: schema.isPending,
        failed: schema.isError,
        retry: () => {
            void schema.refetch();
        },
    };
}
