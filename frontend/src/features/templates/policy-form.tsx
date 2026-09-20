import type { ReactElement } from "react";
import { useState } from "react";

import { useCreatePolicy, useUpdatePolicy } from "../../data/hooks/templates.js";
import type { LinkPolicy, LinkRules } from "../../data/types.js";
import { copy } from "../../copy/index.js";
import { anchorStrategies, templateScopes } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Banner, Button, Drawer, Field, Input, SectionLabel, Select, Switch } from "../../ui/index.js";
import { blankDraft } from "./blank.js";
import { fieldErrorOf, formErrorOf, NumberInput } from "./controls.js";
import { anchorLabel, scopeLabel } from "./labels.js";

const scopeOptions: readonly SelectOption<string>[] = templateScopes.map((value) => ({
    value,
    label: scopeLabel(value),
}));

const anchorOptions: readonly SelectOption<string>[] = anchorStrategies.map((value) => ({
    value,
    label: anchorLabel(value),
}));

interface PolicyDraft {
    name: string;
    scope: string;
    forbidExternal: boolean;
    forbidSelf: boolean;
    anchorStrategy: string;
    rules: LinkRules;
}

function blankRules(): LinkRules {
    const spec = blankDraft();
    return {
        upDepth: spec.upDepth,
        downLinks: spec.downLinks,
        siblingMinWeight: spec.siblingMinWeight,
        maxLinks: spec.maxLinks,
        maxPerTarget: spec.maxPerTarget,
        parentLinkWithinParagraphs: spec.parentLinkWithinParagraphs,
        childrenSection: spec.childrenSection,
    };
}

function draftOfPolicy(policy: LinkPolicy | null): PolicyDraft {
    if (policy === null) {
        return {
            name: "",
            scope: templateScopes[0],
            forbidExternal: false,
            forbidSelf: false,
            anchorStrategy: anchorStrategies[0],
            rules: blankRules(),
        };
    }
    return {
        name: policy.name,
        scope: policy.scope,
        forbidExternal: policy.forbidExternal,
        forbidSelf: policy.forbidSelf,
        anchorStrategy: policy.anchorStrategy,
        rules: { ...policy.rules },
    };
}

export interface PolicyDrawerProps {
    siteId: string;
    policy: LinkPolicy | null;
    seed: LinkPolicy | null;
    onClose: () => void;
}

