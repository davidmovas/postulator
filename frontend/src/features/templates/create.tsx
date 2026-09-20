import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { useCreateTemplate } from "../../data/hooks/templates.js";
import type { Template } from "../../data/types.js";
import type { SegmentedOption, SelectOption } from "../../ui/index.js";
import { Banner, Button, Drawer, Field, Input, Segmented, Select } from "../../ui/index.js";
import { blankDraft } from "./blank.js";
import { fieldErrorOf, formErrorOf } from "./controls.js";
import { pageKindLabel, scopeLabel } from "./labels.js";
import { copyName } from "./naming.js";
import { namesIn } from "./rows.js";
import { specOf } from "./spec.js";

type Start = "blank" | "copy";

const startOptions: readonly SegmentedOption<Start>[] = [
    { value: "blank", label: copy.templates.create.startBlank },
    { value: "copy", label: copy.templates.create.startCopy },
];

const scopeOptions: readonly SelectOption<string>[] = [
    { value: "global", label: scopeLabel("global") },
    { value: "site", label: scopeLabel("site") },
];

export interface CreateTemplateDrawerProps {
    open: boolean;
    siteId: string;
    templates: readonly Template[];
    onOpenChange: (open: boolean) => void;
    onCreated: (id: string) => void;
}

export function CreateTemplateDrawer({
    open,
    siteId,
    templates,
    onOpenChange,
    onCreated,
}: CreateTemplateDrawerProps): ReactElement {
    const create = useCreateTemplate();
    const [start, setStart] = useState<Start>("blank");
    const [sourceId, setSourceId] = useState("");
    const [scope, setScope] = useState("site");
    const [name, setName] = useState("");
    const [pageKind, setPageKind] = useState("");

    useEffect(() => {
        if (open) {
            create.reset();
            setStart("blank");
            setSourceId(templates[0]?.id ?? "");
            setScope("site");
            setName("");
            setPageKind("");
        }
    }, [open]);

    const source = templates.find((held) => held.id === sourceId) ?? null;

    const chooseSource = (id: string): void => {
        setSourceId(id);
        const picked = templates.find((held) => held.id === id);
        if (picked === undefined) {
            return;
        }
        setName(copyName(picked.name, namesIn(templates, scope)));
        setPageKind(picked.pageKind);
    };

    const submit = (): void => {
        const spec = start === "copy" && source !== null ? source.spec : specOf(blankDraft());
        create.mutate(
            {
                scope,
                siteId: scope === "site" ? siteId : undefined,
                name: name.trim(),
                pageKind: pageKind.trim(),
                spec,
            },
            {
                onSuccess: (answered) => {
                    onOpenChange(false);
                    onCreated(answered.template.id);
                },
            },
        );
    };

    const blocked = name.trim() === "" || pageKind.trim() === "" || (start === "copy" && source === null);
    const formError = formErrorOf(create.error);

    return (
        <Drawer
            open={open}
            onOpenChange={onOpenChange}
            title={copy.templates.create.title}
            closeLabel={copy.templates.create.cancel}
            width={688}
            footer={
                <>
                    <Button
                        onClick={() => {
                            onOpenChange(false);
                        }}
                    >
                        {copy.templates.create.cancel}
                    </Button>
                    <Button
                        variant="primary"
                        data-template-create={true}
                        disabled={blocked}
                        busy={create.isPending}
                        onClick={submit}
                    >
                        {copy.templates.create.confirm}
                    </Button>
                </>
            }
        >
            <div className="flex max-w-xl flex-col gap-3 p-4">
                {formError === null ? null : <Banner tone="danger" title={formError} />}
                <Field
                    label={copy.templates.create.startFrom}
                    hint={start === "blank" ? copy.templates.create.startBlankHint : copy.templates.create.copyFromHint}
                >
                    {() => (
                        <Segmented
                            label={copy.templates.create.startFrom}
                            value={start}
                            options={startOptions}
                            onValueChange={(next) => {
                                setStart(next);
                                if (next === "copy") {
                                    chooseSource(sourceId === "" ? (templates[0]?.id ?? "") : sourceId);
                                }
                            }}
                        />
                    )}
                </Field>
                {start === "copy" ? (
                    templates.length === 0 ? (
                        <Banner tone="warn" title={copy.templates.create.nothingToCopy} />
                    ) : (
                        <Field label={copy.templates.create.copyFrom}>
                            {(control) => (
                                <Select
                                    id={control.id}
                                    aria-describedby={control["aria-describedby"]}
                                    value={sourceId === "" ? null : sourceId}
                                    placeholder={copy.templates.create.copyFrom}
                                    options={templates.map((held) => ({
                                        value: held.id,
                                        label: `${held.name} · ${pageKindLabel(held.pageKind)} · ${scopeLabel(held.scope)}`,
                                    }))}
                                    onValueChange={chooseSource}
                                />
                            )}
                        </Field>
                    )
                ) : null}
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
                <Field
                    label={copy.templates.create.pageKind}
                    hint={copy.templates.create.pageKindHint}
                    required={true}
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
                    error={fieldErrorOf(create.error, "scope")}
                >
                    {(control) => (
                        <Select
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            value={scope}
                            options={scopeOptions}
                            onValueChange={setScope}
                        />
                    )}
                </Field>
            </div>
        </Drawer>
    );
}
