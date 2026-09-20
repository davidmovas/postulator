import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { cx, IconButton, Input, RestartAltIcon, Select, Switch } from "../../../ui/index.js";
import { fieldErrorOf, NumberInput } from "../controls.js";
import { ruleBlocks } from "../rules-blocks.js";
import type { Rule, RuleBlock } from "../rules-model.js";
import type { SpecDraft } from "../spec.js";

interface RowProps {
    rule: Rule;
    draft: SpecDraft;
    below: SpecDraft;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

function Control({ rule, draft, error, onChange }: Omit<RowProps, "below">): ReactElement {
    const invalid = fieldErrorOf(error, rule.field) !== null;
    switch (rule.kind) {
        case "switch":
            return (
                <Switch
                    checked={rule.read(draft)}
                    aria-label={rule.label}
                    onChange={(event) => {
                        onChange(rule.write(event.target.checked));
                    }}
                />
            );
        case "number":
            return (
                <div className="w-24">
                    <NumberInput
                        value={rule.read(draft)}
                        invalid={invalid}
                        min={rule.min}
                        max={rule.max}
                        step={rule.step}
                        onValueChange={(next) => {
                            onChange(rule.write(next));
                        }}
                    />
                </div>
            );
        case "select":
            return (
                <div className="w-48">
                    <Select
                        value={rule.read(draft)}
                        options={rule.options}
                        invalid={invalid}
                        aria-label={rule.label}
                        onValueChange={(next) => {
                            onChange(rule.write(next));
                        }}
                    />
                </div>
            );
        default:
            return (
                <Input
                    value={rule.read(draft)}
                    invalid={invalid}
                    mono={rule.mono}
                    aria-label={rule.label}
                    onChange={(event) => {
                        onChange(rule.write(event.target.value));
                    }}
                />
            );
    }
}

function Row({ rule, draft, below, error, onChange }: RowProps): ReactElement {
    const changed = rule.read(draft) !== rule.read(below);
    const failure = fieldErrorOf(error, rule.field);
    const wide = rule.kind === "text";
    return (
        <div
            className={cx(
                "flex flex-col gap-1 border-b border-inset px-3 py-2 last:border-b-0",
                changed && "shadow-[inset_2px_0_0_var(--color-accent)]",
            )}
        >
            <div className={cx("flex min-w-0 gap-3", wide ? "flex-col" : "items-center")}>
                <span
                    className={cx("min-w-0 truncate text-xs text-ink-soft", !wide && "flex-1")}
                    title={rule.hint}
                >
                    {rule.label}
                </span>
                <Control rule={rule} draft={draft} error={error} onChange={onChange} />
                {changed ? (
                    <IconButton
                        icon={RestartAltIcon}
                        label={copy.templates.overrides.revert}
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            onChange(revertRule(rule, below));
                        }}
                    />
                ) : null}
            </div>
            {failure === null ? null : <p className="text-2xs text-danger">{failure}</p>}
        </div>
    );
}

function revertRule(rule: Rule, below: SpecDraft): Partial<SpecDraft> {
    switch (rule.kind) {
        case "switch":
            return rule.write(rule.read(below));
        case "number":
            return rule.write(rule.read(below));
        case "text":
            return rule.write(rule.read(below));
        default:
            return rule.write(rule.read(below));
    }
}

interface BlockProps {
    block: RuleBlock;
    draft: SpecDraft;
    below: SpecDraft;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

function Block({ block, draft, below, error, onChange }: BlockProps): ReactElement {
    return (
        <section className="overflow-hidden rounded-lg border border-hairline bg-panel">
            <header className="flex h-7 items-center border-b border-hairline bg-inset px-3">
                <h3 className="text-2xs font-semibold tracking-label text-ink-faint uppercase">{block.title}</h3>
            </header>
            {block.rules.map((rule) => (
                <Row
                    key={rule.key}
                    rule={rule}
                    draft={draft}
                    below={below}
                    error={error}
                    onChange={onChange}
                />
            ))}
        </section>
    );
}

export interface RulesGroupProps {
    draft: SpecDraft;
    below: SpecDraft;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function RulesGroup({ draft, below, error, onChange }: RulesGroupProps): ReactElement {
    return (
        <div className="flex flex-col gap-4">
            {ruleBlocks.map((block) => (
                <Block
                    key={block.key}
                    block={block}
                    draft={draft}
                    below={below}
                    error={error}
                    onChange={onChange}
                />
            ))}
        </div>
    );
}
