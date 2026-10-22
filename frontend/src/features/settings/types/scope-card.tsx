import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../../data/errors.js";
import { useSetScope } from "../../../data/hooks/templates.js";
import { pushToast } from "../../../data/toasts.js";
import type { FieldScope, PageType } from "../../../data/types.js";
import { Banner, Button, ChevronRightIcon, cx, Field, Input, StatusBadge, Switch, Textarea } from "../../../ui/index.js";
import { NumberInput } from "../../templates/controls.js";
import { fieldLabel, metaKeyOf, providerLabel } from "./labels.js";
import { LinkPolicyEditor } from "./link-policy.js";
import type { ScopeDraft } from "./model.js";
import { carriesLinks, draftOf, lengthProblem, requestOf, sameDraft } from "./model.js";

const said = copy.pageTypes.scope;

const bannerFields: readonly string[] = ["field", "enabled", "provider", "pageTypeId"];

export interface ScopeCardProps {
    type: PageType;
    scope: FieldScope;
    aiWritable?: boolean;
}

export function ScopeCard({ type, scope, aiWritable = true }: ScopeCardProps): ReactElement {
    const save = useSetScope();
    const [open, setOpen] = useState(false);
    const [draft, setDraft] = useState<ScopeDraft>(() => draftOf(scope));
    const base = draftOf(scope);
    const readOnly = !type.writable;
    const custom = metaKeyOf(scope.field) !== null;
    const linked = carriesLinks(scope.field, type.preset);
    const label = fieldLabel(scope.field, scope.label);

    const stamp = JSON.stringify(scope);

    useEffect(() => {
        setDraft(draftOf(scope));
    }, [type.id, stamp]);

    const edit = (patch: Partial<ScopeDraft>): void => {
        setDraft((held) => ({ ...held, ...patch }));
    };

    const dirty = !sameDraft(draft, base);
    const problem = lengthProblem(draft);
    const lengthError = problem === null ? null : said[problem];
    const thrown = save.error;
    const banner = formErrorOf(thrown) ?? bannerFields.map((name) => fieldErrorOf(thrown, name)).find((held) => held !== null) ?? null;

    const submit = (): void => {
        if (problem !== null) {
            return;
        }
        save.mutate(requestOf(type.id, scope.field, aiWritable ? draft : { ...draft, aiWrites: false }), {
            onSuccess: () => {
                pushToast("info", said.saved(label));
            },
        });
    };

    return (
        <li className="rounded-md border border-hairline bg-panel" data-scope-field={scope.field}>
            <button
                type="button"
                aria-expanded={open}
                aria-label={open ? said.collapse(label) : said.expand(label)}
                className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-inset"
                onClick={() => {
                    setOpen((held) => !held);
                }}
            >
                <ChevronRightIcon size={14} className={cx("shrink-0 text-ink-dim transition-transform duration-100", open && "rotate-90")} />
                <span className="min-w-0 flex-1 truncate text-sm text-ink">{label}</span>
                {custom ? (
                    <StatusBadge tone="muted" dot={false}>
                        {said.custom}
                    </StatusBadge>
                ) : null}
                {scope.enabled && scope.aiWrites && aiWritable ? (
                    <StatusBadge tone="accent" dot={false}>
                        {said.modelWrites}
                    </StatusBadge>
                ) : null}
                {scope.enabled && scope.rewrites ? (
                    <StatusBadge tone="warn" dot={false}>
                        {said.rewrites}
                    </StatusBadge>
                ) : null}
                {linked && scope.links.links ? (
                    <StatusBadge tone="muted" dot={false}>
                        {said.linked}
                    </StatusBadge>
                ) : null}
                <StatusBadge tone={scope.enabled ? "ok" : "muted"}>{scope.enabled ? said.on : said.off}</StatusBadge>
            </button>
            {open ? (
                <form
                    className="flex flex-col gap-2.5 border-t border-hairline p-3"
                    onSubmit={(event) => {
                        event.preventDefault();
                        if (dirty) {
                            submit();
                        }
                    }}
                >
                    {banner === null ? null : <Banner tone="danger" title={banner} />}
                    <Switch
                        className="w-full"
                        label={said.enabled}
                        checked={draft.enabled}
                        disabled={readOnly}
                        data-scope-enabled={true}
                        onChange={(event) => {
                            edit({ enabled: event.target.checked });
                        }}
                    />
                    <Switch
                        className="w-full"
                        label={said.aiWrites}
                        title={aiWritable ? said.aiWritesHint : said.personOnly}
                        checked={aiWritable && draft.aiWrites}
                        disabled={readOnly || !aiWritable}
                        onChange={(event) => {
                            edit({ aiWrites: event.target.checked });
                        }}
                    />
                    <p className="text-2xs text-ink-faint">{aiWritable ? said.aiWritesHint : said.personOnly}</p>
                    {custom ? (
                        <div className="grid grid-cols-2 gap-2">
                            <Field label={said.label} required={true} error={fieldErrorOf(thrown, "label")}>
                                {(control) => (
                                    <Input
                                        {...control}
                                        disabled={readOnly}
                                        value={draft.label}
                                        onChange={(event) => {
                                            edit({ label: event.target.value });
                                        }}
                                    />
                                )}
                            </Field>
                            <Field label={said.provider}>
                                {() => <p className="py-1.5 text-sm text-ink-soft">{providerLabel(draft.provider)}</p>}
                            </Field>
                        </div>
                    ) : null}
                    <Field label={said.instruction} hint={said.instructionHint} error={fieldErrorOf(thrown, "instruction")}>
                        {(control) => (
                            <Textarea
                                {...control}
                                rows={2}
                                disabled={readOnly}
                                value={draft.instruction}
                                onChange={(event) => {
                                    edit({ instruction: event.target.value });
                                }}
                            />
                        )}
                    </Field>
                    <div className="grid grid-cols-2 gap-2">
                        <Field label={said.minLength} hint={said.lengthHint} error={fieldErrorOf(thrown, "minLength")}>
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid || lengthError !== null}
                                    disabled={readOnly}
                                    min={0}
                                    value={draft.minLength}
                                    onValueChange={(minLength) => {
                                        edit({ minLength });
                                    }}
                                />
                            )}
                        </Field>
                        <Field
                            label={said.maxLength}
                            hint={said.lengthHint}
                            error={lengthError ?? fieldErrorOf(thrown, "maxLength")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    disabled={readOnly}
                                    min={0}
                                    value={draft.maxLength}
                                    onValueChange={(maxLength) => {
                                        edit({ maxLength });
                                    }}
                                />
                            )}
                        </Field>
                    </div>
                    <Switch
                        className="w-full"
                        label={said.rewritesLabel}
                        checked={draft.rewrites}
                        disabled={readOnly}
                        data-scope-rewrites={true}
                        onChange={(event) => {
                            edit({ rewrites: event.target.checked });
                        }}
                    />
                    <p className="text-2xs text-ink-faint">{said.rewritesHint}</p>
                    {linked ? (
                        <LinkPolicyEditor
                            policy={draft.links}
                            disabled={readOnly}
                            error={thrown}
                            onChange={(links) => {
                                edit({ links });
                            }}
                        />
                    ) : null}
                    {readOnly ? null : (
                        <div className="flex justify-end gap-2">
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                disabled={!dirty}
                                onClick={() => {
                                    setDraft(draftOf(scope));
                                    save.reset();
                                }}
                            >
                                {said.revert}
                            </Button>
                            <Button
                                type="submit"
                                variant="primary"
                                size="sm"
                                busy={save.isPending}
                                disabled={!dirty || problem !== null}
                                data-scope-save={true}
                            >
                                {said.save}
                            </Button>
                        </div>
                    )}
                </form>
            ) : null}
        </li>
    );
}
