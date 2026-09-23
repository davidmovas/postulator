import { readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const frontend = resolve(here, "..");
const listPath = resolve(here, "icons.txt");
const packDir = resolve(frontend, "node_modules/@material-symbols/svg-300/rounded");
const outPath = resolve(frontend, "src/ui/icons/generated.tsx");

const names = readFileSync(listPath, "utf8")
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0);

const duplicates = names.filter((name, index) => names.indexOf(name) !== index);
if (duplicates.length > 0) {
    throw new Error(`icons.txt lists a name twice: ${duplicates.join(", ")}`);
}

const sorted = [...names].sort();
if (sorted.join("\n") !== names.join("\n")) {
    throw new Error("icons.txt must be sorted so the generated module has a stable order");
}

function componentName(name) {
    const parts = name.split("_");
    for (const part of parts) {
        if (!/^[a-z0-9]+$/.test(part)) {
            throw new Error(`icon name is not snake_case ascii: ${name}`);
        }
    }
    return `${parts.map((part) => part[0].toUpperCase() + part.slice(1)).join("")}Icon`;
}

function pathOf(name) {
    let raw;
    try {
        raw = readFileSync(resolve(packDir, `${name}.svg`), "utf8");
    } catch {
        throw new Error(`icon "${name}" is not in @material-symbols/svg-300/rounded`);
    }
    const viewBox = /viewBox="([^"]+)"/.exec(raw);
    if (viewBox === null || viewBox[1] !== "0 -960 960 960") {
        throw new Error(`icon "${name}" has an unexpected viewBox`);
    }
    const inner = raw.trim().replace(/^<svg[^>]*>/, "").replace(/<\/svg>$/, "");
    const single = /^<path d="([^"]+)"\/>$/.exec(inner);
    if (single === null) {
        throw new Error(`icon "${name}" is not a single path and needs the generator extended`);
    }
    return single[1];
}

const seen = new Map();
const lines = ['import { createIcon } from "./create-icon.js";', ""];
for (const name of names) {
    const component = componentName(name);
    const clash = seen.get(component);
    if (clash !== undefined) {
        throw new Error(`"${name}" and "${clash}" both render as ${component}`);
    }
    seen.set(component, name);
    lines.push(`export const ${component} = createIcon("${pathOf(name)}");`);
}

writeFileSync(outPath, `${lines.join("\n")}\n`, "utf8");
process.stdout.write(`${names.length} icons written to ${outPath}\n`);
