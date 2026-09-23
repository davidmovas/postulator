export type Span =
    | { kind: "text"; text: string }
    | { kind: "code"; text: string }
    | { kind: "break" }
    | { kind: "strong"; spans: readonly Span[] }
    | { kind: "em"; spans: readonly Span[] }
    | { kind: "strike"; spans: readonly Span[] }
    | { kind: "link"; href: string; spans: readonly Span[] };

export type Align = "left" | "center" | "right";

export type Cell = readonly Span[];

export interface Item {
    spans: readonly Span[];
    blocks: readonly Block[];
}

export type Block =
    | { kind: "heading"; level: number; spans: readonly Span[] }
    | { kind: "paragraph"; spans: readonly Span[] }
    | { kind: "list"; ordered: boolean; start: number; items: readonly Item[] }
    | { kind: "code"; language: string; text: string }
    | { kind: "quote"; blocks: readonly Block[] }
    | { kind: "rule" }
    | { kind: "table"; align: readonly Align[]; head: readonly Cell[]; rows: readonly (readonly Cell[])[] };

const fence = /^(\s*)(```|~~~)\s*([^\s`]*)/;
const heading = /^ {0,3}(#{1,6})(?:\s+(.*))?$/;
const rule = /^ {0,3}(?:(?:-\s*){3,}|(?:\*\s*){3,}|(?:_\s*){3,})$/;
const quote = /^ {0,3}>\s?(.*)$/;
const bullet = /^(\s*)([-*+])\s+(.*)$/;
const ordered = /^(\s*)(\d{1,9})([.)])\s+(.*)$/;
const divider = /^\s*\|?(?:\s*:?-+:?\s*\|)*\s*:?-+:?\s*\|?\s*$/;
const setextH1 = /^ {0,3}=+\s*$/;
const setextH2 = /^ {0,3}-+\s*$/;
const hardBreak = /(?: {2,}|\\)$/;
const wordLike = /[\p{L}\p{N}_]/u;
const bareLink = /^https?:\/\/[^\s<>[\]()]+/;
const trailingPunctuation = /[.,;:!?)\]]+$/;

const emphasisMarkers = ["***", "___", "**", "__", "~~", "*", "_"] as const;

function marked(marker: string): "strong" | "em" | "strike" {
    if (marker === "~~") {
        return "strike";
    }
    return marker.length >= 2 ? "strong" : "em";
}

function closes(source: string, marker: string, from: number): number {
    let at = from;
    while (at <= source.length - marker.length) {
        if (source[at] === "\\") {
            at += 2;
            continue;
        }
        if (source.startsWith(marker, at)) {
            let run = 0;
            while (source[at + run] === marker[0]) {
                run += 1;
            }
            return run > marker.length ? at + (run - marker.length) : at;
        }
        at += 1;
    }
    return -1;
}

function looseEnough(source: string, marker: string, open: number, close: number): boolean {
    if (marker[0] !== "_") {
        return true;
    }
    const before = open === 0 ? "" : source[open - 1];
    const after = source[close + marker.length] ?? "";
    return !wordLike.test(before) && !wordLike.test(after);
}

function codeSpan(source: string, index: number): { text: string; length: number } | null {
    let ticks = 0;
    while (source[index + ticks] === "`") {
        ticks += 1;
    }
    if (ticks === 0) {
        return null;
    }

    const opener = "`".repeat(ticks);
    const from = index + ticks;
    let at = from;
    while (at <= source.length - ticks) {
        if (source.startsWith(opener, at) && source[at + ticks] !== "`") {
            return { text: source.slice(from, at).trim(), length: at + ticks - index };
        }
        at += 1;
    }
    return null;
}

function bracketed(source: string, index: number): { label: string; href: string; length: number } | null {
    if (source[index] !== "[") {
        return null;
    }

    let depth = 0;
    let at = index;
    while (at < source.length) {
        const here = source[at];
        if (here === "\\") {
            at += 2;
            continue;
        }
        if (here === "[") {
            depth += 1;
        } else if (here === "]") {
            depth -= 1;
            if (depth === 0) {
                break;
            }
        }
        at += 1;
    }
    if (depth !== 0 || source[at + 1] !== "(") {
        return null;
    }

    const close = source.indexOf(")", at + 2);
    if (close === -1) {
        return null;
    }

    const inside = source.slice(at + 2, close).trim();
    const href = inside.split(/\s+/, 1)[0] ?? "";
    return { label: source.slice(index + 1, at), href, length: close + 1 - index };
}

