import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { AddedPage, Estimate } from "../../data/types.js";
import { tokens as formatTokens, usd as formatUsd } from "../../domain/format.js";
import { Banner, Field, Input, SectionLabel } from "../../ui/index.js";

export interface StartCapsProps {
    cap: string;
    tokenCap: string;
    capValue: number;
    tokenCapValue: number;
    onCap: (next: string) => void;
    onTokenCap: (next: string) => void;
}

export function StartCaps({ cap, tokenCap, capValue, tokenCapValue, onCap, onTokenCap }: StartCapsProps): ReactElement {
    return (
        <div className="grid grid-cols-2 gap-2">
            <Field
                label={copy.runs.start.cap}
                tooltip={copy.runs.start.capHint}
                hint={capValue === 0 ? copy.runs.start.capZero : undefined}
            >
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        mono={true}
                        inputMode="decimal"
                        value={cap}
                        onChange={(event) => {
                            onCap(event.target.value);
                        }}
                    />
                )}
            </Field>
            <Field
                label={copy.runs.start.tokenCap}
                tooltip={copy.runs.start.tokenCapHint}
                hint={tokenCapValue === 0 ? copy.runs.start.tokenCapZero : undefined}
            >
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        data-run-token-cap={true}
                        mono={true}
                        inputMode="numeric"
                        value={tokenCap}
                        onChange={(event) => {
                            onTokenCap(event.target.value);
                        }}
                    />
                )}
            </Field>
        </div>
    );
}

export interface StartPriceProps {
    estimate: Estimate | null;
    added: readonly AddedPage[];
    over: boolean;
    refusal: string | null;
}

export function StartPrice({ estimate, added, over, refusal }: StartPriceProps): ReactElement {
    return (
        <>
            {estimate === null ? null : (
                <div className="flex items-baseline justify-between gap-2 rounded-md border border-hairline bg-inset px-2.5 py-2">
                    <SectionLabel>{copy.runs.start.estimate}</SectionLabel>
                    <span className="font-mono text-sm text-ink">
                        {copy.runs.start.estimated(formatUsd(estimate.usd), formatTokens(estimate.tokens))}
                    </span>
                </div>
            )}

            {added.length === 0 ? null : (
                <Banner
                    tone="info"
                    title={copy.runs.start.addedTitle}
                    body={
                        <ul className="flex flex-col gap-0.5 font-mono text-xs">
                            {added.map((page) => (
                                <li key={page.pageId}>{copy.runs.start.added(page.path, page.neededBy)}</li>
                            ))}
                        </ul>
                    }
                />
            )}

            {(estimate?.findings ?? []).map((finding) => (
                <Banner key={finding.code + finding.message} tone="info" title={finding.message} />
            ))}

            {over ? <Banner tone="warn" title={copy.runs.start.overCap} /> : null}
            {refusal === null ? null : <Banner tone="danger" title={refusal} />}
        </>
    );
}
