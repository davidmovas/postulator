import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { react } from "../../data/errors.js";
import { useCreatePage } from "../../data/hooks/pages.js";
import { copy } from "../../copy/index.js";
import { pageStatuses, pageWpTypes } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { AddIcon, Dialog, Field, Input, Select } from "../../ui/index.js";
import { ConflictNotice } from "./conflict-notice.js";
import type { EntityIndex } from "./entities.js";

const noEntity = "none";

const wpTypeOptions: readonly SelectOption<string>[] = pageWpTypes.map((value) => ({ value, label: value }));
const statusOptions: readonly SelectOption<string>[] = pageStatuses.map((value) => ({ value, label: value }));

function fieldErrorOf(thrown: unknown, field: string): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    if (reaction.kind === "field" && reaction.field === field) {
        return reaction.message;
    }
    return null;
}

function formErrorOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "form" ? reaction.message : null;
}

export interface PlanPageDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    index: EntityIndex;
    search: string;
    initialEntityId?: string;
}

export function PlanPageDialog({
    open,
    onOpenChange,
    siteId,
    index,
    search,
    initialEntityId,
}: PlanPageDialogProps): ReactElement {
    const navigate = useNavigate();
    const create = useCreatePage();
    const [path, setPath] = useState("");
    const [title, setTitle] = useState("");
    const [wpType, setWpType] = useState<string>(pageWpTypes[0]);
    const [status, setStatus] = useState<string>(pageStatuses[0]);
    const [entityId, setEntityId] = useState(initialEntityId ?? noEntity);

    useEffect(() => {
        if (open) {
            setEntityId(initialEntityId ?? noEntity);
        }
    }, [open, initialEntityId]);

    const reset = (): void => {
        setPath("");
        setTitle("");
        setWpType(pageWpTypes[0]);
        setStatus(pageStatuses[0]);
        setEntityId(initialEntityId ?? noEntity);
        create.reset();
    };

    const entityOptions: SelectOption<string>[] = [
        { value: noEntity, label: copy.pages.create.noEntity },
        ...index.entities.map((entity) => ({ value: entity.id, label: entity.name })),
    ];

    const submit = (): void => {
        create.mutate(
            {
                siteId,
                path,
                title,
                h1: title,
                metaTitle: "",
                metaDescription: "",
                canonical: "",
                wpType,
                status,
                entityId: entityId === noEntity ? null : entityId,
            },
            {
                onSuccess: (answered) => {
                    reset();
                    onOpenChange(false);
                    void navigate(`/s/${siteId}/pages/${answered.page.id}${search}`);
                },
            },
        );
    };

    return (
        <Dialog
            open={open}
            onOpenChange={(next) => {
                if (!next) {
                    reset();
                }
                onOpenChange(next);
            }}
            title={copy.pages.create.title}
            description={copy.pages.create.body}
            confirmLabel={copy.pages.create.confirm}
            cancelLabel={copy.pages.create.cancel}
            busy={create.isPending}
            icon={AddIcon}
            onConfirm={submit}
        >
            <div className="mt-1 flex flex-col gap-2.5">
                <Field
                    label={copy.pages.detail.path}
                    required={true}
                    tooltip={copy.pages.detail.pathHint}
                    error={fieldErrorOf(create.error, "path")}
                >
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            mono={true}
                            value={path}
                            placeholder="/collections/ceramic-mugs/"
                            onChange={(event) => {
                                setPath(event.target.value);
                            }}
                        />
                    )}
                </Field>
                <Field label={copy.pages.detail.title} error={fieldErrorOf(create.error, "title")}>
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={title}
                            onChange={(event) => {
                                setTitle(event.target.value);
                            }}
                        />
                    )}
                </Field>
                <div className="grid grid-cols-2 gap-2">
                    <Field label={copy.pages.detail.wpType} error={fieldErrorOf(create.error, "wpType")}>
                        {(control) => (
                            <Select
                                id={control.id}
                                value={wpType}
                                options={wpTypeOptions}
                                invalid={control.invalid}
                                onValueChange={setWpType}
                            />
                        )}
                    </Field>
                    <Field label={copy.pages.detail.status} error={fieldErrorOf(create.error, "status")}>
                        {(control) => (
                            <Select
                                id={control.id}
                                value={status}
                                options={statusOptions}
                                invalid={control.invalid}
                                onValueChange={setStatus}
                            />
                        )}
                    </Field>
                </div>
                <Field
                    label={copy.pages.detail.entity}
                    tooltip={copy.pages.create.entityHint}
                    error={fieldErrorOf(create.error, "entityId")}
                >
                    {(control) => (
                        <Select
                            id={control.id}
                            value={entityId}
                            options={entityOptions}
                            invalid={control.invalid}
                            disabled={index.entities.length === 0}
                            onValueChange={setEntityId}
                        />
                    )}
                </Field>
                {formErrorOf(create.error) === null ? null : (
                    <p className="text-xs text-danger">{formErrorOf(create.error)}</p>
                )}
                <ConflictNotice thrown={create.error} siteId={siteId} search={search} />
            </div>
        </Dialog>
    );
}
