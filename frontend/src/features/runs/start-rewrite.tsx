import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import type { StepSpec } from "../../data/types.js";
import { Checkbox, Dialog, EditNoteIcon, Switch } from "../../ui/index.js";

const said = copy.runs.start;

const generateBody = "generate_body";

export function writesBody(recipe: readonly StepSpec[] | null | undefined): boolean {
    if (recipe === null || recipe === undefined || recipe.length === 0) {
        return true;
    }
    return recipe.some((step) => step.enabled && step.name === generateBody);
}

export function rewriteOf(confirmed: ReadonlySet<string>, chosen: ReadonlySet<string>): string[] {
    return [...chosen].filter((pageId) => confirmed.has(pageId));
}

export interface ChosenPage {
    id: string;
    path: string;
}

export interface RewriteSwitchProps {
    pages: readonly ChosenPage[];
    rewrite: ReadonlySet<string>;
    onChange: (next: ReadonlySet<string>) => void;
}

export function RewriteSwitch({ pages, rewrite, onChange }: RewriteSwitchProps): ReactElement {
    const [asking, setAsking] = useState(false);
    const [ticked, setTicked] = useState<ReadonlySet<string>>(new Set());
    const rewritten = pages.filter((page) => rewrite.has(page.id));
    const on = rewritten.length > 0;

    return (
        <div className="flex flex-col gap-1.5 rounded-md border border-hairline bg-panel px-3 py-2.5" data-run-rewrite={on}>
            <Switch
                tone="danger"
                label={said.rewrite}
                checked={on}
                disabled={pages.length === 0}
                onChange={(event) => {
                    if (event.currentTarget.checked) {
                        setTicked(new Set(pages.map((page) => page.id)));
                        setAsking(true);
                        return;
                    }
                    onChange(new Set());
                }}
            />
            <p className="text-2xs text-ink-faint">
                {on ? said.rewriteOn(rewritten.map((page) => page.path)) : said.rewriteHint}
            </p>
            <Dialog
                open={asking}
                onOpenChange={setAsking}
                title={said.rewriteTitle}
                description={said.rewriteBody}
                confirmLabel={said.rewriteConfirm(ticked.size)}
                cancelLabel={said.rewriteCancel}
                destructive={true}
                icon={EditNoteIcon}
                onConfirm={() => {
                    onChange(new Set(ticked));
                    setAsking(false);
                }}
            >
                <ul className="flex max-h-64 flex-col gap-1.5 overflow-auto rounded-md border border-hairline bg-panel px-3 py-2.5">
                    {pages.map((page) => (
                        <li key={page.id}>
                            <Checkbox
                                label={page.path}
                                checked={ticked.has(page.id)}
                                onChange={(event) => {
                                    const next = new Set(ticked);
                                    if (event.currentTarget.checked) {
                                        next.add(page.id);
                                    } else {
                                        next.delete(page.id);
                                    }
                                    setTicked(next);
                                }}
                            />
                        </li>
                    ))}
                </ul>
            </Dialog>
        </div>
    );
}
