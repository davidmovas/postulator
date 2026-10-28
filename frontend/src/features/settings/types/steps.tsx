import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { ChevronRightIcon, HubIcon, LockIcon } from "../../../ui/index.js";
import { findingsOfCode } from "../../findings/index.js";

const said = copy.pageTypes.screen;

const withoutEntity = "page_without_entity";

const steps = [said.steps.read, said.steps.rules, said.steps.type] as const;

export function TypeSteps({ siteId }: { siteId: string }): ReactElement {
    return (
        <div className="@container flex flex-col gap-2.5">
            <ol
                aria-label={said.steps.label}
                className="grid rounded-lg border border-hairline bg-panel @2xl:grid-cols-3"
            >
                {steps.map((step, at) => (
                    <li
                        key={step.title}
                        className="relative flex gap-3 border-b border-hairline px-3 py-2.5 last:border-b-0 @2xl:border-r @2xl:border-b-0 @2xl:py-3 @2xl:last:border-r-0"
                    >
                        <span
                            aria-hidden={true}
                            className="mt-px inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-edge bg-inset font-mono text-2xs text-ink-soft"
                        >
                            {at + 1}
                        </span>
                        <span className="flex min-w-0 flex-col gap-0.5">
                            <span data-step-title={true} className="text-sm font-semibold text-ink">
                                {step.title}
                            </span>
                            <span className="text-xs leading-snug text-ink-dim">{step.body}</span>
                        </span>
                        {at === steps.length - 1 ? null : (
                            <span
                                aria-hidden={true}
                                className="absolute top-1/2 -right-2.5 z-[1] hidden h-5 w-5 -translate-y-1/2 items-center justify-center rounded-full border border-hairline bg-canvas text-ink-faint @2xl:inline-flex"
                            >
                                <ChevronRightIcon size={13} />
                            </span>
                        )}
                    </li>
                ))}
            </ol>
            <ul className="flex flex-col gap-1.5 px-1 text-xs leading-snug text-ink-soft">
                <li className="flex items-start gap-2">
                    <LockIcon size={14} className="mt-px shrink-0 text-ink-faint" />
                    <span>{said.byHand}</span>
                </li>
                <li className="flex items-start gap-2">
                    <HubIcon size={14} className="mt-px shrink-0 text-ink-faint" />
                    <span>
                        <span>{said.notGraph}</span>{" "}
                        <Link to={findingsOfCode(siteId, withoutEntity)} className="whitespace-nowrap text-accent hover:underline">
                            {said.withoutEntity}
                        </Link>
                    </span>
                </li>
            </ul>
        </div>
    );
}
