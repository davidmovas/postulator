import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../data/errors.js";
import { useUpdateSite } from "../../data/hooks/sites.js";
import type { Site } from "../../data/types.js";
import type { UrlForm } from "../../generated/vocab.js";
import { urlForms } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Banner, Button, Field, Input, SectionLabel, Select } from "../../ui/index.js";
import { urlFormLabel } from "../pages/model/labels.js";
import type { SettingsDraft } from "./settings.js";
import { draftOf, problemsOf, sameDraft, settingsOf } from "./settings.js";

const said = copy.sites.settings;

const urlFormOptions: readonly SelectOption<UrlForm>[] = urlForms.map((value) => ({ value, label: urlFormLabel(value) }));

export interface SiteSettingsFormProps {
    site: Site;
}

export function SiteSettingsForm({ site }: SiteSettingsFormProps): ReactElement {
    const update = useUpdateSite();
    const [draft, setDraft] = useState<SettingsDraft>(() => draftOf(site.settings));

    useEffect(() => {
        setDraft(draftOf(site.settings));
        update.reset();
    }, [site.id, site.updatedAt]);

    const stored = draftOf(site.settings);
    const problems = problemsOf(draft);
    const dirty = !sameDraft(stored, draft);
    const request = settingsOf(site.settings, draft);
    const formError = formErrorOf(update.error);

    const edit = (patch: Partial<SettingsDraft>): void => {
        setDraft((held) => ({ ...held, ...patch }));
    };

    return (
        <section aria-label={said.title} className="flex flex-col gap-2">
            <SectionLabel>{said.title}</SectionLabel>
            <Field
                label={said.userAgent}
                tooltip={said.userAgentTooltip}
                error={fieldErrorOf(update.error, "settings.userAgent")}
            >
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        mono={true}
                        placeholder={said.userAgentDefault}
                        value={draft.userAgent}
                        onChange={(event) => {
                            edit({ userAgent: event.target.value });
                        }}
                    />
                )}
            </Field>
            <div className="grid grid-cols-2 gap-2">
                <Field
                    label={said.concurrency}
                    tooltip={said.concurrencyTooltip}
                    error={problems.renderConcurrency ?? fieldErrorOf(update.error, "settings.observe.renderConcurrency")}
                >
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            mono={true}
                            inputMode="numeric"
                            value={draft.renderConcurrency}
                            onChange={(event) => {
                                edit({ renderConcurrency: event.target.value });
                            }}
                        />
                    )}
                </Field>
                <Field
                    label={said.cadence}
                    tooltip={said.cadenceTooltip}
                    hint={said.cadenceUnit}
                    error={problems.fullWalkEvery ?? fieldErrorOf(update.error, "settings.observe.fullWalkEvery")}
                >
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            mono={true}
                            inputMode="numeric"
                            value={draft.fullWalkEvery}
                            onChange={(event) => {
                                edit({ fullWalkEvery: event.target.value });
                            }}
                        />
                    )}
                </Field>
            </div>
            <Field
                label={said.urlForm}
                tooltip={said.urlFormTooltip}
                error={fieldErrorOf(update.error, "settings.linkUrlForm")}
            >
                {(control) => (
                    <Select
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        value={draft.linkUrlForm}
                        options={urlFormOptions}
                        onValueChange={(next) => {
                            edit({ linkUrlForm: next });
                        }}
                    />
                )}
            </Field>
            {formError === null ? null : <Banner tone="danger" title={formError} />}
            <div className="flex justify-end gap-2">
                <Button
                    variant="ghost"
                    size="sm"
                    disabled={!dirty}
                    onClick={() => {
                        setDraft(stored);
                        update.reset();
                    }}
                >
                    {said.revert}
                </Button>
                <Button
                    variant="primary"
                    size="sm"
                    disabled={!dirty || request === null}
                    busy={update.isPending}
                    onClick={() => {
                        if (request !== null) {
                            update.mutate({ id: site.id, settings: request });
                        }
                    }}
                >
                    {said.save}
                </Button>
            </div>
        </section>
    );
}
