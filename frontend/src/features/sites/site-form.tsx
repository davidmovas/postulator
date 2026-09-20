import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useCreateSite, useTestConnection, useUpdateSite } from "../../data/hooks/sites.js";
import type { Reachability, Site } from "../../data/types.js";
import { localAddress } from "../../domain/address.js";
import { Banner, Button, Drawer, Field, Input, Switch, TravelExploreIcon } from "../../ui/index.js";
import { ReachabilityReport } from "./reachability.js";

type FieldErrors = Readonly<Record<string, string>>;

const drawerWidth = 688;

export interface SiteFormProps {
    site: Site | null;
    onClose: () => void;
    onSaved: (site: Site) => void;
}

export function SiteForm({ site, onClose, onSaved }: SiteFormProps): ReactElement {
    const editing = site !== null;

    const [name, setName] = useState(site?.name ?? "");
    const [baseUrl, setBaseUrl] = useState(site?.baseUrl ?? "");
    const [username, setUsername] = useState(site?.username ?? "");
    const [password, setPassword] = useState("");
    const [allowInsecure, setAllowInsecure] = useState(site?.allowInsecure ?? false);
    const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
    const [formError, setFormError] = useState<string | null>(null);
    const [reachability, setReachability] = useState<Reachability | null>(null);

    const local = localAddress(baseUrl);

    const create = useCreateSite();
    const update = useUpdateSite();
    const probe = useTestConnection();

    const clear = (field: string): void => {
        setFormError(null);
        setFieldErrors((held) => {
            if (held[field] === undefined) {
                return held;
            }
            const next = { ...held };
            delete next[field];
            return next;
        });
    };

    const absorb = (thrown: unknown): void => {
        const reaction = react(thrown);
        if (reaction.kind === "field") {
            setFieldErrors((held) => ({ ...held, [reaction.field]: reaction.message }));
            setFormError(null);
            return;
        }
        setFormError(reaction.kind === "form" ? reaction.message : null);
    };

    const runTest = (): void => {
        setFieldErrors({});
        setFormError(null);
        setReachability(null);
        probe.mutate(
            {
                siteId: site?.id ?? "",
                baseUrl: baseUrl.trim(),
                username: username.trim(),
                password,
                allowInsecure,
            },
            {
                onSuccess: (answered) => {
                    setReachability(answered.reachability);
                },
                onError: absorb,
            },
        );
    };

    const save = (): void => {
        setFieldErrors({});
        setFormError(null);
        const settled = {
            onSuccess: (answered: { site: Site }) => {
                onSaved(answered.site);
                onClose();
            },
            onError: absorb,
        };
        if (site === null) {
            create.mutate(
                {
                    name: name.trim(),
                    baseUrl: baseUrl.trim(),
                    username: username.trim(),
                    password,
                    allowInsecure,
                },
                settled,
            );
            return;
        }
        update.mutate(
            {
                id: site.id,
                name: name.trim(),
                baseUrl: baseUrl.trim(),
                username: username.trim(),
                allowInsecure,
                ...(password === "" ? {} : { password }),
            },
            settled,
        );
    };

    const saving = create.isPending || update.isPending;

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={editing ? copy.sites.editTitle : copy.sites.addTitle}
            closeLabel={copy.app.cancel}
            width={drawerWidth}
            footer={
                <>
                    <Button variant="ghost" onClick={onClose}>
                        {copy.app.cancel}
                    </Button>
                    <Button variant="primary" busy={saving} onClick={save}>
                        {copy.app.save}
                    </Button>
                </>
            }
        >
            <div className="flex flex-col gap-3 p-3">
                <Field label={copy.sites.field.name} required error={fieldErrors["name"]}>
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={name}
                            autoFocus={true}
                            onChange={(event) => {
                                setName(event.target.value);
                                clear("name");
                            }}
                        />
                    )}
                </Field>
                <Field label={copy.sites.field.baseUrl} required error={fieldErrors["baseUrl"]}>
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            mono={true}
                            inputMode="url"
                            placeholder={copy.sites.field.baseUrlExample}
                            value={baseUrl}
                            onChange={(event) => {
                                setBaseUrl(event.target.value);
                                clear("baseUrl");
                            }}
                        />
                    )}
                </Field>
                <Field label={copy.sites.field.username} required={!editing} error={fieldErrors["username"]}>
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            value={username}
                            onChange={(event) => {
                                setUsername(event.target.value);
                                clear("username");
                            }}
                        />
                    )}
                </Field>
                <Field
                    label={copy.sites.field.password}
                    required={!editing}
                    tooltip={copy.sites.field.passwordTooltip}
                    error={fieldErrors["password"]}
                >
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            invalid={control.invalid}
                            type="password"
                            mono={true}
                            autoComplete="off"
                            placeholder={editing ? copy.sites.field.passwordKeep : undefined}
                            value={password}
                            onChange={(event) => {
                                setPassword(event.target.value);
                                clear("password");
                            }}
                        />
                    )}
                </Field>
                {local ? null : (
                    <Switch
                        label={copy.sites.field.allowInsecure}
                        checked={allowInsecure}
                        tone="danger"
                        onChange={(event) => {
                            setAllowInsecure(event.target.checked);
                            clear("baseUrl");
                        }}
                    />
                )}
                {allowInsecure && !local ? <Banner tone="warn" title={copy.sites.insecureWarning} /> : null}
                {formError === null ? null : <Banner tone="danger" title={formError} />}
                <div className="flex items-center gap-2">
                    <Button variant="secondary" icon={TravelExploreIcon} busy={probe.isPending} onClick={runTest}>
                        {probe.isPending ? copy.sites.testing : copy.sites.test}
                    </Button>
                </div>
                {reachability === null ? null : (
                    <ReachabilityReport
                        result={reachability}
                        onUseSuggested={(suggested) => {
                            setBaseUrl(suggested);
                            setAllowInsecure(false);
                            setReachability(null);
                            clear("baseUrl");
                        }}
                    />
                )}
            </div>
        </Drawer>
    );
}