export function spansOf(source: string): readonly Span[] {
    const out: Span[] = [];
    let plain = "";

    const flush = (): void => {
        if (plain !== "") {
            out.push({ kind: "text", text: plain });
            plain = "";
        }
    };

    let index = 0;
    while (index < source.length) {
        const here = source[index];

        if (here === "\\" && index + 1 < source.length) {
            plain += source[index + 1];
            index += 2;
            continue;
        }
        if (here === "\n") {
            flush();
            out.push({ kind: "break" });
            index += 1;
            continue;
        }
        if (here === "`") {
            const code = codeSpan(source, index);
            if (code !== null) {
                flush();
                out.push({ kind: "code", text: code.text });
                index += code.length;
                continue;
            }
        }
        if (here === "<") {
            const close = source.indexOf(">", index);
            const inside = close === -1 ? "" : source.slice(index + 1, close);
            if (bareLink.test(inside)) {
                flush();
                out.push({ kind: "link", href: inside, spans: [{ kind: "text", text: inside }] });
                index = close + 1;
                continue;
            }
        }
        if (here === "!" && source[index + 1] === "[") {
            const image = bracketed(source, index + 1);
            if (image !== null) {
                flush();
                out.push(...spansOf(image.label));
                index += image.length + 1;
                continue;
            }
        }
        if (here === "[") {
            const link = bracketed(source, index);
            if (link !== null) {
                flush();
                out.push({
                    kind: "link",
                    href: link.href,
                    spans: link.label === "" ? [{ kind: "text", text: link.href }] : spansOf(link.label),
                });
                index += link.length;
                continue;
            }
        }
        if (here === "h" && bareLink.test(source.slice(index))) {
            const matched = bareLink.exec(source.slice(index));
            if (matched !== null) {
                const href = matched[0].replace(trailingPunctuation, "");
                flush();
                out.push({ kind: "link", href, spans: [{ kind: "text", text: href }] });
                index += href.length;
                continue;
            }
        }

        const emphasis = emphasisAt(source, index);
        if (emphasis !== null) {
            flush();
            out.push(emphasis.span);
            index += emphasis.length;
            continue;
        }

        plain += here;
        index += 1;
    }

    flush();
    return out;
}

function emphasisAt(source: string, index: number): { span: Span; length: number } | null {
    for (const marker of emphasisMarkers) {
        if (!source.startsWith(marker, index)) {
            continue;
        }
        const from = index + marker.length;
        if (source[from] === undefined || /\s/.test(source[from])) {
            continue;
        }

        const close = closes(source, marker, from);
        if (close === -1 || close === from || !looseEnough(source, marker, index, close)) {
            continue;
        }

        const inner = source.slice(from, close);
        const length = close + marker.length - index;
        if (marker === "***" || marker === "___") {
            return {
                span: { kind: "strong", spans: [{ kind: "em", spans: spansOf(inner) }] },
                length,
            };
        }
        return { span: { kind: marked(marker), spans: spansOf(inner) }, length };
    }
    return null;
}

function alignOf(line: string): readonly Align[] {
    return cellTexts(line).map((cell) => {
        const left = cell.startsWith(":");
        const right = cell.endsWith(":");
        if (left && right) {
            return "center";
        }
        return right ? "right" : "left";
    });
}

function cellTexts(line: string): string[] {
    const trimmed = line.trim().replace(/^\|/, "").replace(/\|$/, "");
    const cells: string[] = [];
    let held = "";
    for (let index = 0; index < trimmed.length; index += 1) {
        if (trimmed[index] === "\\" && trimmed[index + 1] === "|") {
            held += "|";
            index += 1;
            continue;
        }
        if (trimmed[index] === "|") {
            cells.push(held.trim());
            held = "";
            continue;
        }
        held += trimmed[index];
    }
    cells.push(held.trim());
    return cells;
}

function cellsOf(line: string): Cell[] {
    return cellTexts(line).map((cell) => spansOf(cell));
}

function paragraphText(lines: readonly string[]): string {
    const out: string[] = [];
    for (let at = 0; at < lines.length; at += 1) {
        const line = lines[at];
        out.push(line.trim());
        if (at === lines.length - 1) {
            continue;
        }
        out.push(hardBreak.test(line) ? "\n" : " ");
    }
    return out.join("").replaceAll(/[ \t]*\n[ \t]*/g, "\n");
}

interface Pending {
    paragraph: string[];
}

function settle(pending: Pending, blocks: Block[]): void {
    if (pending.paragraph.length === 0) {
        return;
    }
    blocks.push({ kind: "paragraph", spans: spansOf(paragraphText(pending.paragraph)) });
    pending.paragraph = [];
}

function indentOf(line: string): number {
    return line.replaceAll("\t", "  ").length - line.replaceAll("\t", "  ").trimStart().length;
}

function dedent(lines: readonly string[], by: number): string[] {
    return lines.map((line) => {
        const widened = line.replaceAll("\t", "  ");
        return widened.slice(Math.min(by, indentOf(widened)));
    });
}

