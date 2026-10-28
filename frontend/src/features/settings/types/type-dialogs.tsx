import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../../data/errors.js";
import { useCreatePageType, useDeletePageType, useSavePageType } from "../../../data/hooks/templates.js";
import { pushToast } from "../../../data/toasts.js";
import type { PageType } from "../../../data/types.js";
import { creatablePageTypePresets } from "../../../generated/vocab.js";
import type { SelectOption } from "../../../ui/index.js";
import { DeleteIcon, Dialog, Field, Input, Select } from "../../../ui/index.js";
import { presetHint, presetLabel } from "./labels.js";
import { reassignTargets } from "./model.js";

const said = copy.pageTypes.form;

const noTemplate = "none";

const firstPreset: string = creatablePageTypePresets[0] ?? "";

const presetOptions: readonly SelectOption<string>[] = creatablePageTypePresets.map((value) => ({
    value,
    label: presetLabel(value),
}));

export interface TemplateChoice {
    id: string;
    name: string;
}

export interface TypeFormDialogProps {
    open: boolean;
    siteId: string;
    type: PageType | null;
    templates: readonly TemplateChoice[];
    onOpenChange: (open: boolean) => void;
    onCreated?: (id: string) => void;
}

export function TypeFormDialog({ open, siteId, type, templates, onOpenChange, onCreated }: TypeFormDialogProps): ReactElement {
    const create = useCreatePageType();
    const save = useSavePageType();
    const [name, setName] = useState("");
    const [preset, setPreset] = useState<string>(firstPreset);
    const [templateId, setTemplateId] = useState(noTemplate);
    const [missing, setMissing] = useState(false);

    useEffect(() => {
        if (!open) {
            return;
        }
        create.reset();
        save.reset();
        setMissing(false);
        setName(type?.name ?? "");
        setPreset(type?.preset ?? firstPreset);
        setTemplateId(type?.templateId ?? noTemplate);
    }, [open, type?.id]);

    const editing = type !== null;
    const thrown = editing ? save.error : create.error;
    const chosen = templateId === noTemplate ? "" : templateId;
    const options: readonly SelectOption<string>[] = [
        { value: noTemplate, label: said.noTemplate },
        ...templates.map((held) => ({ value: held.id, label: held.name })),
    ];

    const confirm = (): void => {
        if (name.trim() === "") {
            setMissing(true);
            return;
        }
        if (editing) {
            save.mutate(
                { id: type.id, name: name.trim(), templateId: chosen },
                {
                    onSuccess: () => {
                        onOpenChange(false);
                    },
                },
            );
            return;
        }
        create.mutate(
            { siteId, name: name.trim(), preset, templateId: chosen === "" ? undefined : chosen },
            {
                onSuccess: (answered) => {
                    onOpenChange(false);
                    onCreated?.(answered.id);
                },
            },
        );
    };

    const formError = formErrorOf(thrown);

    return (
        <Dialog
            open={open}
            onOpenChange={onOpenChange}
            title={editing ? said.editTitle : said.createTitle}
            description={editing ? said.editBody : said.createBody}
            confirmLabel={editing ? said.save : said.create}
            cancelLabel={said.cancel}
            busy={create.isPending || save.isPending}
            onConfirm={confirm}
        >
            <div className="flex flex-col gap-3 pt-1">
                <Field
                    label={said.name}
                    hint={said.nameHint}
                    required={true}
                    error={missing ? said.nameMissing : fieldErrorOf(thrown, "name")}
                >
                    {(control) => (
                        <Input
                            {...control}
                            value={name}
                            data-type-name={true}
                            onChange={(event) => {
                                setName(event.target.value);
                                setMissing(false);
                            }}
                        />
                    )}
                </Field>
                {editing ? null : (
                    <Field label={said.preset} hint={presetHint(preset)} error={fieldErrorOf(thrown, "preset")}>
                        {(control) => (
                            <Select
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                value={preset}
                                options={presetOptions}
                                onValueChange={setPreset}
                            />
                        )}
                    </Field>
                )}
                <Field label={said.template} hint={said.templateHint} error={fieldErrorOf(thrown, "templateId")}>
                    {(control) => (
                        <Select
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={templateId}
                            options={options}
                            onValueChange={setTemplateId}
                        />
                    )}
                </Field>
                {formError === null ? null : <p className="text-xs text-danger">{formError}</p>}
            </div>
        </Dialog>
    );
}

export interface DeleteTypeDialogProps {
    open: boolean;
    type: PageType;
    types: readonly PageType[];
    onOpenChange: (open: boolean) => void;
    onDeleted: () => void;
}

export function DeleteTypeDialog({ open, type, types, onOpenChange, onDeleted }: DeleteTypeDialogProps): ReactElement {
    const remove = useDeletePageType();
    const targets = reassignTargets(types, type);
    const [target, setTarget] = useState(targets[0]?.id ?? "");

    useEffect(() => {
        if (open) {
            remove.reset();
            setTarget(reassignTargets(types, type)[0]?.id ?? "");
        }
    }, [open, type.id]);

    const options: readonly SelectOption<string>[] = targets.map((held) => ({
        value: held.id,
        label: `${held.name} · ${presetLabel(held.preset)}`,
    }));
    const said = copy.pageTypes.remove;
    const thrown = remove.error;
    const problem = fieldErrorOf(thrown, "id") ?? formErrorOf(thrown);

    return (
        <Dialog
            open={open}
            onOpenChange={onOpenChange}
            title={said.title(type.name)}
            description={said.body(type.pages)}
            confirmLabel={said.confirm}
            cancelLabel={said.cancel}
            destructive={true}
            icon={DeleteIcon}
            busy={remove.isPending}
            onConfirm={() => {
                remove.mutate(
                    { id: type.id, reassignTo: target },
                    {
                        onSuccess: (answered) => {
                            if (answered.moved > 0) {
                                pushToast("info", said.moved(answered.moved));
                            }
                            onOpenChange(false);
                            onDeleted();
                        },
                    },
                );
            }}
        >
            <div className="flex flex-col gap-2 pt-1">
                <Field label={said.target} error={fieldErrorOf(thrown, "reassignTo")}>
                    {(control) => (
                        <Select
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={target === "" ? null : target}
                            options={options}
                            onValueChange={setTarget}
                        />
                    )}
                </Field>
                {problem === null ? null : <p className="text-xs text-danger">{problem}</p>}
            </div>
        </Dialog>
    );
}
