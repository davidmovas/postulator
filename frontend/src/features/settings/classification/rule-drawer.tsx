import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { failure, fieldOf, formErrorOf } from "../../../data/errors.js";
import { useDeleteRule, useSaveRule } from "../../../data/hooks/classification.js";
import { usePagePaths } from "../../../data/hooks/pages.js";
import { pushToast } from "../../../data/toasts.js";
import type { PageType, Rule, SaveRuleRequest } from "../../../data/types.js";
import type { SelectOption } from "../../../ui/index.js";
import { Button, DeleteIcon, Dialog, Drawer, Field, Input, SectionLabel, Select, Switch } from "../../../ui/index.js";
import { ConditionEditors } from "./conditions.js";
import type { ConditionsDraft, FieldPlace } from "./model.js";
import { asksAnything, conditionsOf, draftOf, placeOf } from "./model.js";
import { RuleTester } from "./tester.js";

const said = copy.classification.drawer;

type Local = "name" | "type" | "empty";

export interface RuleDrawerProps {
    siteId: string;
    rule: Rule | null;
    types: readonly PageType[];
    typeName: (typeId: string) => string;
    ruleName: (ruleId: string) => string;
    onClose: () => void;
}

export function RuleDrawer({ siteId, rule, types, typeName, ruleName, onClose }: RuleDrawerProps): ReactElement {
    const [name, setName] = useState(rule?.name ?? "");
    const [typeId, setTypeId] = useState(rule?.typeId ?? "");
    const [enabled, setEnabled] = useState(rule?.enabled ?? true);
    const [draft, setDraft] = useState<ConditionsDraft>(() => draftOf(rule?.conditions));
    const [local, setLocal] = useState<Local | null>(null);
    const [removing, setRemoving] = useState(false);
    const save = useSaveRule();
    const remove = useDeleteRule();
    const underIds = useMemo(() => (draft.under === "page" && draft.underPage !== "" ? [draft.underPage] : []), [draft.under, draft.underPage]);
    const underPaths = usePagePaths(underIds, underIds.length > 0);

    const request: SaveRuleRequest = useMemo(() => {
        const held: SaveRuleRequest = { siteId, name: name.trim(), enabled, typeId, conditions: conditionsOf(draft) };
        return rule === null ? held : { ...held, id: rule.id };
    }, [siteId, name, enabled, typeId, draft, rule]);

    const thrown = save.error;
    const field = thrown === null ? null : fieldOf(failure(thrown));
    const place = placeOf(field);
    const message = thrown === null ? null : failure(thrown).message;
    const problems = (asked: FieldPlace): string | null =>
        place !== null && place.condition === asked.condition && place.index === asked.index ? message : null;
    const typeOptions: SelectOption<string>[] = types.map((type) => ({ value: type.id, label: type.name }));
    const shownName = request.name === "" ? copy.classification.unnamed : request.name;

    const submit = (): void => {
        if (request.name === "") {
            setLocal("name");
            return;
        }
        if (request.typeId === "") {
            setLocal("type");
            return;
        }
        if (!asksAnything(request.conditions)) {
            setLocal("empty");
            return;
        }
        setLocal(null);
        save.mutate(request, {
            onSuccess: () => {
                pushToast("info", rule === null ? said.created(shownName) : said.saved(shownName));
                onClose();
            },
        });
    };

    const patch = (next: Partial<ConditionsDraft>): void => {
        setDraft((held) => ({ ...held, ...next }));
        if (local === "empty") {
            setLocal(null);
        }
    };

    const formProblem = place === null && field !== "name" && field !== "typeId" ? formErrorOf(thrown) : null;

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={rule === null ? said.createTitle : said.editTitle(rule.number)}
            description={said.description}
            closeLabel={said.close}
            width={680}
            footer={
                <>
                    {rule === null ? (
                        <span className="mr-auto" />
                    ) : (
                        <Button
                            variant="ghost"
                            icon={DeleteIcon}
                            className="mr-auto"
                            onClick={() => {
                                setRemoving(true);
                            }}
                        >
                            {said.remove}
                        </Button>
                    )}
                    {formProblem === null ? null : <span className="min-w-0 truncate text-xs text-danger">{formProblem}</span>}
                    <Button variant="secondary" onClick={onClose}>
                        {said.cancel}
                    </Button>
                    <Button variant="primary" busy={save.isPending} disabled={save.isPending} onClick={submit}>
                        {rule === null ? said.create : said.save}
                    </Button>
                </>
            }
        >
            <div className="flex flex-col gap-4 p-4" data-rule-drawer={rule?.id ?? "new"}>
                <div className="grid grid-cols-[minmax(0,1fr)_14rem] gap-3">
                    <Field
                        label={said.name}
                        hint={said.nameHint}
                        error={local === "name" ? said.nameMissing : field === "name" ? message : null}
                    >
                        {(control) => (
                            <Input
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                value={name}
                                onChange={(event) => {
                                    setName(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    <Field label={said.type} error={local === "type" ? said.noType : field === "typeId" ? message : null}>
                        {(control) => (
                            <Select
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                placeholder={said.noType}
                                value={typeId === "" ? null : typeId}
                                options={typeOptions}
                                onValueChange={setTypeId}
                            />
                        )}
                    </Field>
                </div>
                <div className="flex flex-col gap-0.5">
                    <Switch
                        label={said.enabled}
                        checked={enabled}
                        onChange={(event) => {
                            setEnabled(event.target.checked);
                        }}
                    />
                    <p className="text-2xs text-ink-faint">{said.enabledHint}</p>
                </div>
                <section aria-label={said.conditions} className="flex flex-col gap-1">
                    <SectionLabel>{said.conditions}</SectionLabel>
                    <p className="text-2xs text-ink-faint">{said.blank}</p>
                    {local === "empty" ? <p className="text-xs text-danger">{said.empty}</p> : null}
                    <ConditionEditors
                        siteId={siteId}
                        draft={draft}
                        types={types}
                        underPath={underIds.length === 0 ? null : (underPaths.get(draft.underPage) ?? null)}
                        problems={problems}
                        onChange={patch}
                    />
                </section>
                <RuleTester siteId={siteId} rule={request} typeName={typeName} ruleName={ruleName} />
            </div>
            {rule === null ? null : (
                <Dialog
                    open={removing}
                    onOpenChange={setRemoving}
                    title={said.removeTitle(rule.name === "" ? copy.classification.unnamed : rule.name)}
                    description={said.removeBody}
                    confirmLabel={said.removeConfirm}
                    cancelLabel={said.cancel}
                    destructive={true}
                    icon={DeleteIcon}
                    busy={remove.isPending}
                    onConfirm={() => {
                        remove.mutate(
                            { id: rule.id },
                            {
                                onSuccess: () => {
                                    pushToast("info", said.removed(rule.name === "" ? copy.classification.unnamed : rule.name));
                                    setRemoving(false);
                                    onClose();
                                },
                            },
                        );
                    }}
                >
                    {remove.error === null ? null : <p className="text-xs text-danger">{failure(remove.error).message}</p>}
                </Dialog>
            )}
        </Drawer>
    );
}
