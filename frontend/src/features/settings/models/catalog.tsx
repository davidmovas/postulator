import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { useDisableModel, useModelCatalog } from "../../../data/hooks/models.js";
import type { CatalogModel } from "../../../data/types.js";
import { usd } from "../../../domain/format.js";
import {
    AddIcon,
    Button,
    DenseTable,
    Dialog,
    Drawer,
    EmptyState,
    KeyOffIcon,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../../ui/index.js";
import { ModelForm } from "./catalog-form.js";

const said = copy.settings.models.catalog;

type Pane = { kind: "list" } | { kind: "form"; editing: CatalogModel | null };

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
                <DenseTable columns="minmax(10rem,1fr) 5rem 5rem 4rem 4rem 8rem 7rem" label={said.title}>
                    <TableHead>
                        <TableCell>{said.model}</TableCell>
                        <TableCell align="right">{said.inputPrice}</TableCell>
                        <TableCell align="right">{said.outputPrice}</TableCell>
                        <TableCell align="right">{said.context}</TableCell>
                        <TableCell align="right">{said.rpm}</TableCell>
                        <TableCell>{said.flags}</TableCell>
                        <TableCell align="right">{copy.app.actions}</TableCell>
                    </TableHead>
                    {models.map((model) => (
                        <TableRow key={`${model.provider}/${model.model}`}>
                            <TableCell mono={true}>{`${model.provider}/${model.model}`}</TableCell>
                            <TableCell align="right" mono={true}>
                                {usd(model.inputUsdPerM)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {usd(model.outputUsdPerM)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {String(model.contextTokens)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {String(model.rpm)}
                            </TableCell>
                            <TableCell>
                                <div className="flex flex-wrap gap-1">
                                    {model.supportsStructured ? (
                                        <StatusBadge tone="muted" dot={false}>
                                            {said.structured}
                                        </StatusBadge>
                                    ) : null}
                                    {model.supportsImages ? (
                                        <StatusBadge tone="muted" dot={false}>
                                            {said.images}
                                        </StatusBadge>
                                    ) : null}
                                    {model.reasoning ? (
                                        <StatusBadge tone="muted" dot={false}>
                                            {said.reasoning}
                                        </StatusBadge>
                                    ) : null}
                                </div>
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
                title={disabling === null ? "" : said.disableTitle(`${disabling.provider}/${disabling.model}`)}
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
