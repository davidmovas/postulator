import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { failure, fieldErrorOf, formErrorOf, providerMessageOf } from "../../../data/errors.js";
import { useTestProvider } from "../../../data/hooks/models.js";
import { useDeleteProviderKey, useProviderKeys, useSetProviderKey } from "../../../data/hooks/settings.js";
import {
    Button,
    Dialog,
    Field,
    IconButton,
    Input,
    KeyIcon,
    KeyOffIcon,
    Panel,
    Skeleton,
    StatusBadge,
    VisibilityIcon,
    VisibilityOffIcon,
} from "../../../ui/index.js";

const said = copy.settings.models.keys;
const names = copy.settings.models.providers as Readonly<Record<string, string>>;

interface Tested {
    provider: string;
    latencyMs: number;
    tokens: number;
}

export function ProviderCards(): ReactElement {
    const providers = useProviderKeys();
    const store = useSetProviderKey();
    const revoke = useDeleteProviderKey();
    const probe = useTestProvider();

    const [storing, setStoring] = useState<string | null>(null);
    const [revoking, setRevoking] = useState<string | null>(null);
    const [apiKey, setApiKey] = useState("");
    const [revealed, setRevealed] = useState(false);
    const [tested, setTested] = useState<Tested | null>(null);
    const [failed, setFailed] = useState<string | null>(null);

    const rows = providers.data?.providers ?? [];
    const reported = failed === null || probe.error === null ? null : failure(probe.error);

    if (providers.isPending) {
        return <Skeleton height={104} />;
    }

    return (
        <div className="@container">
            <div className="grid grid-cols-1 gap-3 @lg:grid-cols-2 @4xl:grid-cols-4">
                {rows.map((row) => {
                    return (
                        <Panel key={row.provider}>
                            <div className="flex flex-col gap-3 p-3">
                                <div className="flex items-center justify-between gap-2">
                                    <span className="min-w-0 truncate text-sm font-semibold text-ink">
                                        {names[row.provider] ?? row.provider}
                                    </span>
                                    <StatusBadge
                                        tone={row.configured ? "ok" : "warn"}
                                        icon={row.configured ? KeyIcon : KeyOffIcon}
                                    >
                                        {row.configured ? said.stored : said.missing}
                                    </StatusBadge>
                                </div>
                                <div className="flex flex-wrap gap-1">
                                    <Button
                                        size="sm"
                                        variant={row.configured ? "secondary" : "primary"}
                                        onClick={() => {
                                            setApiKey("");
                                            setRevealed(false);
                                            setStoring(row.provider);
                                        }}
                                    >
                                        {row.configured ? said.replace : said.store}
                                    </Button>
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        busy={probe.isPending}
                                        disabled={!row.configured}
                                        title={row.configured ? undefined : said.noKey}
                                        onClick={() => {
                                            setTested(null);
                                            setFailed(null);
                                            probe.mutate(
                                                { provider: row.provider, model: "" },
                                                {
                                                    onSuccess: (answered) => {
                                                        setTested({
                                                            provider: row.provider,
                                                            latencyMs: answered.latencyMs,
                                                            tokens: answered.usage?.total ?? 0,
                                                        });
                                                    },
                                                    onError: () => {
                                                        setFailed(row.provider);
                                                    },
                                                },
                                            );
                                        }}
                                    >
                                        {probe.isPending ? said.testing : said.test}
                                    </Button>
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        disabled={!row.configured}
                                        onClick={() => {
                                            setRevoking(row.provider);
                                        }}
                                    >
                                        {said.revoke}
                                    </Button>
                                </div>
                                {tested !== null && tested.provider === row.provider ? (
                                    <p className="text-xs text-ok">{said.tested(tested.latencyMs, tested.tokens)}</p>
                                ) : null}
                                {failed === row.provider && reported !== null ? (
                                    <div className="flex flex-col items-start gap-1">
                                        <StatusBadge tone="danger" dot={false}>
                                            {reported.code}
                                        </StatusBadge>
                                        <p className="text-xs text-danger">{reported.message}</p>
                                        {providerMessageOf(reported) === null ? null : (
                                            <p className="font-mono text-2xs break-words text-ink-dim">
                                                {providerMessageOf(reported)}
                                            </p>
                                        )}
                                    </div>
                                ) : null}
                            </div>
                        </Panel>
                    );
                })}
            </div>

            <Dialog
                open={storing !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setStoring(null);
                    }
                }}
                title={storing === null ? "" : said.storeTitle(names[storing] ?? storing)}
                description={said.apiKeyHint}
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
                    label={said.apiKey}
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
                                label={said.apiKey}
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
                title={revoking === null ? "" : said.revokeTitle(names[revoking] ?? revoking)}
                description={revoking === null ? "" : said.revokeBody(names[revoking] ?? revoking)}
                confirmLabel={said.revoke}
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
        </div>
    );
}
