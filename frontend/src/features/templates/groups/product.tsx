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
    RestartAltIcon,
    Switch,
    Textarea,
} from "../../../ui/index.js";
import { NumberInput } from "../controls.js";
import type { ProductDraft, SpecDraft } from "../spec.js";
import { emptyProduct } from "../spec.js";

const said = copy.templates.product;

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
                onChange({ product: below.product === null ? null : { ...below.product } });
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
