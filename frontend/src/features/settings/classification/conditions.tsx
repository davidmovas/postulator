import type { ReactElement, ReactNode } from "react";
import { useId } from "react";

import { copy } from "../../../copy/index.js";
import type { PageType } from "../../../data/types.js";
import type { SchemaType, URLMode } from "../../../generated/vocab.js";
import { schemaTypes, urlModes } from "../../../generated/vocab.js";
import type { SegmentedOption, SelectOption } from "../../../ui/index.js";
import { AddIcon, Button, Checkbox, ChipInput, CloseIcon, IconButton, Input, Segmented, Select } from "../../../ui/index.js";
import { PageSearch } from "../../findings/page-search.js";
import { conditionLabel, postTypeOption, schemaOption, urlModeHint, urlModeLabel } from "./labels.js";
import type { ConditionsDraft, FieldPlace, ListingKind, TriState, UnderKind } from "./model.js";
import { knownPostTypes } from "./model.js";

const said = copy.classification.drawer;

const triOptions: readonly SegmentedOption<TriState>[] = [
    { value: "any", label: said.any },
    { value: "yes", label: said.yes },
    { value: "no", label: said.no },
];

const underOptions: readonly SegmentedOption<UnderKind>[] = [
    { value: "none", label: said.underNone },
    { value: "page", label: said.underPage },
    { value: "type", label: said.underType },
];

const listingOptions: readonly SegmentedOption<ListingKind>[] = [
    { value: "any", label: said.any },
    { value: "present", label: said.listingPresent },
    { value: "absent", label: said.listingAbsent },
];

const modeOptions: readonly SelectOption<URLMode>[] = urlModes.map((mode) => ({ value: mode, label: urlModeLabel(mode) }));

interface RowProps {
    condition: string;
    problem: string | null;
    children: (labelId: string) => ReactNode;
}

function Row({ condition, problem, children }: RowProps): ReactElement {
    const labelId = useId();
    return (
        <div data-condition={condition} className="grid grid-cols-[8.5rem_minmax(0,1fr)] items-start gap-x-3 gap-y-1 border-b border-inset py-2.5 last:border-b-0">
            <span id={labelId} className="pt-1 text-xs font-medium text-ink-soft">
                {conditionLabel(condition)}
            </span>
            <div className="flex min-w-0 flex-col gap-1.5">
                {children(labelId)}
                {problem === null ? null : <p className="text-xs text-danger">{problem}</p>}
            </div>
        </div>
    );
}

interface NumberBoxProps {
    label: string;
    value: string;
    onChange: (value: string) => void;
}

function NumberBox({ label, value, onChange }: NumberBoxProps): ReactElement {
    return (
        <label className="flex items-center gap-2 text-xs text-ink-dim">
            <span>{label}</span>
            <span className="w-20">
                <Input
                    type="number"
                    min={0}
                    size="sm"
                    mono={true}
                    aria-label={label}
                    value={value}
                    onChange={(event) => {
                        onChange(event.target.value);
                    }}
                />
            </span>
        </label>
    );
}

export interface ConditionEditorsProps {
    siteId: string;
    draft: ConditionsDraft;
    types: readonly PageType[];
    underPath: string | null;
    problems: (place: FieldPlace) => string | null;
    onChange: (patch: Partial<ConditionsDraft>) => void;
}

