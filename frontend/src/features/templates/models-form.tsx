import type { ReactElement } from "react";
import { useMemo } from "react";

import { useModelCatalog } from "../../data/hooks/models.js";
import { copy } from "../../copy/index.js";
import { modelRoles } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import {
    Banner,
    DeleteIcon,
    EmptyState,
    Field,
    IconButton,
    Panel,
    PanelHeader,
    Select,
    SmartToyIcon,
} from "../../ui/index.js";
import { fieldErrorOf } from "./controls.js";
import { profilePath } from "./patch.js";
import type { LayerView } from "./provenance.js";
import { LayerField } from "./provenance.js";
import type { ProfileDraft, SpecDraft } from "./spec.js";

const separator = "/";

function refValue(profile: ProfileDraft): string {
    return `${profile.provider}${separator}${profile.model}`;
}

function refOf(value: string): { provider: string; model: string } {
    const at = value.indexOf(separator);
    return at < 0
        ? { provider: value, model: "" }
        : { provider: value.slice(0, at), model: value.slice(at + separator.length) };
}

export interface ModelsFormProps {
    draft: SpecDraft;
    layers: LayerView;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function ModelsForm({ draft, layers, error, onChange }: ModelsFormProps): ReactElement {
    const catalog = useModelCatalog();
    const models = useMemo(() => catalog.data?.models ?? [], [catalog.data]);

    const baseOptions: SelectOption<string>[] = models.map((model) => ({
        value: `${model.provider}${separator}${model.model}`,
        label: `${model.provider} · ${model.model}`,
    }));

    const pinnedRoles = draft.profiles.map((profile) => profile.role);
    const freeRoles = [...modelRoles].filter((role) => !pinnedRoles.includes(role));

    const replace = (role: string, patch: Partial<ProfileDraft>): void => {
        onChange({
            profiles: draft.profiles.map((profile) => (profile.role === role ? { ...profile, ...patch } : profile)),
        });
    };

    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.templates.models.title}>
                <div className="w-52">
                    <Select
                        aria-label={copy.templates.models.add}
                        value={null}
                        placeholder={freeRoles.length === 0 ? copy.templates.models.allPinned : copy.templates.models.add}
                        disabled={freeRoles.length === 0 || models.length === 0}
                        options={freeRoles.map((role) => ({ value: role, label: role }))}
                        onValueChange={(role) => {
                            const first = models[0];
                            onChange({
                                profiles: [
                                    ...draft.profiles,
                                    { role, provider: first?.provider ?? "", model: first?.model ?? "" },
                                ],
                            });
                        }}
                    />
                </div>
            </PanelHeader>
            <div className="flex flex-col gap-3 p-3">
                <p className="text-xs text-ink-dim">{copy.templates.models.body}</p>
                {models.length === 0 && !catalog.isPending ? (
                    <Banner tone="warn" title={copy.templates.models.catalogEmpty} />
                ) : null}
                {draft.profiles.length === 0 ? (
                    <EmptyState
                        icon={SmartToyIcon}
                        title={copy.templates.models.empty}
                        body={copy.templates.models.emptyBody}
                    />
                ) : (
                    draft.profiles.map((profile) => {
                        const current = refValue(profile);
                        const options = baseOptions.some((option) => option.value === current)
                            ? baseOptions
                            : [{ value: current, label: `${profile.provider} · ${profile.model}` }, ...baseOptions];
                        const base = layers.base.profiles.find((held) => held.role === profile.role);
                        return (
                            <LayerField
                                key={profile.role}
                                layers={layers}
                                path={profilePath(profile.role)}
                                templateValue={base === undefined ? copy.templates.layer.none : `${base.provider} ${base.model}`}
                                onFollow={() => {
                                    onChange({
                                        profiles:
                                            base === undefined
                                                ? draft.profiles.filter((held) => held.role !== profile.role)
                                                : draft.profiles.map((held) =>
                                                      held.role === profile.role ? { ...base } : held,
                                                  ),
                                    });
                                }}
                            >
                                <div className="flex items-end gap-2">
                                    <Field
                                        className="min-w-0 flex-1"
                                        label={profile.role}
                                        error={fieldErrorOf(error, `modelProfiles.${profile.role}`)}
                                    >
                                        {(control) => (
                                            <Select
                                                id={control.id}
                                                aria-describedby={control["aria-describedby"]}
                                                invalid={control.invalid}
                                                value={current}
                                                placeholder={copy.templates.models.pickModel}
                                                options={options}
                                                onValueChange={(next) => {
                                                    replace(profile.role, refOf(next));
                                                }}
                                            />
                                        )}
                                    </Field>
                                    <IconButton
                                        icon={DeleteIcon}
                                        label={copy.templates.models.remove(profile.role)}
                                        variant="ghost"
                                        onClick={() => {
                                            onChange({
                                                profiles: draft.profiles.filter((held) => held.role !== profile.role),
                                            });
                                        }}
                                    />
                                </div>
                            </LayerField>
                        );
                    })
                )}
            </div>
        </Panel>
    );
}