function itemOf(first: string, rest: readonly string[], indent: number): Item {
    const blocks = blocksOf([first, ...dedent(rest, indent)].join("\n"));
    if (blocks.length > 0 && blocks[0].kind === "paragraph") {
        return { spans: blocks[0].spans, blocks: blocks.slice(1) };
    }
    return { spans: [], blocks };
}

function listAt(lines: readonly string[], from: number): { block: Block; next: number } | null {
    const opener = ordered.exec(lines[from]) ?? bullet.exec(lines[from]);
    if (opener === null) {
        return null;
    }

    const isOrdered = ordered.test(lines[from]);
    const start = isOrdered ? Number.parseInt(opener[2], 10) : 1;
    const outer = opener[1].replaceAll("\t", "  ").length;
    const items: Item[] = [];

    let index = from;
    while (index < lines.length) {
        const marker = ordered.exec(lines[index]) ?? bullet.exec(lines[index]);
        if (marker === null || marker[1].replaceAll("\t", "  ").length !== outer) {
            break;
        }
        if (ordered.test(lines[index]) !== isOrdered) {
            break;
        }

        const content = isOrdered ? marker[4] : marker[3];
        const width = lines[index].replaceAll("\t", "  ").indexOf(content, outer);
        const rest: string[] = [];
        index += 1;
        while (index < lines.length) {
            const line = lines[index];
            if (line.trim() === "") {
                const following = lines[index + 1] ?? "";
                if (following.trim() === "" || indentOf(following) <= outer) {
                    break;
                }
                rest.push("");
                index += 1;
                continue;
            }
            if (indentOf(line) <= outer) {
                break;
            }
            rest.push(line);
            index += 1;
        }
        items.push(itemOf(content, rest, width > outer ? width : outer + 2));
    }

    return { block: { kind: "list", ordered: isOrdered, start, items }, next: index };
}

export function blocksOf(source: string): readonly Block[] {
    const lines = source.replaceAll("\r\n", "\n").split("\n");
    const blocks: Block[] = [];
    const pending: Pending = { paragraph: [] };

    let index = 0;
    while (index < lines.length) {
        const line = lines[index];

        const open = fence.exec(line);
        if (open !== null) {
            settle(pending, blocks);
            const closer = open[2];
            const body: string[] = [];
            index += 1;
            while (index < lines.length && !lines[index].trimStart().startsWith(closer)) {
                body.push(lines[index]);
                index += 1;
            }
            index += 1;
            blocks.push({ kind: "code", language: open[3], text: body.join("\n") });
            continue;
        }

        if (line.trim() === "") {
            settle(pending, blocks);
            index += 1;
            continue;
        }

        if (pending.paragraph.length > 0 && setextH1.test(line)) {
            blocks.push({ kind: "heading", level: 1, spans: spansOf(paragraphText(pending.paragraph)) });
            pending.paragraph = [];
            index += 1;
            continue;
        }
        if (pending.paragraph.length > 0 && setextH2.test(line) && !line.includes(" ")) {
            blocks.push({ kind: "heading", level: 2, spans: spansOf(paragraphText(pending.paragraph)) });
            pending.paragraph = [];
            index += 1;
            continue;
        }

        const titled = heading.exec(line);
        if (titled !== null) {
            settle(pending, blocks);
            blocks.push({ kind: "heading", level: titled[1].length, spans: spansOf((titled[2] ?? "").trim()) });
            index += 1;
            continue;
        }

        if (rule.test(line)) {
            settle(pending, blocks);
            blocks.push({ kind: "rule" });
            index += 1;
            continue;
        }

        const next = lines[index + 1] ?? "";
        if (line.includes("|") && next.includes("-") && divider.test(next)) {
            settle(pending, blocks);
            const head = cellsOf(line);
            const align = alignOf(next);
            const rows: Cell[][] = [];
            index += 2;
            while (index < lines.length && lines[index].includes("|") && lines[index].trim() !== "") {
                rows.push(cellsOf(lines[index]));
                index += 1;
            }
            blocks.push({ kind: "table", align, head, rows });
            continue;
        }

        if (quote.test(line)) {
            settle(pending, blocks);
            const body: string[] = [];
            while (index < lines.length) {
                const quoted = quote.exec(lines[index]);
                if (quoted === null) {
                    if (lines[index].trim() === "" || body.length === 0) {
                        break;
                    }
                    body.push(lines[index].trim());
                    index += 1;
                    continue;
                }
                body.push(quoted[1]);
                index += 1;
            }
            blocks.push({ kind: "quote", blocks: blocksOf(body.join("\n")) });
            continue;
        }

        const list = listAt(lines, index);
        if (list !== null) {
            settle(pending, blocks);
            blocks.push(list.block);
            index = list.next;
            continue;
        }

        pending.paragraph.push(line);
        index += 1;
    }

    settle(pending, blocks);
    return blocks;
}
