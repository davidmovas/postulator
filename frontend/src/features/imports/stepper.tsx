import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { CheckSmallIcon, cx } from "../../ui/index.js";
import type { ImportStep } from "./params.js";
import { importSteps, reachable, stepIndex } from "./params.js";

export interface StepperProps {
    step: ImportStep;
    path: string;
    locked: boolean;
    onStep: (step: ImportStep) => void;
}

export function Stepper({ step, path, locked, onStep }: StepperProps): ReactElement {
    const current = stepIndex(step);
    return (
        <div className="flex min-w-0 flex-1 items-center gap-1">
            {importSteps.map((named, at) => {
                const open = reachable(named, path) && !locked;
                const done = at < current;
                return (
                    <div key={named} className="flex min-w-0 flex-1 items-center gap-1">
                        <button
                            type="button"
                            disabled={!open}
                            aria-current={at === current ? "step" : undefined}
                            onClick={() => {
                                onStep(named);
                            }}
                            className={cx(
                                "flex h-6 shrink-0 items-center gap-1.5 rounded-md px-1.5 text-xs transition-colors duration-100 ease-out",
                                open ? "hover:bg-raised" : "cursor-not-allowed",
                                at === current ? "font-semibold text-ink" : open ? "text-ink-dim" : "text-ink-faint",
                            )}
                        >
                            <span
                                aria-hidden={true}
                                className={cx(
                                    "flex h-4 w-4 items-center justify-center rounded-full border font-mono text-2xs",
                                    at === current
                                        ? "border-accent-border bg-accent text-on-accent"
                                        : done
                                          ? "border-ok-border text-ok"
                                          : "border-hairline text-ink-faint",
                                )}
                            >
                                {done ? <CheckSmallIcon size={12} /> : at + 1}
                            </span>
                            {copy.imports.steps[named]}
                        </button>
                        {at === importSteps.length - 1 ? null : (
                            <span
                                aria-hidden={true}
                                className={cx("h-px min-w-4 flex-1", done ? "bg-ok-border" : "bg-hairline")}
                            />
                        )}
                    </div>
                );
            })}
        </div>
    );
}
