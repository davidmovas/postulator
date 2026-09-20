import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { copy } from "../../copy/index.js";
import { useSettingsSchema, useSettingValues } from "../../data/hooks/settings.js";
import { keys } from "../../data/keys.js";
import type { SettingDescriptor } from "../../data/types.js";
import { settingGroups } from "../../generated/vocab.js";
import { Button, EmptyState, Input, Panel, PanelHeader, RefreshIcon, Switch, TuneIcon } from "../../ui/index.js";
import { kindOf, sameValue } from "./model/value.js";
import { SettingRow } from "./setting-row.js";

const described = copy.settings.keys as Readonly<Record<string, { label: string; help: string }>>;
const groupProse = copy.settings.groups as Readonly<Record<string, { title: string; blurb: string }>>;

function matches(descriptor: SettingDescriptor, query: string): boolean {
    if (query === "") {
        return true;
    }
    const needle = query.toLowerCase();
    const prose = described[descriptor.key];
    return (
        descriptor.key.toLowerCase().includes(needle) ||
        (prose?.label.toLowerCase().includes(needle) ?? false) ||
        (prose?.help.toLowerCase().includes(needle) ?? false)
    );
}

export function GeneralSettingsScreen(): ReactElement {
    const client = useQueryClient();
    const schema = useSettingsSchema();
    const [query, setQuery] = useState("");
    const [changedOnly, setChangedOnly] = useState(false);

    const declared = useMemo(() => schema.data?.settings ?? [], [schema.data]);
    const settingKeys = useMemo(() => declared.map((descriptor) => descriptor.key), [declared]);
    const values = useSettingValues(settingKeys);

    const rows = declared.map((descriptor, index) => {
        const answered = values[index];
        return {
            descriptor,
            value: answered?.data === undefined ? descriptor.default : answered.data.value,
            loading: answered?.isPending ?? true,
            known: answered?.data !== undefined,
        };
    });

    const visible = rows.filter((row) => {
        if (!matches(row.descriptor, query)) {
            return false;
        }
        if (!changedOnly) {
            return true;
        }
        return row.known && !sameValue(kindOf(row.descriptor), row.value, row.descriptor.default);
    });

    const grouped = settingGroups.map((group) => ({
        group,
        rows: visible.filter((row) => row.descriptor.group === group),
    }));
    const populated = grouped.filter((entry) => entry.rows.length > 0);

    return (
        <div className="flex max-w-3xl flex-col gap-3">
            <div className="flex items-center gap-2">
                <div className="w-64">
                    <Input
                        aria-label={copy.settings.search}
                        placeholder={copy.settings.search}
                        value={query}
                        onChange={(event) => {
                            setQuery(event.target.value);
                        }}
                    />
                </div>
                <Switch
                    label={copy.settings.changedOnly}
                    checked={changedOnly}
                    onChange={(event) => {
                        setChangedOnly(event.target.checked);
                    }}
                />
                <Button
                    variant="ghost"
                    size="sm"
                    icon={RefreshIcon}
                    className="ml-auto"
                    onClick={() => {
                        void client.invalidateQueries({ queryKey: keys.settings.root() });
                    }}
                >
                    {copy.settings.reread}
                </Button>
            </div>

            {populated.length === 0 ? (
                <EmptyState icon={TuneIcon} title={copy.settings.title} body={copy.empty.settings} />
            ) : (
                populated.map(({ group, rows: groupRows }) => (
                    <Panel key={group}>
                        <PanelHeader title={groupProse[group]?.title ?? group} />
                        <p className="border-b border-hairline px-3 py-2 text-xs text-ink-dim">
                            {groupProse[group]?.blurb ?? ""}
                        </p>
                        {groupRows.map((row) => (
                            <SettingRow
                                key={row.descriptor.key}
                                descriptor={row.descriptor}
                                value={row.value}
                                loading={row.loading}
                            />
                        ))}
                    </Panel>
                ))
            )}
        </div>
    );
}
