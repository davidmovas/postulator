const zone = "UTC";

const weekdayNames = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"] as const;

const monthNames = [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
] as const;

const weekdayAliases: Readonly<Record<string, number>> = {
    SUN: 0,
    MON: 1,
    TUE: 2,
    WED: 3,
    THU: 4,
    FRI: 5,
    SAT: 6,
};

const monthAliases: Readonly<Record<string, number>> = {
    JAN: 1,
    FEB: 2,
    MAR: 3,
    APR: 4,
    MAY: 5,
    JUN: 6,
    JUL: 7,
    AUG: 8,
    SEP: 9,
    OCT: 10,
    NOV: 11,
    DEC: 12,
};

const descriptors: Readonly<Record<string, string>> = {
    "@yearly": "0 0 1 1 *",
    "@annually": "0 0 1 1 *",
    "@monthly": "0 0 1 * *",
    "@weekly": "0 0 * * 0",
    "@daily": "0 0 * * *",
    "@midnight": "0 0 * * *",
    "@hourly": "0 * * * *",
};

const durationUnits: Readonly<Record<string, number>> = {
    ns: 1e-9,
    us: 1e-6,
    ms: 1e-3,
    s: 1,
    m: 60,
    h: 3600,
};

const maxListedTimes = 3;
const weekdayRun = [1, 2, 3, 4, 5];
const weekendRun = [0, 6];

type Field =
    | { kind: "all" }
    | { kind: "step"; every: number }
    | { kind: "values"; values: readonly number[] };

function plural(count: number, unit: string): string {
    return count === 1 ? `Every ${unit}` : `Every ${count} ${unit}s`;
}

function join(parts: readonly string[]): string {
    if (parts.length <= 1) {
        return parts[0] ?? "";
    }
    return `${parts.slice(0, -1).join(", ")} and ${parts[parts.length - 1]}`;
}

function ordinal(value: number): string {
    const tens = value % 100;
    if (tens >= 11 && tens <= 13) {
        return `${value}th`;
    }
    switch (value % 10) {
        case 1:
            return `${value}st`;
        case 2:
            return `${value}nd`;
        case 3:
            return `${value}rd`;
        default:
            return `${value}th`;
    }
}

function pad(value: number): string {
    return value < 10 ? `0${value}` : String(value);
}

function integer(raw: string, aliases: Readonly<Record<string, number>>): number | null {
    const named = aliases[raw.toUpperCase()];
    if (named !== undefined) {
        return named;
    }
    if (!/^\d{1,2}$/.test(raw)) {
        return null;
    }
    return Number(raw);
}

function expand(
    part: string,
    low: number,
    high: number,
    aliases: Readonly<Record<string, number>>,
): number[] | null {
    const [span, stepRaw] = part.split("/");
    if (span === undefined || part.split("/").length > 2) {
        return null;
    }
    let step = 1;
    if (stepRaw !== undefined) {
        if (!/^\d{1,2}$/.test(stepRaw) || Number(stepRaw) < 1) {
            return null;
        }
        step = Number(stepRaw);
    }
    let from = low;
    let to = high;
    if (span !== "*") {
        const bounds = span.split("-");
        if (bounds.length > 2) {
            return null;
        }
        const start = integer(bounds[0] ?? "", aliases);
        if (start === null) {
            return null;
        }
        from = start;
        to = start;
        if (bounds.length === 2) {
            const end = integer(bounds[1] ?? "", aliases);
            if (end === null) {
                return null;
            }
            to = end;
        }
    }
    if (from < low || to > high || from > to) {
        return null;
    }
    const out: number[] = [];
    for (let value = from; value <= to; value += step) {
        out.push(value);
    }
    return out;
}

function parseField(
    raw: string,
    low: number,
    high: number,
    aliases: Readonly<Record<string, number>> = {},
): Field | null {
    if (raw === "*") {
        return { kind: "all" };
    }
    const wholeStep = /^\*\/(\d{1,2})$/.exec(raw);
    if (wholeStep !== null) {
        const every = Number(wholeStep[1]);
        return every >= 1 && every <= high ? { kind: "step", every } : null;
    }
    const collected = new Set<number>();
    for (const part of raw.split(",")) {
        const values = expand(part, low, high, aliases);
        if (values === null) {
            return null;
        }
        for (const value of values) {
            collected.add(value);
        }
    }
    if (collected.size === 0) {
        return null;
    }
    return { kind: "values", values: [...collected].sort((left, right) => left - right) };
}

function valuesOf(field: Field): readonly number[] | null {
    return field.kind === "values" ? field.values : null;
}

function sameRun(values: readonly number[], run: readonly number[]): boolean {
    return values.length === run.length && values.every((value, at) => value === run[at]);
}

function clockTimes(hours: readonly number[], minutes: readonly number[]): string | null {
    const stamps: string[] = [];
    for (const hour of hours) {
        for (const minute of minutes) {
            stamps.push(`${pad(hour)}:${pad(minute)}`);
        }
    }
    return stamps.length > maxListedTimes ? null : join(stamps);
}

