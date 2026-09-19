const relative = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

const absolute = new Intl.DateTimeFormat("en-GB", {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
});

const wholeNumbers = new Intl.NumberFormat("en", { maximumFractionDigits: 0 });

const compactNumbers = new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 });

const money = new Intl.NumberFormat("en", { style: "currency", currency: "USD", maximumFractionDigits: 2 });

const preciseMoney = new Intl.NumberFormat("en", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
});

export const never = "never";

interface Division {
    unit: Intl.RelativeTimeFormatUnit;
    seconds: number;
}

const divisions: readonly Division[] = [
    { unit: "year", seconds: 31_536_000 },
    { unit: "month", seconds: 2_592_000 },
    { unit: "week", seconds: 604_800 },
    { unit: "day", seconds: 86_400 },
    { unit: "hour", seconds: 3_600 },
    { unit: "minute", seconds: 60 },
    { unit: "second", seconds: 1 },
];

function parse(value: string | null | undefined): Date | null {
    if (value === null || value === undefined || value === "") {
        return null;
    }
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? null : parsed;
}

export function relativeTime(value: string | null | undefined, now: Date = new Date()): string {
    const moment = parse(value);
    if (moment === null) {
        return never;
    }
    const seconds = (moment.getTime() - now.getTime()) / 1000;
    const magnitude = Math.abs(seconds);
    for (const division of divisions) {
        if (magnitude >= division.seconds || division.unit === "second") {
            const amount = Math.round(seconds / division.seconds);
            return relative.format(amount, division.unit);
        }
    }
    return relative.format(0, "second");
}

export function absoluteTime(value: string | null | undefined): string {
    const moment = parse(value);
    return moment === null ? never : absolute.format(moment);
}

export function duration(milliseconds: number): string {
    if (milliseconds < 1000) {
        return `${wholeNumbers.format(milliseconds)} ms`;
    }
    const seconds = milliseconds / 1000;
    if (seconds < 60) {
        return `${seconds.toFixed(1)} s`;
    }
    const minutes = Math.floor(seconds / 60);
    const rest = Math.round(seconds - minutes * 60);
    return `${minutes}m ${rest}s`;
}

const byteUnits = ["B", "KB", "MB", "GB", "TB"] as const;

export function bytes(count: number): string {
    let size = Math.max(count, 0);
    let unit = 0;
    while (size >= 1024 && unit < byteUnits.length - 1) {
        size /= 1024;
        unit += 1;
    }
    const digits = unit === 0 ? 0 : 1;
    return `${size.toFixed(digits)} ${byteUnits[unit]}`;
}

export function tokens(count: number): string {
    return count < 10_000 ? wholeNumbers.format(count) : compactNumbers.format(count);
}

export function usd(amount: number): string {
    if (amount !== 0 && Math.abs(amount) < 0.01) {
        return preciseMoney.format(amount);
    }
    return money.format(amount);
}

