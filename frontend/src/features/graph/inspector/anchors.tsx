import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { useSetAnchors } from "../../../data/hooks/graph.js";
import type { Anchor, Entity } from "../../../data/types.js";
import { Button, CloseIcon, IconButton, Input, StatusBadge } from "../../../ui/index.js";
import { formErrorOf } from "./fields.js";

function same(left: readonly Anchor[], right: readonly Anchor[]): boolean {
    return (
        left.length === right.length &&
        left.every((anchor, index) => anchor.text === right[index].text && anchor.weight === right[index].weight && anchor.source === right[index].source)
    );
}

export interface AnchorsEditorProps {
    entity: Entity;
}

export function AnchorsEditor({ entity }: AnchorsEditorProps): ReactElement {
    const setAnchors = useSetAnchors();
    const base = entity.anchors ?? [];
    const [anchors, setDraft] = useState<Anchor[]>(() => [...base]);
    const [text, setText] = useState("");
    const [weight, setWeight] = useState("1");
    const [duplicate, setDuplicate] = useState(false);

    useEffect(() => {
        setDraft([...(entity.anchors ?? [])]);
        setAnchors.reset();
    }, [entity.id, entity.updatedAt]);

    const dirty = !same(anchors, base);

    const add = (): void => {
        const trimmed = text.trim();
        if (trimmed === "") {
            return;
        }
        if (anchors.some((held) => held.text.toLowerCase() === trimmed.toLowerCase())) {
            setDuplicate(true);
            return;
        }
        const parsed = Number.parseFloat(weight);
        setDraft([...anchors, { text: trimmed, source: "user", weight: Number.isFinite(parsed) ? Math.min(Math.max(parsed, 0), 1) : 1 }]);
        setText("");
        setDuplicate(false);
    };

    const formError = formErrorOf(setAnchors.error);

    return (
        <div className="flex flex-col gap-2">
            {anchors.length === 0 ? (
                <p className="text-xs text-ink-dim">{copy.graph.inspector.noAnchors}</p>
            ) : (
                <ul className="flex flex-col gap-0.5">
                    {anchors.map((anchor, position) => (
                        <li key={anchor.text} className="flex items-center gap-2 text-xs">
                            <span className="flex-1 truncate font-mono text-ink">{anchor.text}</span>
                            <StatusBadge tone={anchor.source === "user" ? "ok" : "info"} dot={false}>
                                {anchor.source}
                            </StatusBadge>
                            <div className="w-16 shrink-0">
                                <Input
                                    type="number"
                                    size="sm"
                                    min={0}
                                    max={1}
                                    step={0.1}
                                    mono={true}
                                    aria-label={copy.graph.anchorsForm.weight}
                                    value={anchor.weight}
                                    onChange={(event) => {
                                        const parsed = Number.parseFloat(event.target.value);
                                        setDraft(
                                            anchors.map((held, index) =>
                                                index === position ? { ...held, weight: Number.isFinite(parsed) ? Math.min(Math.max(parsed, 0), 1) : held.weight } : held,
                                            ),
                                        );
                                    }}
                                />
                            </div>
                            <IconButton
                                icon={CloseIcon}
                                label={copy.graph.anchorsForm.remove}
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                    setDraft(anchors.filter((_, index) => index !== position));
                                }}
                            />
                        </li>
                    ))}
                </ul>
            )}
            <form
                className="flex items-end gap-2"
                onSubmit={(event) => {
                    event.preventDefault();
                    add();
                }}
            >
                <label className="flex min-w-0 flex-1 flex-col gap-1">
                    <span className="text-2xs text-ink-faint">{copy.graph.anchorsForm.text}</span>
                    <Input
                        size="sm"
                        mono={true}
                        value={text}
                        invalid={duplicate}
                        onChange={(event) => {
                            setText(event.target.value);
                            setDuplicate(false);
                        }}
                    />
                </label>
                <label className="flex w-16 shrink-0 flex-col gap-1">
                    <span className="text-2xs text-ink-faint">{copy.graph.anchorsForm.weight}</span>
                    <Input
                        type="number"
                        size="sm"
                        min={0}
                        max={1}
                        step={0.1}
                        mono={true}
                        value={weight}
                        onChange={(event) => {
                            setWeight(event.target.value);
                        }}
                    />
                </label>
                <Button type="submit" size="sm" variant="secondary" disabled={text.trim() === ""}>
                    {copy.graph.anchorsForm.add}
                </Button>
            </form>
            {duplicate ? <p className="text-xs text-danger">{copy.graph.anchorsForm.duplicate}</p> : null}
            <p className="text-2xs text-ink-faint">{copy.graph.anchorsForm.hint}</p>
            {formError === null ? null : <p className="text-xs text-danger">{formError}</p>}
            {dirty ? (
                <div className="flex justify-end gap-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            setDraft([...base]);
                            setAnchors.reset();
                        }}
                    >
                        {copy.graph.form.discard}
                    </Button>
                    <Button
                        variant="primary"
                        size="sm"
                        busy={setAnchors.isPending}
                        onClick={() => {
                            setAnchors.mutate({ entityId: entity.id, anchors });
                        }}
                    >
                        {copy.graph.anchorsForm.save}
                    </Button>
                </div>
            ) : null}
        </div>
    );
}
