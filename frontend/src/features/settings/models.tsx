import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../data/errors.js";
import {
    useDisableModel,
    useModelCatalog,
    useRoleProfiles,
    useSetProfile,
    useTestProvider,
} from "../../data/hooks/models.js";
import { useDeleteProviderKey, useProviderKeys, useSetProviderKey } from "../../data/hooks/settings.js";
import { usd } from "../../domain/format.js";
import { modelRoles } from "../../generated/vocab.js";
import {
    Banner,
    Button,
    DenseTable,
    Dialog,
    EmptyState,
    Field,
    IconButton,
    Input,
    KeyIcon,
    KeyOffIcon,
    Panel,
    PanelHeader,
    Select,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    TravelExploreIcon,
    VisibilityIcon,
    VisibilityOffIcon,
} from "../../ui/index.js";
import { profileRows, refOf, refText } from "./model/profiles.js";
import type { ProfileSource } from "./model/profiles.js";
import { SpendPanel } from "./spend.js";

const roleLabels = copy.settings.models.profiles.labels as Readonly<Record<string, string>>;

const sourceLabels: Readonly<Record<ProfileSource, string>> = {
    global: copy.settings.models.profiles.sourceGlobal,
    seeded: copy.settings.models.profiles.sourceSeeded,
    none: copy.settings.models.profiles.sourceNone,
};

interface Tested {
    provider: string;
    latencyMs: number;
    tokens: number;
}

function ProviderKeysPanel(): ReactElement {
    const providers = useProviderKeys();
    const catalog = useModelCatalog();
    const profiles = useRoleProfiles();
    const store = useSetProviderKey();
    const revoke = useDeleteProviderKey();
    const probe = useTestProvider();

    const [storing, setStoring] = useState<string | null>(null);
    const [revoking, setRevoking] = useState<string | null>(null);
    const [apiKey, setApiKey] = useState("");
    const [revealed, setRevealed] = useState(false);
    const [tested, setTested] = useState<Tested | null>(null);

    const rows = providers.data?.providers ?? [];
    const models = catalog.data?.models ?? [];

    const modelFor = (provider: string): string | null => {
        const chat = profileRows([...modelRoles], profiles.data?.profiles ?? []).find((row) => row.role === "chat");
        if (chat?.effective?.provider === provider) {
            return chat.effective.model;
        }
        return models.find((model) => model.provider === provider)?.model ?? null;
    };

    return (
        <Panel>
            <PanelHeader title={copy.settings.models.keys.title} />
            <p className="border-b border-hairline px-3 py-2 text-xs text-ink-dim">
                {copy.settings.models.keys.blurb}
            </p>
            {rows.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={KeyOffIcon} title={copy.settings.models.keys.title} body={copy.empty.providerKeys} />
                </div>
            ) : (
                <DenseTable columns="minmax(8rem,1fr) 8rem minmax(10rem,1fr) 14rem" label={copy.settings.models.keys.title}>
                    <TableHead>
                        <TableCell>{copy.settings.models.keys.provider}</TableCell>
                        <TableCell>{copy.settings.models.keys.state}</TableCell>
                        <TableCell>{copy.settings.models.keys.test}</TableCell>
                        <TableCell align="right">{copy.app.actions}</TableCell>
                    </TableHead>
                    {rows.map((row) => (
                        <TableRow key={row.provider}>
                            <TableCell mono={true}>{row.provider}</TableCell>
                            <TableCell>
                                <StatusBadge tone={row.configured ? "ok" : "warn"} icon={row.configured ? KeyIcon : KeyOffIcon}>
                                    {row.configured ? copy.settings.models.keys.stored : copy.settings.models.keys.missing}
                                </StatusBadge>
                            </TableCell>
                            <TableCell muted={true}>
                                {tested !== null && tested.provider === row.provider
                                    ? copy.settings.models.keys.tested(tested.latencyMs, tested.tokens)
                                    : ""}
                            </TableCell>
                            <TableCell align="right">
                                <div className="flex justify-end gap-1">
                                    <Button
                                        size="sm"
                                        variant="secondary"
                                        onClick={() => {
                                            setApiKey("");
                                            setRevealed(false);
                                            setStoring(row.provider);
                                        }}
                                    >
                                        {row.configured ? copy.settings.models.keys.replace : copy.settings.models.keys.store}
                                    </Button>
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        icon={TravelExploreIcon}
                                        busy={probe.isPending}
                                        disabled={!row.configured || modelFor(row.provider) === null}
                                        title={modelFor(row.provider) === null ? copy.settings.models.keys.noModel : undefined}
                                        onClick={() => {
                                            const model = modelFor(row.provider);
                                            if (model === null) {
                                                return;
                                            }
                                            probe.mutate(
                                                { provider: row.provider, model },
                                                {
                                                    onSuccess: (answered) => {
                                                        setTested({
                                                            provider: row.provider,
                                                            latencyMs: answered.latencyMs,
                                                            tokens: answered.usage?.total ?? 0,
                                                        });
                                                    },
                                                },
                                            );
                                        }}
                                    >
                                        {probe.isPending ? copy.settings.models.keys.testing : copy.settings.models.keys.test}
                                    </Button>
                                    <Button
                                        size="sm"
                                        variant="danger"
                                        disabled={!row.configured}
                                        onClick={() => {
                                            setRevoking(row.provider);
                                        }}
                                    >
                                        {copy.settings.models.keys.revoke}
                                    </Button>
                                </div>
                            </TableCell>
                        </TableRow>
                    ))}
                </DenseTable>
            )}
            {formErrorOf(probe.error) === null ? null : (
                <div className="px-3 pb-3">
                    <Banner tone="danger" title={formErrorOf(probe.error) ?? ""} />
                </div>
            )}

            <Dialog
                open={storing !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setStoring(null);
                    }
                }}
                title={storing === null ? "" : copy.settings.models.keys.storeTitle(storing)}
                description={copy.settings.models.keys.apiKeyHint}
                confirmLabel={copy.app.save}
                cancelLabel={copy.app.cancel}
                busy={store.isPending}
                icon={KeyIcon}
                onConfirm={() => {
                    if (storing === null || apiKey.trim() === "") {
                        return;
                    }
                    store.mutate(
                        { provider: storing, apiKey },
                        {
                            onSuccess: () => {
                                setApiKey("");
                                setStoring(null);
                            },
                        },
                    );
                }}
            >
                <Field
                    label={copy.settings.models.keys.apiKey}
                    required={true}
                    error={fieldErrorOf(store.error, "apiKey") ?? formErrorOf(store.error)}
                >
                    {(binding) => (
                        <div className="flex items-center gap-1">
                            <div className="min-w-0 flex-1">
                                <Input
                                    id={binding.id}
                                    aria-describedby={binding["aria-describedby"]}
                                    invalid={binding.invalid}
                                    mono={true}
                                    autoComplete="off"
                                    type={revealed ? "text" : "password"}
                                    value={apiKey}
                                    onChange={(event) => {
                                        setApiKey(event.target.value);
                                    }}
                                />
                            </div>
                            <IconButton
                                icon={revealed ? VisibilityOffIcon : VisibilityIcon}
                                label={copy.settings.models.keys.apiKey}
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                    setRevealed(!revealed);
                                }}
                            />
                        </div>
                    )}
                </Field>
            </Dialog>

            <Dialog
                open={revoking !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setRevoking(null);
                    }
                }}
                title={revoking === null ? "" : copy.settings.models.keys.revokeTitle(revoking)}
                description={revoking === null ? "" : copy.settings.models.keys.revokeBody(revoking)}
                confirmLabel={copy.settings.models.keys.revoke}
                cancelLabel={copy.app.cancel}
                destructive={true}
                icon={KeyOffIcon}
                busy={revoke.isPending}
                onConfirm={() => {
                    if (revoking === null) {
                        return;
                    }
                    revoke.mutate(
                        { provider: revoking },
                        {
                            onSuccess: () => {
                                setRevoking(null);
                            },
                        },
                    );
                }}
            />
        </Panel>
    );
}

