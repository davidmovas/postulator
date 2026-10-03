import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { fieldErrorOf } from "../../../data/errors.js";
import {
    AddIcon,
    Button,
    CategoryIcon,
    cx,
    DeleteIcon,
    EmptyState,
    IconButton,
    Input,
    RestartAltIcon,
    Switch,
    Textarea,
} from "../../../ui/index.js";
import { NumberInput } from "../controls.js";
import type { ProductDraft, SpecDraft, SpecificationDraft } from "../spec.js";
import { emptyProduct } from "../spec.js";

const said = copy.templates.product;

interface AttributeRowProps {
    row: SpecificationDraft;
    index: number;
    error: unknown;
    onChange: (patch: Partial<SpecificationDraft>) => void;
    onRemove: () => void;
}

function AttributeRow({ row, index, error, onChange, onRemove }: AttributeRowProps): ReactElement {
    const prefix = `product.specifications[${index}]`;
    const nameError = fieldErrorOf(error, `${prefix}.name`);
    return (
        <div className="flex flex-col gap-1.5 rounded-md border border-hairline p-2">
            <div className="flex items-center gap-2">
                <div className="min-w-0 flex-1">
                    <Input
                        value={row.name}
                        invalid={nameError !== null}
                        aria-label={said.attributeName}
                        placeholder={said.attributeName}
                        onChange={(event) => {
                            onChange({ name: event.target.value });
                        }}
                    />
                </div>
                <IconButton icon={DeleteIcon} label={said.removeAttribute} variant="ghost" size="sm" onClick={onRemove} />
            </div>
            {nameError === null ? null : <p className="text-2xs text-danger">{nameError}</p>}
            <Textarea
                rows={2}
                value={row.intent}
                aria-label={said.attributeIntent}
                placeholder={said.attributeIntent}
                invalid={fieldErrorOf(error, `${prefix}.intent`) !== null}
                onChange={(event) => {
                    onChange({ intent: event.target.value });
                }}
            />
        </div>
    );
}

export interface ProductGroupProps {
    draft: SpecDraft;
    below: SpecDraft;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function ProductGroup({ draft, below, error, onChange }: ProductGroupProps): ReactElement {
    const product = draft.product;
    const changed = JSON.stringify(product) !== JSON.stringify(below.product);

    const revert = changed ? (
        <Button
            icon={RestartAltIcon}
            onClick={() => {
                onChange({
                    product:
                        below.product === null
                            ? null
                            : { ...below.product, specifications: below.product.specifications.map((row) => ({ ...row })) },
                });
            }}
        >
            {copy.templates.overrides.revert}
        </Button>
    ) : null;

    if (product === null) {
        return (
            <EmptyState
                icon={CategoryIcon}
                title={said.none}
                body={said.emptyBody}
                actions={
                    <>
                        <Button
                            variant="primary"
                            icon={AddIcon}
                            onClick={() => {
                                onChange({ product: emptyProduct() });
                            }}
                        >
                            {said.add}
                        </Button>
                        {revert}
                    </>
                }
            />
        );
    }

    const update = (patch: Partial<ProductDraft>): void => {
        onChange({ product: { ...product, ...patch } });
    };
    const replace = (index: number, patch: Partial<SpecificationDraft>): void => {
        update({ specifications: product.specifications.map((row, at) => (at === index ? { ...row, ...patch } : row)) });
    };

    return (
        <div className={cx("flex flex-col gap-3", changed && "pl-2 shadow-[inset_2px_0_0_var(--color-accent)]")}>
            <p className="text-xs text-ink-muted">{said.body}</p>

            <section className="flex flex-col gap-2 rounded-lg border border-hairline bg-panel p-2.5">
                <h3 className="text-xs font-semibold text-ink">{said.short}</h3>
                <Switch
                    label={said.shortEnabled}
                    checked={product.shortEnabled}
                    onChange={(event) => {
                        update({ shortEnabled: event.target.checked });
                    }}
                />
                {product.shortEnabled ? (
                    <>
                        <Textarea
                            rows={2}
                            value={product.shortIntent}
                            aria-label={said.shortIntent}
                            placeholder={said.shortIntent}
                            invalid={fieldErrorOf(error, "product.shortDescription.intent") !== null}
                            onChange={(event) => {
                                update({ shortIntent: event.target.value });
                            }}
                        />
                        <div className="flex items-center gap-2">
                            <span className="shrink-0 text-2xs text-ink-faint">{said.shortWords}</span>
                            <div className="w-20 shrink-0">
                                <NumberInput
                                    value={product.shortWords}
                                    min={0}
                                    invalid={fieldErrorOf(error, "product.shortDescription.targetWords") !== null}
                                    onValueChange={(shortWords) => {
                                        update({ shortWords });
                                    }}
                                />
                            </div>
                        </div>
                        <Switch
                            label={said.shortKeyword}
                            checked={product.shortKeyword}
                            onChange={(event) => {
                                update({ shortKeyword: event.target.checked });
                            }}
                        />
                    </>
                ) : null}
            </section>

            <section className="flex flex-col gap-2 rounded-lg border border-hairline bg-panel p-2.5">
                <h3 className="text-xs font-semibold text-ink">{said.attributes}</h3>
                <p className="text-2xs text-ink-faint">{said.attributesHint}</p>
                {product.specifications.length === 0 ? (
                    <p className="text-2xs text-ink-faint">{said.attributesEmpty}</p>
                ) : (
                    product.specifications.map((row, index) => (
                        <AttributeRow
                            key={index}
                            row={row}
                            index={index}
                            error={error}
                            onChange={(patch) => {
                                replace(index, patch);
                            }}
                            onRemove={() => {
                                update({ specifications: product.specifications.filter((_held, at) => at !== index) });
                            }}
                        />
                    ))
                )}
                <div>
                    <Button
                        size="sm"
                        icon={AddIcon}
                        onClick={() => {
                            update({ specifications: [...product.specifications, { name: "", intent: "" }] });
                        }}
                    >
                        {said.addAttribute}
                    </Button>
                </div>
            </section>

            <div className="flex gap-2">
                <Button
                    icon={DeleteIcon}
                    onClick={() => {
                        onChange({ product: null });
                    }}
                >
                    {said.remove}
                </Button>
                {revert}
            </div>
        </div>
    );
}
