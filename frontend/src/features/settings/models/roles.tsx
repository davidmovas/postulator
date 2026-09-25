import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { formErrorOf } from "../../../data/errors.js";
import { useModelCatalog, useRoleProfiles, useSetProfile } from "../../../data/hooks/models.js";
import {
    Banner,
    EmptyState,
    KeyOffIcon,
    Panel,
    PanelHeader,
    Select,
    StatusBadge,
} from "../../../ui/index.js";
import { pickableRoles, profileRows, refOf, refText } from "../model/profiles.js";

const said = copy.settings.models.profiles;
const roleLabels = said.labels as Readonly<Record<string, string>>;

export function RoleTable(): ReactElement {
    const profiles = useRoleProfiles();
    const catalog = useModelCatalog();
    const choose = useSetProfile();

    const models = catalog.data?.models ?? [];
    const options = models.map((model) => ({
        value: refText({ provider: model.provider, model: model.model }),
        label: refText({ provider: model.provider, model: model.model }),
    }));
    const rows = profileRows(pickableRoles, profiles.data?.profiles ?? []);

    return (
        <Panel>
            <PanelHeader title={said.title} />
            {models.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={KeyOffIcon} title={said.title} body={copy.empty.models} />
                </div>
            ) : (
                <div role="list" aria-label={said.title}>
                    {rows.map((row) => (
                        <div
                            key={row.role}
                            role="listitem"
                            className="flex h-9 items-center gap-3 border-b border-hairline px-3 last:border-b-0"
                        >
                            <span className="min-w-0 flex-1 truncate text-sm text-ink-soft">
                                {roleLabels[row.role] ?? row.role}
                            </span>
                            {row.source === "global" ? (
                                <StatusBadge tone="accent" dot={false}>
                                    {said.sourceGlobal}
                                </StatusBadge>
                            ) : null}
                            <div className="w-64 shrink-0">
                                <Select
                                    aria-label={`${roleLabels[row.role] ?? row.role} ${said.choice}`}
                                    value={row.chosen === null ? null : refText(row.chosen)}
                                    placeholder={row.effective === null ? said.sourceNone : refText(row.effective)}
                                    options={options}
                                    onValueChange={(picked) => {
                                        const ref = refOf(picked);
                                        if (ref === null) {
                                            return;
                                        }
                                        choose.mutate({ role: row.role, provider: ref.provider, model: ref.model });
                                    }}
                                />
                            </div>
                        </div>
                    ))}
                </div>
            )}
            <p className="border-t border-hairline px-3 py-2 text-xs text-ink-dim">{said.imagesElsewhere}</p>
            {formErrorOf(choose.error) === null ? null : (
                <div className="px-3 pb-3">
                    <Banner tone="danger" title={formErrorOf(choose.error) ?? ""} />
                </div>
            )}
        </Panel>
    );
}