function RoleProfilesPanel(): ReactElement {
    const profiles = useRoleProfiles();
    const catalog = useModelCatalog();
    const choose = useSetProfile();

    const models = catalog.data?.models ?? [];
    const options = models.map((model) => ({
        value: refText({ provider: model.provider, model: model.model }),
        label: refText({ provider: model.provider, model: model.model }),
    }));
    const rows = profileRows([...modelRoles], profiles.data?.profiles ?? []);

    return (
        <Panel>
            <PanelHeader title={copy.settings.models.profiles.title} />
            <p className="border-b border-hairline px-3 py-2 text-xs text-ink-dim">
                {copy.settings.models.profiles.blurb}
            </p>
            {models.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={KeyOffIcon} title={copy.settings.models.profiles.title} body={copy.empty.models} />
                </div>
            ) : (
                <DenseTable
                    columns="8rem minmax(12rem,1fr) minmax(10rem,1fr) 9rem"
                    label={copy.settings.models.profiles.title}
                >
                    <TableHead>
                        <TableCell>{copy.settings.models.profiles.role}</TableCell>
                        <TableCell>{copy.settings.models.profiles.choice}</TableCell>
                        <TableCell>{copy.settings.models.profiles.effective}</TableCell>
                        <TableCell>{copy.settings.models.profiles.source}</TableCell>
                    </TableHead>
                    {rows.map((row) => (
                        <TableRow key={row.role}>
                            <TableCell>{roleLabels[row.role] ?? row.role}</TableCell>
                            <TableCell>
                                <Select
                                    aria-label={`${roleLabels[row.role] ?? row.role} ${copy.settings.models.profiles.choice}`}
                                    value={row.chosen === null ? null : refText(row.chosen)}
                                    placeholder={copy.settings.models.profiles.sourceNone}
                                    options={options}
                                    onValueChange={(picked) => {
                                        const ref = refOf(picked);
                                        if (ref === null) {
                                            return;
                                        }
                                        choose.mutate({ role: row.role, provider: ref.provider, model: ref.model });
                                    }}
                                />
                            </TableCell>
                            <TableCell mono={true} muted={row.effective === null}>
                                {refText(row.effective)}
                            </TableCell>
                            <TableCell muted={true}>{sourceLabels[row.source]}</TableCell>
                        </TableRow>
                    ))}
                </DenseTable>
            )}
            {formErrorOf(choose.error) === null ? null : (
                <div className="px-3 pb-3">
                    <Banner tone="danger" title={formErrorOf(choose.error) ?? ""} />
                </div>
            )}
        </Panel>
    );
}

