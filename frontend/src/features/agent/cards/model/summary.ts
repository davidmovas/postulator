const encoded = /[{}[\]]|"\s*:/;

export function ownSummary(summary: string, description: string | null): string | null {
    const said = summary.trim();
    if (said === "" || encoded.test(said)) {
        return null;
    }
    const generic = (description ?? "").trim().replace(/\.$/, "");
    if (generic !== "" && (said === generic || said.startsWith(`${generic} with `))) {
        return null;
    }
    return said;
}