export function ConditionEditors({ siteId, draft, types, underPath, problems, onChange }: ConditionEditorsProps): ReactElement {
    const problemOf = (condition: string, index: number | null = null): string | null => problems({ condition, index });
    const others = draft.postTypes.filter((held) => !knownPostTypes.includes(held));
    const typeOptions: SelectOption<string>[] = types.map((type) => ({ value: type.id, label: type.name }));

    const setUrl = (at: number, patch: Partial<{ mode: URLMode; value: string }>): void => {
        onChange({ urls: draft.urls.map((url, index) => (index === at ? { ...url, ...patch } : url)) });
    };

    return (
        <div className="flex flex-col">
            <Row condition="postType" problem={problemOf("postTypes")}>
                {(labelId) => (
                    <div role="group" aria-labelledby={labelId} className="flex flex-col gap-1.5">
                        <div className="flex flex-wrap gap-x-4 gap-y-1">
                            {knownPostTypes.map((postType) => (
                                <Checkbox
                                    key={postType}
                                    label={postTypeOption(postType)}
                                    checked={draft.postTypes.includes(postType)}
                                    onChange={(event) => {
                                        onChange({
                                            postTypes: event.target.checked
                                                ? [...draft.postTypes, postType]
                                                : draft.postTypes.filter((held) => held !== postType),
                                        });
                                    }}
                                />
                            ))}
                        </div>
                        <ChipInput
                            values={others}
                            mono={true}
                            placeholder={said.postTypesOther}
                            removeLabel={said.removeChip}
                            onChange={(next) => {
                                onChange({ postTypes: [...draft.postTypes.filter((held) => knownPostTypes.includes(held)), ...next] });
                            }}
                        />
                        <p className="text-2xs text-ink-faint">{said.postTypesOtherHint}</p>
                    </div>
                )}
            </Row>

            <Row condition="url" problem={problemOf("urls")}>
                {(labelId) => (
                    <div role="group" aria-labelledby={labelId} className="flex flex-col gap-2">
                        {draft.urls.map((url, at) => (
                            <div key={at} data-url-row={at} className="flex flex-col gap-1">
                                <div className="flex items-center gap-1.5">
                                    <div className="w-40 shrink-0">
                                        <Select
                                            size="sm"
                                            aria-label={`${said.urlMode} ${String(at + 1)}`}
                                            value={url.mode}
                                            options={modeOptions}
                                            onValueChange={(mode) => {
                                                setUrl(at, { mode });
                                            }}
                                        />
                                    </div>
                                    <Input
                                        size="sm"
                                        mono={true}
                                        aria-label={`${said.urlValue} ${String(at + 1)}`}
                                        value={url.value}
                                        invalid={problemOf("urls", at) !== null}
                                        onChange={(event) => {
                                            setUrl(at, { value: event.target.value });
                                        }}
                                    />
                                    <IconButton
                                        size="sm"
                                        variant="ghost"
                                        icon={CloseIcon}
                                        label={said.removeUrl(url.value)}
                                        onClick={() => {
                                            onChange({ urls: draft.urls.filter((_, index) => index !== at) });
                                        }}
                                    />
                                </div>
                                <p className="text-2xs text-ink-faint">{urlModeHint(url.mode)}</p>
                                {problemOf("urls", at) === null ? null : <p className="text-xs text-danger">{problemOf("urls", at)}</p>}
                            </div>
                        ))}
                        <Button
                            size="sm"
                            variant="ghost"
                            icon={AddIcon}
                            className="self-start"
                            onClick={() => {
                                onChange({ urls: [...draft.urls, { mode: "prefix", value: "" }] });
                            }}
                        >
                            {said.addUrl}
                        </Button>
                    </div>
                )}
            </Row>

            <Row condition="slug" problem={problemOf("slugs")}>
                {() => (
                    <>
                        <ChipInput
                            values={draft.slugs}
                            mono={true}
                            placeholder={conditionLabel("slug")}
                            removeLabel={said.removeChip}
                            onChange={(slugs) => {
                                onChange({ slugs });
                            }}
                        />
                        <p className="text-2xs text-ink-faint">{said.slugsHint}</p>
                    </>
                )}
            </Row>

            <Row condition="ancestor" problem={problemOf("ancestor")}>
                {() => (
                    <>
                        <Segmented
                            label={conditionLabel("ancestor")}
                            size="sm"
                            value={draft.under}
                            options={underOptions}
                            onValueChange={(under) => {
                                onChange({ under });
                            }}
                        />
                        {draft.under === "page" ? (
                            <div className="flex flex-col gap-1">
                                {underPath === null ? null : <span className="font-mono text-xs text-ink-soft">{underPath}</span>}
                                <PageSearch
                                    siteId={siteId}
                                    picked={draft.underPage === "" ? null : draft.underPage}
                                    onPick={(page) => {
                                        onChange({ underPage: page.pageId });
                                    }}
                                />
                            </div>
                        ) : null}
                        {draft.under === "type" ? (
                            <div className="w-56">
                                <Select
                                    size="sm"
                                    aria-label={said.underTypePick}
                                    placeholder={said.underTypePick}
                                    value={draft.underType === "" ? null : draft.underType}
                                    options={typeOptions}
                                    onValueChange={(underType) => {
                                        onChange({ underType });
                                    }}
                                />
                            </div>
                        ) : null}
                    </>
                )}
            </Row>

            <Row condition="level" problem={problemOf("level")}>
                {() => (
                    <>
                        <div className="flex flex-wrap gap-4">
                            <NumberBox label={said.levelMin} value={draft.levelMin} onChange={(levelMin) => onChange({ levelMin })} />
                            <NumberBox label={said.levelMax} value={draft.levelMax} onChange={(levelMax) => onChange({ levelMax })} />
                        </div>
                        <p className="text-2xs text-ink-faint">{said.levelHint}</p>
                    </>
                )}
            </Row>

            <Row condition="listing" problem={problemOf("listing")}>
                {() => (
                    <>
                        <Segmented
                            label={conditionLabel("listing")}
                            size="sm"
                            value={draft.listing}
                            options={listingOptions}
                            onValueChange={(listing) => {
                                onChange({ listing });
                            }}
                        />
                        {draft.listing === "present" ? (
                            <NumberBox label={said.minProducts} value={draft.minProducts} onChange={(minProducts) => onChange({ minProducts })} />
                        ) : null}
                    </>
                )}
            </Row>

            <Row condition="schema" problem={problemOf("schema")}>
                {(labelId) => (
                    <div role="group" aria-labelledby={labelId} className="grid grid-cols-2 gap-x-4 gap-y-1">
                        {schemaTypes.map((schema: SchemaType) => (
                            <Checkbox
                                key={schema}
                                label={schemaOption(schema)}
                                checked={draft.schema.includes(schema)}
                                onChange={(event) => {
                                    onChange({
                                        schema: event.target.checked ? [...draft.schema, schema] : draft.schema.filter((held) => held !== schema),
                                    });
                                }}
                            />
                        ))}
                    </div>
                )}
            </Row>

            <Row condition="words" problem={problemOf("words")}>
                {() => (
                    <>
                        <div className="flex flex-wrap gap-4">
                            <NumberBox label={said.wordsMin} value={draft.wordsMin} onChange={(wordsMin) => onChange({ wordsMin })} />
                            <NumberBox label={said.wordsMax} value={draft.wordsMax} onChange={(wordsMax) => onChange({ wordsMax })} />
                        </div>
                        <p className="text-2xs text-ink-faint">{said.wordsHint}</p>
                    </>
                )}
            </Row>

            <Row condition="h2" problem={problemOf("minH2")}>
                {() => <NumberBox label={said.minH2} value={draft.minH2} onChange={(minH2) => onChange({ minH2 })} />}
            </Row>

            {(["technical", "frontPage", "shop"] as const).map((flag) => (
                <Row key={flag} condition={flag} problem={problemOf(flag)}>
                    {() => (
                        <Segmented
                            label={conditionLabel(flag)}
                            size="sm"
                            value={draft[flag]}
                            options={triOptions}
                            onValueChange={(value) => {
                                onChange({ [flag]: value });
                            }}
                        />
                    )}
                </Row>
            ))}
        </div>
    );
}