function ModelCatalogPanel(): ReactElement {
    const catalog = useModelCatalog();
    const disable = useDisableModel();
    const [disabling, setDisabling] = useState<{ provider: string; model: string } | null>(null);

    const models = catalog.data?.models ?? [];

    return (
        <Panel>
            <PanelHeader title={copy.settings.models.catalog.title} />
            <p className="border-b border-hairline px-3 py-2 text-xs text-ink-dim">
                {copy.settings.models.catalog.blurb}
            </p>
            {models.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={KeyOffIcon} title={copy.settings.models.catalog.title} body={copy.empty.models} />
                </div>
            ) : (
                <DenseTable
                    columns="minmax(12rem,1.6fr) 6rem 6rem 6rem 6rem 5rem 6rem 10rem 6rem"
                    label={copy.settings.models.catalog.title}
                >
                    <TableHead>
                        <TableCell>{copy.settings.models.catalog.model}</TableCell>
                        <TableCell align="right">{copy.settings.models.catalog.context}</TableCell>
                        <TableCell align="right">{copy.settings.models.catalog.maxOutput}</TableCell>
                        <TableCell align="right">{copy.settings.models.catalog.inputPrice}</TableCell>
                        <TableCell align="right">{copy.settings.models.catalog.outputPrice}</TableCell>
                        <TableCell align="right">{copy.settings.models.catalog.rpm}</TableCell>
                        <TableCell align="right">{copy.settings.models.catalog.tpm}</TableCell>
                        <TableCell>{copy.settings.models.catalog.flags}</TableCell>
                        <TableCell align="right">{copy.app.actions}</TableCell>
                    </TableHead>
                    {models.map((model) => (
                        <TableRow key={`${model.provider}/${model.model}`}>
                            <TableCell mono={true}>{`${model.provider}/${model.model}`}</TableCell>
                            <TableCell align="right" mono={true}>
                                {String(model.contextTokens)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {String(model.maxOutputTokens)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {usd(model.inputUsdPerM)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {usd(model.outputUsdPerM)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {String(model.rpm)}
                            </TableCell>
                            <TableCell align="right" mono={true}>
                                {String(model.tpm)}
                            </TableCell>
                            <TableCell>
                                <div className="flex flex-wrap gap-1">
                                    {model.supportsStructured ? (
                                        <StatusBadge tone="muted" dot={false}>
                                            {copy.settings.models.catalog.structured}
                                        </StatusBadge>
                                    ) : null}
                                    {model.supportsImages ? (
                                        <StatusBadge tone="muted" dot={false}>
                                            {copy.settings.models.catalog.images}
                                        </StatusBadge>
                                    ) : null}
                                    {model.reasoning ? (
                                        <StatusBadge tone="muted" dot={false}>
                                            {copy.settings.models.catalog.reasoning}
                                        </StatusBadge>
                                    ) : null}
                                </div>
                            </TableCell>
                            <TableCell align="right">
                                <Button
                                    size="sm"
                                    variant="danger"
                                    onClick={() => {
                                        setDisabling({ provider: model.provider, model: model.model });
                                    }}
                                >
                                    {copy.settings.models.catalog.disable}
                                </Button>
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
                title={
                    disabling === null
                        ? ""
                        : copy.settings.models.catalog.disableTitle(`${disabling.provider}/${disabling.model}`)
                }
                description={copy.settings.models.catalog.disableBody}
                confirmLabel={copy.settings.models.catalog.disable}
                cancelLabel={copy.app.cancel}
                destructive={true}
                busy={disable.isPending}
                onConfirm={() => {
                    if (disabling === null) {
                        return;
                    }
                    disable.mutate(disabling, {
                        onSuccess: () => {
                            setDisabling(null);
                        },
                    });
                }}
            />
        </Panel>
    );
}

export function ModelSettingsScreen(): ReactElement {
    return (
        <div className="flex max-w-5xl flex-col gap-3">
            <ProviderKeysPanel />
            <RoleProfilesPanel />
            <ModelCatalogPanel />
            <SpendPanel />
        </div>
    );
}
