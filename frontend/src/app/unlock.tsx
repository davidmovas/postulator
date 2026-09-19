import type { FormEvent } from "react";
import { useState } from "react";

import { copy } from "../copy/index.js";
import { failure } from "../data/errors.js";
import { useUnlock } from "../data/hooks/settings.js";
import { Button, Field, Input, LockIcon, Panel, Spinner } from "../ui/index.js";

export function UnlockScreen() {
    const [password, setPassword] = useState("");
    const [problem, setProblem] = useState<string | null>(null);
    const unlocking = useUnlock();

    const submit = (event: FormEvent): void => {
        event.preventDefault();
        setProblem(null);
        unlocking.mutate(
            { password },
            {
                onError: (thrown) => {
                    const reported = failure(thrown);
                    setProblem(reported.code === "LOCKED" ? copy.lock.unrecoverable : copy.lock.wrong);
                },
                onSuccess: () => {
                    setPassword("");
                },
            },
        );
    };

    return (
        <div className="flex h-full items-center justify-center bg-canvas">
            <Panel className="w-90 p-5">
                <form onSubmit={submit} className="flex flex-col gap-4">
                    <div className="flex items-center gap-2">
                        <LockIcon size={20} className="text-accent" />
                        <div className="flex flex-col">
                            <h1 className="text-lg font-semibold text-ink">{copy.lock.title}</h1>
                            <p className="text-xs text-ink-dim">{copy.lock.prompt}</p>
                        </div>
                    </div>
                    <Field label={copy.lock.password} error={problem}>
                        {(control) => (
                            <Input
                                {...control}
                                type="password"
                                autoFocus={true}
                                autoComplete="current-password"
                                disabled={unlocking.isPending}
                                value={password}
                                onChange={(event) => {
                                    setPassword(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    <Button
                        type="submit"
                        variant="primary"
                        busy={unlocking.isPending}
                        disabled={password === ""}
                        className="w-full"
                    >
                        {unlocking.isPending ? copy.lock.unlocking : copy.lock.unlock}
                    </Button>
                    {unlocking.isPending ? (
                        <p className="flex items-center gap-2 text-xs text-ink-dim">
                            <Spinner size={12} />
                            {copy.lock.unlocking}
                        </p>
                    ) : null}
                </form>
            </Panel>
        </div>
    );
}
