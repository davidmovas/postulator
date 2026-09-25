import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { useModelCatalog } from "../../../data/hooks/models.js";
import type { SelectOption } from "../../../ui/index.js";
import { Banner, cx, IconButton, RestartAltIcon, Select } from "../../../ui/index.js";
import { pickableRoles } from "../../settings/model/profiles.js";
import { fieldErrorOf } from "../controls.js";
import { roleLabel } from "../labels.js";
import type { ProfileDraft, SpecDraft } from "../spec.js";

const unpinned = "none";
const separator = "/";

function refValue(profile: ProfileDraft | undefined): string {
    return profile === undefined ? unpinned : `${profile.provider}${separator}${profile.model}`;
}

function refOf(value: string): { provider: string; model: string } {
    const at = value.indexOf(separator);
    return at < 0
        ? { provider: value, model: "" }
        : { provider: value.slice(0, at), model: value.slice(at + separator.length) };
}

export interface ModelsGroupProps {
    draft: SpecDraft;
    below: SpecDraft;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function ModelsGroup({ draft, below, error, onChange }: ModelsGroupProps): ReactElement {
    const catalog = useModelCatalog();
    const models = useMemo(() => catalog.data?.models ?? [], [catalog.data]);
    const roles = useMemo(() => {
        const extra = draft.profiles.map((profile) => profile.role).filter((role) => !pickableRoles.includes(role));
        return [...pickableRoles, ...extra];
    }, [draft.profiles]);

    const catalogOptions: readonly SelectOption<string>[] = models.map((model) => ({
        value: `${model.provider}${separator}${model.model}`,
        label: `${model.provider} · ${model.model}`,
    }));

    const pin = (role: string, value: string): void => {
        if (value === unpinned) {
            onChange({ profiles: draft.profiles.filter((held) => held.role !== role) });
            return;
        }
        const ref = refOf(value);
        onChange({
            profiles: draft.profiles.some((held) => held.role === role)
                ? draft.profiles.map((held) => (held.role === role ? { ...held, ...ref } : held))
                : [...draft.profiles, { role, ...ref }],
        });
    };

    return (
        <div className="flex flex-col gap-3">
            {models.length === 0 && !catalog.isPending ? (
                <Banner tone="warn" title={copy.templates.models.catalogEmpty} />
            ) : null}
            <section className="overflow-hidden rounded-lg border border-hairline bg-panel">
                <header className="grid h-7 grid-cols-[minmax(4rem,7rem)_minmax(0,1fr)_28px] items-center gap-2 border-b border-hairline bg-inset px-3 text-2xs font-semibold tracking-label text-ink-faint uppercase">
                    <div>{copy.templates.models.role}</div>
                    <div>{copy.templates.models.model}</div>
                    <div />
                </header>
                {roles.map((role) => {
                    const held = draft.profiles.find((profile) => profile.role === role);
                    const under = below.profiles.find((profile) => profile.role === role);
                    const current = refValue(held);
                    const changed = current !== refValue(under);
                    const options: readonly SelectOption<string>[] = [
                        { value: unpinned, label: copy.templates.models.follows },
                        ...(current !== unpinned && !catalogOptions.some((option) => option.value === current)
                            ? [{ value: current, label: `${held?.provider ?? ""} · ${held?.model ?? ""}` }]
                            : []),
                        ...catalogOptions,
                    ];
                    return (
                        <div
                            key={role}
                            className={cx(
                                "grid h-10 grid-cols-[minmax(4rem,7rem)_minmax(0,1fr)_28px] items-center gap-2 border-b border-inset px-3 last:border-b-0",
                                changed && "shadow-[inset_2px_0_0_var(--color-accent)]",
                            )}
                        >
                            <span className="truncate text-xs text-ink">{roleLabel(role)}</span>
                            <Select
                                value={current}
                                options={options}
                                invalid={fieldErrorOf(error, `modelProfiles.${role}`) !== null}
                                aria-label={roleLabel(role)}
                                onValueChange={(next) => {
                                    pin(role, next);
                                }}
                            />
                            {changed ? (
                                <IconButton
                                    icon={RestartAltIcon}
                                    label={copy.templates.overrides.revert}
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                        pin(role, refValue(under));
                                    }}
                                />
                            ) : (
                                <span />
                            )}
                        </div>
                    );
                })}
            </section>
        </div>
    );
}