export function PolicyDrawer({ siteId, policy, seed, onClose }: PolicyDrawerProps): ReactElement {
    const create = useCreatePolicy();
    const update = useUpdatePolicy();
    const [draft, setDraft] = useState<PolicyDraft>(() =>
        policy === null
            ? { ...draftOfPolicy(seed), name: "", scope: templateScopes[0] }
            : draftOfPolicy(policy),
    );

    const editing = policy !== null;
    const thrown = editing ? update.error : create.error;
    const busy = create.isPending || update.isPending;

    const edit = (patch: Partial<PolicyDraft>): void => {
        setDraft((held) => ({ ...held, ...patch }));
    };

    const editRules = (patch: Partial<LinkRules>): void => {
        setDraft((held) => ({ ...held, rules: { ...held.rules, ...patch } }));
    };

    const submit = (): void => {
        if (editing) {
            update.mutate(
                {
                    id: policy.id,
                    name: draft.name,
                    rules: draft.rules,
                    forbidExternal: draft.forbidExternal,
                    forbidSelf: draft.forbidSelf,
                    anchorStrategy: draft.anchorStrategy,
                },
                { onSuccess: onClose },
            );
            return;
        }
        create.mutate(
            {
                scope: draft.scope,
                siteId: draft.scope === "site" ? siteId : null,
                name: draft.name,
                rules: draft.rules,
                forbidExternal: draft.forbidExternal,
                forbidSelf: draft.forbidSelf,
                anchorStrategy: draft.anchorStrategy,
            },
            { onSuccess: onClose },
        );
    };

    const formError = formErrorOf(thrown);

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={editing ? copy.policies.create.editTitle : copy.policies.create.title}
            closeLabel={copy.policies.create.cancel}
            width={688}
            footer={
                <>
                    <Button variant="ghost" onClick={onClose}>
                        {copy.policies.create.cancel}
                    </Button>
                    <Button variant="primary" busy={busy} onClick={submit}>
                        {editing ? copy.policies.create.save : copy.policies.create.confirm}
                    </Button>
                </>
            }
        >
            <div className="flex max-w-xl flex-col gap-3 p-4">
                {formError === null ? null : <Banner tone="danger" title={formError} />}
                <Field label={copy.policies.create.name} required={true} error={fieldErrorOf(thrown, "name")}>
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={draft.name}
                            onChange={(event) => {
                                edit({ name: event.target.value });
                            }}
                        />
                    )}
                </Field>
                {editing ? null : (
                    <Field
                        label={copy.policies.create.scope}
                        hint={copy.policies.create.scopeHint}
                        error={fieldErrorOf(thrown, "scope") ?? fieldErrorOf(thrown, "siteId")}
                    >
                        {(control) => (
                            <Select
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                value={draft.scope}
                                options={scopeOptions}
                                onValueChange={(scope) => {
                                    edit({ scope });
                                }}
                            />
                        )}
                    </Field>
                )}
                <Switch
                    className="w-full"
                    label={copy.policies.forbidExternal}
                    checked={draft.forbidExternal}
                    onChange={(event) => {
                        edit({ forbidExternal: event.target.checked });
                    }}
                />
                <Switch
                    className="w-full"
                    label={copy.policies.forbidSelf}
                    checked={draft.forbidSelf}
                    onChange={(event) => {
                        edit({ forbidSelf: event.target.checked });
                    }}
                />
                <Field
                    label={copy.policies.anchorStrategy}
                    hint={copy.policies.anchorStrategyHint}
                    error={fieldErrorOf(thrown, "anchorStrategy")}
                >
                    {(control) => (
                        <Select
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={draft.anchorStrategy}
                            options={anchorOptions}
                            onValueChange={(anchorStrategy) => {
                                edit({ anchorStrategy });
                            }}
                        />
                    )}
                </Field>
                <div className="flex flex-col gap-2.5 border-t border-hairline pt-3">
                    <span className="w-fit cursor-help" title={copy.policies.rulesTooltip}>
                        <SectionLabel>{copy.policies.rulesTitle}</SectionLabel>
                    </span>
                    <div className="grid grid-cols-2 gap-2">
                        <Field
                            label={copy.templates.links.upDepth}
                            error={fieldErrorOf(thrown, "linkRules.upDepth")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.rules.upDepth}
                                    onValueChange={(upDepth) => {
                                        editRules({ upDepth });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.siblingMinWeight}
                            error={fieldErrorOf(thrown, "linkRules.siblingMinWeight")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    max={1}
                                    step={0.05}
                                    value={draft.rules.siblingMinWeight}
                                    onValueChange={(siblingMinWeight) => {
                                        editRules({ siblingMinWeight });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.maxLinks}
                            error={fieldErrorOf(thrown, "linkRules.maxLinks")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.rules.maxLinks}
                                    onValueChange={(maxLinks) => {
                                        editRules({ maxLinks });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.maxPerTarget}
                            error={fieldErrorOf(thrown, "linkRules.maxPerTarget")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.rules.maxPerTarget}
                                    onValueChange={(maxPerTarget) => {
                                        editRules({ maxPerTarget });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.parentLinkWithinParagraphs}
                            error={fieldErrorOf(thrown, "linkRules.parentLinkWithinParagraphs")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.rules.parentLinkWithinParagraphs}
                                    onValueChange={(parentLinkWithinParagraphs) => {
                                        editRules({ parentLinkWithinParagraphs });
                                    }}
                                />
                            )}
                        </Field>
                    </div>
                    <Switch
                        className="w-full"
                        label={copy.templates.links.downLinks}
                        checked={draft.rules.downLinks}
                        onChange={(event) => {
                            editRules({ downLinks: event.target.checked });
                        }}
                    />
                    <Switch
                        className="w-full"
                        label={copy.templates.links.childrenSection}
                        checked={draft.rules.childrenSection}
                        onChange={(event) => {
                            editRules({ childrenSection: event.target.checked });
                        }}
                    />
                </div>
            </div>
        </Drawer>
    );
}
