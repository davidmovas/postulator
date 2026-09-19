import { useState } from "react";

import { copy } from "../copy/index.js";
import { failure } from "../data/errors.js";
import { useUnlock } from "../data/hooks/settings.js";

export function UnlockScreen() {
    const [password, setPassword] = useState("");
    const [problem, setProblem] = useState<string | null>(null);
    const unlocking = useUnlock();

    const submit = (event: React.FormEvent): void => {
        event.preventDefault();
        setProblem(null);
        unlocking.mutate(
            { password },
            {
                onError: (thrown) => {
                    const reported = failure(thrown);
                    if (reported.code === "LOCKED") {
                        setProblem(copy.lock.unrecoverable);
                        return;
                    }
                    setProblem(copy.lock.wrong);
                },
                onSuccess: () => {
                    setPassword("");
                },
            },
        );
    };

    return (
        <div className="flex h-full items-center justify-center bg-base-950">
            <form
                onSubmit={submit}
                className="w-80 rounded-panel border border-base-700 bg-base-900 p-5 shadow-xl"
            >
                <h1 className="text-ink-100 text-sm font-semibold">{copy.lock.title}</h1>
                <p className="text-ink-400 mt-1 text-xs">{copy.lock.prompt}</p>
                <label className="text-ink-300 mt-4 block text-xs" htmlFor="master-password">
                    {copy.lock.password}
                </label>
                <input
                    id="master-password"
                    type="password"
                    autoFocus
                    autoComplete="current-password"
                    className="mt-1 h-8 w-full rounded-panel border border-base-600 bg-base-800 px-2 text-sm text-ink-100"
                    value={password}
                    onChange={(event) => {
                        setPassword(event.target.value);
                    }}
                />
                <button
                    type="submit"
                    disabled={unlocking.isPending || password === ""}
                    className="mt-4 h-8 w-full rounded-panel bg-accent-600 text-sm font-medium text-ink-100 disabled:opacity-50"
                >
                    {unlocking.isPending ? copy.lock.unlocking : copy.lock.unlock}
                </button>
                {unlocking.isPending && <p className="text-ink-400 mt-2 text-xs">{copy.lock.unlocking}</p>}
                {problem !== null && <p className="text-bad-500 mt-2 text-xs">{problem}</p>}
            </form>
        </div>
    );
}