function weekdayPhrase(values: readonly number[]): string {
    if (sameRun(values, weekdayRun)) {
        return "weekday";
    }
    if (sameRun(values, weekendRun)) {
        return "weekend day";
    }
    return join(values.map((value) => weekdayNames[value] ?? String(value)));
}

function everySeconds(seconds: number): string | null {
    if (seconds <= 0 || !Number.isInteger(seconds)) {
        return null;
    }
    if (seconds % 60 !== 0) {
        return plural(seconds, "second");
    }
    return everyMinutes(seconds / 60);
}

function everyMinutes(minutes: number): string | null {
    if (minutes <= 0 || !Number.isInteger(minutes)) {
        return null;
    }
    if (minutes % 1440 === 0) {
        return plural(minutes / 1440, "day");
    }
    if (minutes % 60 === 0) {
        return plural(minutes / 60, "hour");
    }
    return plural(minutes, "minute");
}

function goDurationSeconds(raw: string): number | null {
    if (!/^(\d+(\.\d+)?(ns|us|ms|s|m|h))+$/.test(raw)) {
        return null;
    }
    let seconds = 0;
    for (const match of raw.matchAll(/(\d+(?:\.\d+)?)(ns|us|ms|s|m|h)/g)) {
        seconds += Number(match[1]) * (durationUnits[match[2] ?? ""] ?? 0);
    }
    return seconds;
}

function normaliseWeekdays(field: Field): Field {
    const values = valuesOf(field);
    if (values === null) {
        return field;
    }
    const mapped = new Set(values.map((value) => (value === 7 ? 0 : value)));
    return { kind: "values", values: [...mapped].sort((left, right) => left - right) };
}

function stepCadence(minute: Field, hour: Field, day: Field, month: Field, week: Field): string | null {
    if (day.kind !== "all" || month.kind !== "all" || week.kind !== "all") {
        return null;
    }
    if (minute.kind === "step" && hour.kind === "all") {
        return everyMinutes(minute.every);
    }
    if (minute.kind === "all" && hour.kind === "all") {
        return "Every minute";
    }
    const minutes = valuesOf(minute);
    if (minutes === null || minutes.length !== 1) {
        return null;
    }
    const only = minutes[0] ?? 0;
    const tail = only === 0 ? "" : ` at :${pad(only)}`;
    if (hour.kind === "step") {
        const words = everyMinutes(hour.every * 60);
        return words === null ? null : `${words}${tail}`;
    }
    if (hour.kind === "all") {
        return `Every hour${tail}`;
    }
    return null;
}

function clockCadence(minute: Field, hour: Field, day: Field, month: Field, week: Field): string | null {
    const minutes = valuesOf(minute);
    const hours = valuesOf(hour);
    if (minutes === null || hours === null) {
        return null;
    }
    const at = clockTimes(hours, minutes);
    if (at === null) {
        return null;
    }
    const tail = `at ${at} ${zone}`;
    if (month.kind === "all") {
        if (day.kind === "all" && week.kind === "all") {
            return `Every day ${tail}`;
        }
        if (day.kind === "all") {
            const weekdays = valuesOf(week);
            return weekdays === null ? null : `Every ${weekdayPhrase(weekdays)} ${tail}`;
        }
        if (week.kind !== "all") {
            return null;
        }
        if (day.kind === "step") {
            const words = everyMinutes(day.every * 1440);
            return words === null ? null : `${words} ${tail}`;
        }
        const days = valuesOf(day);
        if (days === null || days.length > maxListedTimes) {
            return null;
        }
        return `Every month on the ${join(days.map(ordinal))} ${tail}`;
    }
    const days = valuesOf(day);
    const months = valuesOf(month);
    if (days === null || months === null || week.kind !== "all") {
        return null;
    }
    if (days.length !== 1 || months.length !== 1) {
        return null;
    }
    return `Every year on ${days[0]} ${monthNames[(months[0] ?? 1) - 1]} ${tail}`;
}

export function cadenceWords(cron: string, intervalMinutes: number): string {
    if (intervalMinutes > 0) {
        return everyMinutes(intervalMinutes) ?? "";
    }
    const trimmed = cron.trim();
    if (trimmed === "") {
        return "";
    }
    if (trimmed.startsWith("@every ")) {
        const seconds = goDurationSeconds(trimmed.slice("@every ".length).trim());
        return seconds === null ? trimmed : (everySeconds(seconds) ?? trimmed);
    }
    const expression = descriptors[trimmed.toLowerCase()] ?? trimmed;
    const fields = expression.split(/\s+/);
    if (fields.length !== 5) {
        return trimmed;
    }
    const minute = parseField(fields[0] ?? "", 0, 59);
    const hour = parseField(fields[1] ?? "", 0, 23);
    const day = parseField(fields[2] ?? "", 1, 31);
    const month = parseField(fields[3] ?? "", 1, 12, monthAliases);
    const week = parseField(fields[4] ?? "", 0, 7, weekdayAliases);
    if (minute === null || hour === null || day === null || month === null || week === null) {
        return trimmed;
    }
    const weekdays = normaliseWeekdays(week);
    return (
        stepCadence(minute, hour, day, month, weekdays) ??
        clockCadence(minute, hour, day, month, weekdays) ??
        trimmed
    );
}
