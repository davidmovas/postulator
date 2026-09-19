import { copy } from "../copy/index.js";

export interface NotBuiltProps {
    screen: string;
    wave: string;
}

export function NotBuilt({ screen, wave }: NotBuiltProps) {
    return (
        <section className="m-6 max-w-xl rounded-panel border border-base-700 bg-base-850 p-5">
            <h2 className="text-ink-100 text-sm font-semibold">
                {screen} — {copy.notBuilt.title}
            </h2>
            <p className="text-ink-300 mt-2 text-sm">{copy.notBuilt.body}</p>
            <p className="text-ink-400 mt-1 text-xs">{copy.notBuilt.wave(wave)}</p>
        </section>
    );
}
