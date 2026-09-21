import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { formErrorOf } from "../../../data/errors.js";
import { useModelCatalog, useRoleProfiles, useSetProfile } from "../../../data/hooks/models.js";
import { modelRoles } from "../../../generated/vocab.js";
import {
    Banner,
    DenseTable,
    EmptyState,
    KeyOffIcon,
    Panel,
    PanelHeader,
    Select,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../../ui/index.js";
import { profileRows, refOf, refText } from "../model/profiles.js";

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
    const rows = profileRows([...modelRoles], profiles.data?.profiles ?? []);

    return (
        <Panel>
            <PanelHeader title={said.title} />
            {models.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={KeyOffIcon} title={said.title} body={copy.empty.models} />
                </div>
            ) : (
                <DenseTable columns="10rem minmax(12rem,1fr) 9rem" label={said.title}>
                    <TableHead>
                        <TableCell>{said.role}</TableCell>
                        <TableCell>{said.choice}</TableCell>
                        <TableCell>{""}</TableCell>
                    </TableHead>
                    {rows.map((row) => (
                        <TableRow key={row.role}>
                            <TableCell>{roleLabels[row.role] ?? row.role}</TableCell>
                            <TableCell>
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
                            </TableCell>
                            <TableCell>
                                {row.source === "global" ? null : (
                                    <StatusBadge tone="muted" dot={false}>
                                        {row.source === "seeded" ? said.sourceSeeded : said.sourceNone}
                                    </StatusBadge>
                                )}
                            </TableCell>
                        </TableRow>
                    ))}
                </DenseTable>
            )}
            {formErrorOf(choose.error) === null ? null : (
                <div className="px-3 pb-3">
                    <Banner tone="danger" title={formErrorOf(choose.error) ?? ""} />
                </div>
            )}
        </Panel>
    );
}
