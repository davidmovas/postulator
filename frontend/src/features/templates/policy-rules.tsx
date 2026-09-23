import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { LinkRules } from "../../data/types.js";
import { Field, SectionLabel, Switch } from "../../ui/index.js";
import { fieldErrorOf, NumberInput } from "./controls.js";

export interface PolicyRulesProps {
    rules: LinkRules;
    error: unknown;
    onChange: (patch: Partial<LinkRules>) => void;
}

export function PolicyRules({ rules, error, onChange }: PolicyRulesProps): ReactElement {
    return (
                <div className="flex flex-col gap-2.5 border-t border-hairline pt-3">
                    <span className="w-fit cursor-help" title={copy.policies.rulesTooltip}>
                        <SectionLabel>{copy.policies.rulesTitle}</SectionLabel>
                    </span>
                    <div className="grid grid-cols-2 gap-2">
                        <Field
                            label={copy.templates.links.upDepth}
                            error={fieldErrorOf(error, "linkRules.upDepth")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={rules.upDepth}
                                    onValueChange={(upDepth) => {
                                        onChange({ upDepth });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.siblingMinWeight}
                            error={fieldErrorOf(error, "linkRules.siblingMinWeight")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    max={1}
                                    step={0.05}
                                    value={rules.siblingMinWeight}
                                    onValueChange={(siblingMinWeight) => {
                                        onChange({ siblingMinWeight });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.maxLinks}
                            error={fieldErrorOf(error, "linkRules.maxLinks")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={rules.maxLinks}
                                    onValueChange={(maxLinks) => {
                                        onChange({ maxLinks });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.maxPerTarget}
                            error={fieldErrorOf(error, "linkRules.maxPerTarget")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={rules.maxPerTarget}
                                    onValueChange={(maxPerTarget) => {
                                        onChange({ maxPerTarget });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={copy.templates.links.parentLinkWithinParagraphs}
                            error={fieldErrorOf(error, "linkRules.parentLinkWithinParagraphs")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={rules.parentLinkWithinParagraphs}
                                    onValueChange={(parentLinkWithinParagraphs) => {
                                        onChange({ parentLinkWithinParagraphs });
                                    }}
                                />
                            )}
                        </Field>
                    </div>
                    <Switch
                        className="w-full"
                        label={copy.templates.links.downLinks}
                        checked={rules.downLinks}
                        onChange={(event) => {
                            onChange({ downLinks: event.target.checked });
                        }}
                    />
                    <Switch
                        className="w-full"
                        label={copy.templates.links.childrenSection}
                        checked={rules.childrenSection}
                        onChange={(event) => {
                            onChange({ childrenSection: event.target.checked });
                        }}
                    />
                </div>
    );
}
