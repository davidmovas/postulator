import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { usePageTypes } from "../../data/hooks/templates.js";
import { Field, Select } from "../../ui/index.js";
import type { RowCandidate } from "./model/subject.js";
import { PageSearch } from "./page-search.js";

const said = copy.findings.resolve;

export interface TypePickProps {
    siteId: string;
    value: string | null;
    onChange: (pageTypeId: string) => void;
}

export function TypePick({ siteId, value, onChange }: TypePickProps): ReactElement {
    const types = usePageTypes(siteId);
    const options = (types.data?.types ?? []).map((type) => ({ value: type.id, label: type.name }));
    return (
        <Field label={said.typeLabel}>
            {(control) => (
                <Select id={control.id} value={value} options={options} placeholder={said.typeFirst} onValueChange={onChange} />
            )}
        </Field>
    );
}

export interface RowPagePickProps {
    siteId: string;
    candidates: readonly RowCandidate[];
    value: string | null;
    onChange: (pageId: string) => void;
}

export function RowPagePick({ siteId, candidates, value, onChange }: RowPagePickProps): ReactElement {
    if (candidates.length === 0) {
        return (
            <div className="flex flex-col gap-1.5">
                <span className="text-xs font-medium text-ink-dim">{said.pageLabel}</span>
                <PageSearch
                    siteId={siteId}
                    picked={value}
                    onPick={(page) => {
                        onChange(page.pageId);
                    }}
                />
            </div>
        );
    }
    return (
        <div role="radiogroup" aria-label={said.candidates} className="flex flex-col gap-1">
            <span className="text-xs font-medium text-ink-dim">{said.candidates}</span>
            {candidates.map((candidate) => (
                <label key={candidate.pageId} className="flex h-6 items-center gap-2 text-xs text-ink-soft">
                    <input
                        type="radio"
                        name="row-page"
                        checked={value === candidate.pageId}
                        onChange={() => {
                            onChange(candidate.pageId);
                        }}
                    />
                    <span className="truncate font-mono">{candidate.path}</span>
                </label>
            ))}
        </div>
    );
}
