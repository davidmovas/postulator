import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { useCreateTemplate } from "../../data/hooks/templates.js";
import type { Template } from "../../data/types.js";
import { copy } from "../../copy/index.js";
import { templateScopes } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { AddIcon, Dialog, Field, Input, Select } from "../../ui/index.js";
import { fieldErrorOf, formErrorOf } from "./controls.js";
import { scopeLabel } from "./labels.js";

const scopeOptions: readonly SelectOption<string>[] = templateScopes.map((value) => ({
    value,
    label: scopeLabel(value),
}));

export interface CreateTemplateDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    templates: readonly Template[];
}

export function CreateTemplateDialog({
    open,
    onOpenChange,
    siteId,
    templates,
}: CreateTemplateDialogProps): ReactElement {
    const navigate = useNavigate();
    const create = useCreateTemplate();
    const [sourceId, setSourceId] = useState("");
    const [name, setName] = useState("");
    const [pageKind, setPageKind] = useState("");
    const [scope, setScope] = useState<string>(templateScopes[0]);

    useEffect(() => {
        if (!open) {
            return;
        }
        const first = templates[0];
        setSourceId(first?.id ?? "");
        setName(first?.name ?? "");
        setPageKind(first?.pageKind ?? "");
        setScope(templateScopes[0]);
        create.reset();
    }, [open, templates]);

    const source = templates.find((template) => template.id === sourceId) ?? null;

    const submit = (): void => {
        if (source === null) {
            return;
        }
        create.mutate(
            {
                scope,
                siteId: scope === "site" ? siteId : null,
                name,
                pageKind,
                spec: source.spec,
            },
            {
                onSuccess: (answered) => {
                    onOpenChange(false);
                    void navigate(`/s/${siteId}/templates/${answered.template.id}`);
                },
            },
        );
    };

    return (
        <Dialog
            open={open}
            onOpenChange={onOpenChange}
            title={copy.templates.create.title}
            description={copy.templates.create.body}
            confirmLabel={copy.templates.create.confirm}
            cancelLabel={copy.templates.create.cancel}
            icon={AddIcon}
            busy={create.isPending}
            onConfirm={submit}
        >
            <div className="mt-1 flex flex-col gap-2.5">
                <Field label={copy.templates.create.copyFrom} hint={copy.templates.create.copyFromHint}>
                    {(control) => (
                        <Select
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            value={sourceId}
                            placeholder={copy.templates.nothingToCopy}
                            disabled={templates.length === 0}
                            options={templates.map((template) => ({
                                value: template.id,
                                label: `${template.name} · ${template.pageKind}`,
                            }))}
                            onValueChange={(next) => {
                                setSourceId(next);
                                const picked = templates.find((template) => template.id === next);
                                setName(picked?.name ?? "");
                                setPageKind(picked?.pageKind ?? "");
                            }}
                        />
                    )}
                </Field>
                <Field
                    label={copy.templates.create.name}
                    required={true}
                    error={fieldErrorOf(create.error, "name")}
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
                <div className="grid grid-cols-2 gap-2">
                    <Field
                        label={copy.templates.create.pageKind}
                        required={true}
                        hint={copy.templates.create.pageKindHint}
                        error={fieldErrorOf(create.error, "pageKind")}
                    >
                        {(control) => (
                            <Input
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                mono={true}
                                value={pageKind}
                                onChange={(event) => {
                                    setPageKind(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    <Field
                        label={copy.templates.create.scope}
                        hint={copy.templates.create.scopeHint}
                        error={fieldErrorOf(create.error, "scope") ?? fieldErrorOf(create.error, "siteId")}
                    >
                        {(control) => (
                            <Select
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                value={scope}
                                options={scopeOptions}
                                onValueChange={setScope}
                            />
                        )}
                    </Field>
                </div>
                {formErrorOf(create.error) === null ? null : (
                    <p className="text-xs text-danger">{formErrorOf(create.error)}</p>
                )}
            </div>
        </Dialog>
    );
}
