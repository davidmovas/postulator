const localSuffixes = [".local", ".localhost", ".test"];

function octets(host: string): number[] | null {
    const parts = host.split(".");
    if (parts.length !== 4) {
        return null;
    }
    const parsed = parts.map((part) => (/^\d{1,3}$/.test(part) ? Number(part) : -1));
    return parsed.every((value) => value >= 0 && value <= 255) ? parsed : null;
}

function localIPv4(parts: number[]): boolean {
    const [first, second] = parts as [number, number, number, number];
    if (first === 127) {
        return true;
    }
    if (first === 10) {
        return true;
    }
    if (first === 172 && second >= 16 && second <= 31) {
        return true;
    }
    if (first === 192 && second === 168) {
        return true;
    }
    return first === 169 && second === 254;
}

function localIPv6(host: string): boolean {
    if (host === "::1") {
        return true;
    }
    const leading = host.split(":")[0] ?? "";
    if (leading.length < 2) {
        return false;
    }
    const prefix = leading.slice(0, 2);
    return prefix === "fc" || prefix === "fd" || prefix === "fe";
}

export function localAddress(raw: string): boolean {
    const trimmed = raw.trim();
    if (trimmed === "") {
        return false;
    }

    let host: string;
    try {
        host = new URL(trimmed).hostname.toLowerCase();
    } catch {
        return false;
    }
    if (host.startsWith("[") && host.endsWith("]")) {
        host = host.slice(1, -1);
    }
    if (host === "" || host === "localhost") {
        return host === "localhost";
    }

    const parts = octets(host);
    if (parts !== null) {
        return localIPv4(parts);
    }
    if (host.includes(":")) {
        return localIPv6(host);
    }
    return localSuffixes.some((suffix) => host.endsWith(suffix));
}
