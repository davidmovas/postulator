import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { useDisableModel, useModelCatalog } from "../../../data/hooks/models.js";
import type { CatalogModel } from "../../../data/types.js";
import {
    AddIcon,
    Button,
    CheckIcon,
    DenseTable,
    Dialog,
    Drawer,
    EmptyState,
    KeyOffIcon,
    TableCell,
    TableHead,
    TableRow,
} from "../../../ui/index.js";
import { ModelForm } from "./catalog-form.js";

const said = copy.settings.models.catalog;

const perMillion = new Intl.NumberFormat("en", { maximumFractionDigits: 3 });

function rates(...prices: readonly number[]): string {
    return prices.map((price) => perMillion.format(price)).join(" · ");
}

function offersFlex(model: CatalogModel): boolean {
    return model.flexInputUsdPerM > 0;
}

function standardOf(model: CatalogModel): string {
    const cached = model.cachedInputUsdPerM > 0 ? model.cachedInputUsdPerM : model.inputUsdPerM;
    return rates(model.inputUsdPerM, cached, model.outputUsdPerM);
}

function flexOf(model: CatalogModel): string {
    return offersFlex(model) ? rates(model.flexInputUsdPerM, model.flexOutputUsdPerM) : said.noFlex;
}

type Pane ={ kind: "list" } | { kind: "form"; editing: CatalogModel | null };

export interface CatalogDrawerProps {
    onClose: () => void;
}

export function CatalogDrawer({ onClose }: CatalogDrawerProps): ReactElement {
    const catalog = useModelCatalog();
    const disable = useDisableModel();
    const [pane, setPane] = useState<Pane>({ kind: "list" });
    const [disabling, setDisabling] = useState<CatalogModel | null>(null);

    const models = catalog.data?.models ?? [];

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={pane.kind === "list" ? said.title : pane.editing === null ? said.addTitle : said.editTitle(pane.editing.model)}
            description={pane.kind === "list" ? said.subtitle(models.length) : undefined}
            closeLabel={said.close}
            width={688}
            header={
                pane.kind === "list" ? (
                    <Button
                        size="sm"
                        variant="secondary"
                        icon={AddIcon}
                        onClick={() => {
                            setPane({ kind: "form", editing: null });
                        }}
                    >
                        {said.add}
                    </Button>
                ) : undefined
            }
        >
            {pane.kind === "form" ? (
                <ModelForm
                    editing={pane.editing}
                    onDone={() => {
                        setPane({ kind: "list" });
                    }}
                />
            ) : models.length === 0 ? (
                <div className="p-3">
                    <EmptyState
                        icon={KeyOffIcon}
                        title={said.title}
                        body={copy.empty.models}
                        actions={
                            <Button
                                variant="primary"
                                icon={AddIcon}
                                onClick={() => {
                                    setPane({ kind: "form", editing: null });
                                }}
                            >
                                {said.add}
                            </Button>
                        }
                    />
                </div>
            ) : (
                <DenseTable columns="minmax(8rem,1fr) 9.5rem 6.5rem 3.5rem 7rem" label={said.title}>
                    <TableHead>
                        <TableCell>{said.model}</TableCell>
                        <TableCell align="right" title={said.standardOrder}>
                            {said.standardPrice}
                        </TableCell>
                        <TableCell align="right" title={said.flexOrder}>
                            {said.flexPrice}
                        </TableCell>
                        <TableCell align="right">{said.images}</TableCell>
                        <TableCell align="right">{copy.app.actions}</TableCell>
                    </TableHead>
                    {models.map((model) => (
                        <TableRow key={`${model.provider}/${model.model}`}>
                            <TableCell mono={true}>{model.model}</TableCell>
                            <TableCell align="right" mono={true} title={said.standardOrder}>
                                {standardOf(model)}
                            </TableCell>
                            <TableCell align="right" mono={true} muted={!offersFlex(model)} title={said.flexOrder}>
                                {flexOf(model)}
                            </TableCell>
                            <TableCell align="right">
                                {model.supportsImages ? <CheckIcon size={14} /> : null}
                            </TableCell>
                            <TableCell align="right">
                                <div className="flex justify-end gap-1">
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        onClick={() => {
                                            setPane({ kind: "form", editing: model });
                                        }}
                                    >
                                        {said.edit}
                                    </Button>
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        onClick={() => {
                                            setDisabling(model);
                                        }}
                                    >
                                        {said.disable}
                                    </Button>
                                </div>
                            </TableCell>
                        </TableRow>
                    ))}
                </DenseTable>
            )}

            <Dialog
                open={disabling !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setDisabling(null);
                    }
                }}
                title={disabling === null ? "" : said.disableTitle(disabling.model)}
                description={said.disableBody}
                confirmLabel={said.disable}
                cancelLabel={copy.app.cancel}
                destructive={true}
                busy={disable.isPending}
                onConfirm={() => {
                    if (disabling === null) {
                        return;
                    }
                    disable.mutate(
                        { provider: disabling.provider, model: disabling.model },
                        {
                            onSuccess: () => {
                                setDisabling(null);
                            },
                        },
                    );
                }}
            />
        </Drawer>
    );
}
