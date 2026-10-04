import { useEffect, useState } from "react";

const secondMs = 1000;

export function useNow(intervalMs: number, active: boolean): number {
    const [now, setNow] = useState(() => Date.now());

    useEffect(() => {
        if (!active) {
            return undefined;
        }
        const timer = window.setInterval(() => {
            setNow(Date.now());
        }, intervalMs);
        return () => {
            window.clearInterval(timer);
        };
    }, [intervalMs, active]);

    return now;
}

export function useElapsed(running: boolean): number {
    const [seconds, setSeconds] = useState(0);

    useEffect(() => {
        if (!running) {
            setSeconds(0);
            return undefined;
        }
        const started = Date.now();
        const timer = window.setInterval(() => {
            setSeconds(Math.round((Date.now() - started) / secondMs));
        }, secondMs);
        return () => {
            window.clearInterval(timer);
        };
    }, [running]);

    return seconds;
}
