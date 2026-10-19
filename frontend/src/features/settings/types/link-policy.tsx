import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { fieldErrorOf } from "../../../data/errors.js";
import type { LinkPolicy, LinkRules } from "../../../data/types.js";
import { anchorStrategies } from "../../../generated/vocab.js";
import type { SelectOption } from "../../../ui/index.js";
import { Field, SectionLabel, Select, Switch } from "../../../ui/index.js";
import { NumberInput } from "../../templates/controls.js";
import { anchorLabel } from "./labels.js";
import { anchorOf } from "./model.js";

const said = copy.pageTypes.links;

const anchorOptions: readonly SelectOption<string>[] = anchorStrategies.map((value) => ({
    value,
    label: anchorLabel(value),
}));

interface RuleNumber {
    key: "upDepth" | "siblingMinWeight" | "maxLinks" | "maxPerTarget" | "parentLinkWithinParagraphs";
    label: string;
    hint?: string;
    max?: number;
    step?: number;
}

const numbers: readonly RuleNumber[] = [
    { key: "upDepth", label: said.upDepth, hint: said.upDepthHint },
    { key: "siblingMinWeight", label: said.siblingMinWeight, hint: said.siblingMinWeightHint, max: 1, step: 0.05 },
    { key: "maxLinks", label: said.maxLinks, hint: said.maxLinksHint },
    { key: "maxPerTarget", label: said.maxPerTarget },
    { key: "parentLinkWithinParagraphs", label: said.parentLinkWithinParagraphs, hint: said.parentLinkWithinParagraphsHint },
];

export interface LinkPolicyEditorProps {
    policy: LinkPolicy;
    disabled: boolean;
    error: unknown;
    onChange: (policy: LinkPolicy) => void;
}

export function LinkPolicyEditor({ policy, disabled, error, onChange }: LinkPolicyEditorProps): ReactElement {
    const editRules = (patch: Partial<LinkRules>): void => {
        onChange({ ...policy, rules: { ...policy.rules, ...patch } });
    };
    const off = disabled || !policy.links;

    return (
        <section className="flex flex-col gap-2.5 border-t border-hairline pt-3" aria-label={said.title}>
            <SectionLabel>{said.title}</SectionLabel>
            <Switch
                className="w-full"
                label={said.links}
                title={said.linksHint}
                checked={policy.links}
                disabled={disabled}
                data-scope-links={true}
                onChange={(event) => {
                    onChange({ ...policy, links: event.target.checked });
                }}
            />
            <p className="text-2xs text-ink-faint">{said.linksHint}</p>
            <Switch
                className="w-full"
                label={said.forbidExternal}
                checked={policy.forbidExternal}
                disabled={off}
                onChange={(event) => {
                    onChange({ ...policy, forbidExternal: event.target.checked });
                }}
            />
            <Switch
                className="w-full"
                label={said.forbidSelf}
                checked={policy.forbidSelf}
                disabled={off}
                onChange={(event) => {
                    onChange({ ...policy, forbidSelf: event.target.checked });
                }}
            />
            <Field label={said.anchorStrategy} hint={said.anchorStrategyHint} error={fieldErrorOf(error, "links.anchorStrategy")}>
                {(control) => (
                    <Select
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        disabled={off}
                        value={policy.anchorStrategy}
                        options={anchorOptions}
                        onValueChange={(value) => {
                            onChange({ ...policy, anchorStrategy: anchorOf(value, policy.anchorStrategy) });
                        }}
                    />
                )}
            </Field>
            <SectionLabel>{said.rules}</SectionLabel>
            <div className="grid grid-cols-2 gap-2">
                {numbers.map((rule) => (
                    <Field key={rule.key} label={rule.label} hint={rule.hint} error={fieldErrorOf(error, `links.rules.${rule.key}`)}>
                        {(control) => (
                            <NumberInput
                                id={control.id}
                                describedBy={control["aria-describedby"]}
                                invalid={control.invalid}
                                disabled={off}
                                min={0}
                                max={rule.max}
                                step={rule.step}
                                value={policy.rules[rule.key]}
                                onValueChange={(value) => {
                                    const next: LinkRules = { ...policy.rules };
                                    next[rule.key] = value;
                                    onChange({ ...policy, rules: next });
                                }}
                            />
                        )}
                    </Field>
                ))}
            </div>
            <Switch
                className="w-full"
                label={said.downLinks}
                checked={policy.rules.downLinks}
                disabled={off}
                onChange={(event) => {
                    editRules({ downLinks: event.target.checked });
                }}
            />
            <Switch
                className="w-full"
                label={said.childrenSection}
                checked={policy.rules.childrenSection}
                disabled={off}
                onChange={(event) => {
                    editRules({ childrenSection: event.target.checked });
                }}
            />
        </section>
    );
}
