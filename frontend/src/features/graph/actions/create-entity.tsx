import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { useAddEdge, useCreateEntity } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import { entityKinds } from "../../../generated/vocab.js";
import { Button, ChipInput, Drawer, Field, Input, Select, Textarea, toneClasses } from "../../../ui/index.js";
import type { SelectOption } from "../../../ui/index.js";
import { entityIcon, kindLabel, kindTone } from "../labels.js";
import type { GraphIndex } from "../model/index.js";
import { fieldErrorOf, formErrorOf } from "../inspector/fields.js";

const kindOptions: readonly SelectOption<string>[] = entityKinds.map((value) => ({
    value,
    label: kindLabel(value),
}));

export interface CreateEntityDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    index: GraphIndex;
    parentId: string | null;
    onCreated: (id: string) => void;
}

export function CreateEntityDrawer({ open, onOpenChange, siteId, index, parentId, onCreated }: CreateEntityDrawerProps): ReactElement {
    const create = useCreateEntity();
    const addEdge = useAddEdge();
    const [name, setName] = useState("");
    const [kind, setKind] = useState<string>(entityKinds[2]);
    const [intent, setIntent] = useState("");
    const [primaryKeyword, setPrimaryKeyword] = useState("");
    const [secondaryKeywords, setSecondaryKeywords] = useState<string[]>([]);
    const [anchors, setAnchors] = useState<string[]>([]);
    const parent = parentId === null ? undefined : index.byId.get(parentId);
    const ParentIcon = entityIcon(parent?.kind ?? "");

    useEffect(() => {
        if (open) {
            setName("");
            setKind(parent === undefined ? entityKinds[0] : entityKinds[2]);
            setIntent("");
            setPrimaryKeyword("");
            setSecondaryKeywords([]);
            setAnchors([]);
            create.reset();
            addEdge.reset();
        }
    }, [open, parentId]);

    const submit = async (): Promise<void> => {
        const created = await create.mutateAsync({
            siteId,
            name: name.trim(),
            kind,
            intent: intent.trim(),
            primaryKeyword: primaryKeyword.trim() === "" ? name.trim().toLowerCase() : primaryKeyword.trim(),
            secondaryKeywords,
            anchors: anchors.map((text) => ({ text, source: "user", weight: 1 })),
        });
        const id = created.entity.id;
        if (parent !== undefined) {
            try {
                await addEdge.mutateAsync({ siteId, fromEntityId: id, toEntityId: parent.id, kind: "parent", weight: 1 });
            } catch {
                pushToast("warning", copy.graph.create.attachFailed(created.entity.name));
            }
        }
        pushToast("info", copy.graph.create.created(created.entity.name));
        onOpenChange(false);
        onCreated(id);
    };

    const formError = formErrorOf(create.error);

    return (
        <Drawer
            open={open}
            onOpenChange={onOpenChange}
            title={parent === undefined ? copy.graph.create.title : copy.graph.create.under(parent.name)}
            closeLabel={copy.graph.create.cancel}
            width={440}
            footer={
                <div className="flex justify-end gap-2">
                    <Button
                        variant="ghost"
                        onClick={() => {
                            onOpenChange(false);
                        }}
                    >
                        {copy.graph.create.cancel}
                    </Button>
                    <Button
                        variant="primary"
                        busy={create.isPending || addEdge.isPending}
                        disabled={name.trim() === ""}
                        onClick={() => {
                            void submit().catch(() => undefined);
                        }}
                    >
                        {copy.graph.create.submit}
                    </Button>
                </div>
            }
        >
            <form
                className="flex flex-col gap-3 p-4"
                onSubmit={(event) => {
                    event.preventDefault();
                    if (name.trim() !== "") {
                        void submit().catch(() => undefined);
                    }
                }}
            >
                <div className="flex items-center gap-2 rounded-md border border-hairline bg-inset px-2.5 py-2 text-xs">
                    <span className="text-ink-faint">{copy.graph.create.parent}</span>
                    {parent === undefined ? (
                        <span className="text-ink-dim">{copy.graph.create.noParent}</span>
                    ) : (
                        <span className="flex min-w-0 items-center gap-1.5 text-ink">
                            <ParentIcon size={14} className={toneClasses[kindTone(parent.kind)].ink} />
                            <span className="truncate">{parent.name}</span>
                        </span>
                    )}
                </div>
                <Field label={copy.graph.form.name} required={true} error={fieldErrorOf(create.error, "name")}>
                    {(control) => (
                        <Input
                            {...control}
                            autoFocus={true}
                            value={name}
                            onChange={(event) => {
                                setName(event.target.value);
                            }}
                        />
                    )}
                </Field>
                <Field label={copy.graph.form.kind} error={fieldErrorOf(create.error, "kind")}>
                    {(control) => (
                        <Select id={control.id} aria-describedby={control["aria-describedby"]} invalid={control.invalid} value={kind} options={kindOptions} onValueChange={setKind} />
                    )}
                </Field>
                <Field label={copy.graph.form.primaryKeyword} error={fieldErrorOf(create.error, "primaryKeyword")}>
                    {(control) => (
                        <Input
                            {...control}
                            mono={true}
                            placeholder={name.trim().toLowerCase()}
                            value={primaryKeyword}
                            onChange={(event) => {
                                setPrimaryKeyword(event.target.value);
                            }}
                        />
                    )}
                </Field>
                <Field label={copy.graph.form.secondaryKeywords} hint={copy.graph.form.keywordsHint} error={fieldErrorOf(create.error, "secondaryKeywords")}>
                    {(control) => (
                        <ChipInput
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            mono={true}
                            values={secondaryKeywords}
                            removeLabel={copy.graph.form.remove}
                            onChange={setSecondaryKeywords}
                        />
                    )}
                </Field>
                <Field label={copy.graph.create.anchors} hint={copy.graph.create.anchorsHint} error={fieldErrorOf(create.error, "anchors")}>
                    {(control) => (
                        <ChipInput
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            mono={true}
                            values={anchors}
                            removeLabel={copy.graph.form.remove}
                            onChange={setAnchors}
                        />
                    )}
                </Field>
                <Field label={copy.graph.form.intent} hint={copy.graph.form.intentHint} error={fieldErrorOf(create.error, "intent")}>
                    {(control) => (
                        <Textarea
                            {...control}
                            rows={2}
                            value={intent}
                            onChange={(event) => {
                                setIntent(event.target.value);
                            }}
                        />
                    )}
                </Field>
                {formError === null ? null : <p className="text-xs text-danger">{formError}</p>}
            </form>
        </Drawer>
    );
}
