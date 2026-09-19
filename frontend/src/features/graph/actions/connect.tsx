import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { useAddEdge } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import { Banner, Button, Drawer, Field, Input, toneClasses, WarningIcon } from "../../../ui/index.js";
import { entityIcon, kindTone } from "../labels.js";
import { cycleOf } from "../model/cycle.js";
import type { GraphIndex } from "../model/index.js";
import { formErrorOf } from "../inspector/fields.js";

type Choice = "parentOfFrom" | "parentOfTo" | "related";

export interface ConnectDrawerProps {
    siteId: string;
    index: GraphIndex;
    fromId: string | null;
    toId: string | null;
    onClose: () => void;
}

interface ChoiceRowProps {
    value: Choice;
    current: Choice;
    label: string;
    onPick: (choice: Choice) => void;
}

function ChoiceRow({ value, current, label, onPick }: ChoiceRowProps): ReactElement {
    const active = value === current;
    return (
        <label className={`flex cursor-pointer items-center gap-2 rounded-md border px-2.5 py-2 text-xs ${active ? "border-accent bg-accent-soft text-ink" : "border-hairline text-ink-soft hover:bg-inset"}`}>
            <input
                type="radio"
                name="connect-choice"
                className="accent-accent"
                checked={active}
                onChange={() => {
                    onPick(value);
                }}
            />
            {label}
        </label>
    );
}

export function ConnectDrawer({ siteId, index, fromId, toId, onClose }: ConnectDrawerProps): ReactElement {
    const addEdge = useAddEdge();
    const [choice, setChoice] = useState<Choice>("parentOfFrom");
    const [weight, setWeight] = useState("0.5");
    const from = fromId === null ? undefined : index.byId.get(fromId);
    const to = toId === null ? undefined : index.byId.get(toId);
    const open = from !== undefined && to !== undefined;

    useEffect(() => {
        if (open) {
            setChoice("parentOfFrom");
            setWeight("0.5");
            addEdge.reset();
        }
    }, [open, fromId, toId]);

    const submit = (next: Choice): void => {
        if (from === undefined || to === undefined) {
            return;
        }
        const parsed = Number.parseFloat(weight);
        const request =
            next === "related"
                ? { siteId, fromEntityId: from.id, toEntityId: to.id, kind: "related", weight: Number.isFinite(parsed) ? Math.min(Math.max(parsed, 0), 1) : 0.5 }
                : next === "parentOfFrom"
                  ? { siteId, fromEntityId: from.id, toEntityId: to.id, kind: "parent", weight: 1 }
                  : { siteId, fromEntityId: to.id, toEntityId: from.id, kind: "parent", weight: 1 };
        addEdge.mutate(request, {
            onSuccess: () => {
                pushToast("info", copy.graph.connect.added);
                onClose();
            },
        });
    };

    const cycle = cycleOf(addEdge.error);
    const formError = cycle === null ? formErrorOf(addEdge.error) : null;
    const FromIcon = entityIcon(from?.kind ?? "");
    const ToIcon = entityIcon(to?.kind ?? "");

    return (
        <Drawer
            open={open}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={from === undefined || to === undefined ? copy.graph.connect.start : copy.graph.connect.title(from.name, to.name)}
            closeLabel={copy.graph.connect.close}
            width={440}
            footer={
                <div className="flex justify-end gap-2">
                    <Button variant="ghost" onClick={onClose}>
                        {copy.graph.connect.close}
                    </Button>
                    <Button
                        variant="primary"
                        busy={addEdge.isPending}
                        onClick={() => {
                            submit(choice);
                        }}
                    >
                        {copy.graph.connect.submit}
                    </Button>
                </div>
            }
        >
            {from === undefined || to === undefined ? null : (
                <div className="flex flex-col gap-3 p-4">
                    <div className="flex items-center gap-2 text-sm text-ink">
                        <FromIcon size={16} className={toneClasses[kindTone(from.kind)].ink} />
                        <span className="truncate">{from.name}</span>
                        <span className="text-ink-faint">·</span>
                        <ToIcon size={16} className={toneClasses[kindTone(to.kind)].ink} />
                        <span className="truncate">{to.name}</span>
                    </div>
                    <div className="flex flex-col gap-1.5">
                        <ChoiceRow value="parentOfFrom" current={choice} label={copy.graph.connect.parentOf(from.name, to.name)} onPick={setChoice} />
                        <ChoiceRow value="parentOfTo" current={choice} label={copy.graph.connect.parentOf(to.name, from.name)} onPick={setChoice} />
                        <ChoiceRow value="related" current={choice} label={copy.graph.connect.related} onPick={setChoice} />
                    </div>
                    {choice === "related" ? (
                        <Field label={copy.graph.connect.weight} hint={copy.graph.connect.weightHint}>
                            {(control) => (
                                <Input
                                    {...control}
                                    type="number"
                                    min={0}
                                    max={1}
                                    step={0.05}
                                    mono={true}
                                    className="w-24"
                                    value={weight}
                                    onChange={(event) => {
                                        setWeight(event.target.value);
                                    }}
                                />
                            )}
                        </Field>
                    ) : null}
                    {cycle === null ? null : (
                        <Banner
                            tone="danger"
                            icon={WarningIcon}
                            title={copy.graph.connect.cycle}
                            body={
                                <div className="flex flex-col gap-1">
                                    <p className="font-mono text-2xs">{cycle.map((id) => index.byId.get(id)?.name ?? id).join(" → ")}</p>
                                    <p>{copy.graph.connect.cycleBody}</p>
                                </div>
                            }
                            actions={
                                <Button
                                    size="sm"
                                    variant="secondary"
                                    onClick={() => {
                                        setChoice("related");
                                        submit("related");
                                    }}
                                >
                                    {copy.graph.connect.makeRelated}
                                </Button>
                            }
                        />
                    )}
                    {formError === null ? null : <p className="text-xs text-danger">{formError}</p>}
                </div>
            )}
        </Drawer>
    );
}
