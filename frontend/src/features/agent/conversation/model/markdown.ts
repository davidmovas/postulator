export type Span =
    | { kind: "text"; text: string }
    | { kind: "strong"; text: string }
    | { kind: "em"; text: string }
    | { kind: "code"; text: string }
    | { kind: "link"; text: string; href: string };

export interface Item {
    depth: number;
    marker: string;
    spans: readonly Span[];
}

export type Block =
    | { kind: "heading"; level: number; spans: readonly Span[] }
    | { kind: "paragraph"; spans: readonly Span[] }
    | { kind: "list"; ordered: boolean; items: readonly Item[] }
    | { kind: "code"; language: string; text: string }
    | { kind: "quote"; spans: readonly Span[] }
    | { kind: "rule" }
    | { kind: "table"; head: readonly (readonly Span[])[]; rows: readonly (readonly (readonly Span[])[])[] };

const fence = /^\s*```(\w*)\s*$/;
const heading = /^(#{1,6})\s+(.*)$/;
const rule = /^(?:-{3,}|\*{3,}|_{3,})\s*$/;
const quote = /^>\s?(.*)$/;
const bullet = /^(\s*)[-*+]\s+(.*)$/;
const ordered = /^(\s*)(\d+)[.)]\s+(.*)$/;
const divider = /^\s*\|?[\s:-]*-[\s:|-]*\|?\s*$/;
const indentPerLevel = 2;
const maxDepth = 4;

const inlineCode = /^`([^`]+)`/;
const inlineStrong = /^\*\*([^*]+)\*\*/;
const inlineEm = /^\*([^*\s][^*]*)\*/;
const inlineLink = /^\[([^\]]*)\]\(([^)\s]+)\)/;

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
        const rest = source.slice(index);

        const code = inlineCode.exec(rest);
        if (code !== null) {
            flush();
            out.push({ kind: "code", text: code[1] });
            index += code[0].length;
            continue;
        }

        const strong = inlineStrong.exec(rest);
        if (strong !== null) {
            flush();
            out.push({ kind: "strong", text: strong[1] });
            index += strong[0].length;
            continue;
        }

        const emphasis = inlineEm.exec(rest);
        if (emphasis !== null) {
            flush();
            out.push({ kind: "em", text: emphasis[1] });
            index += emphasis[0].length;
            continue;
        }

        const link = inlineLink.exec(rest);
        if (link !== null) {
            flush();
            out.push({ kind: "link", text: link[1] === "" ? link[2] : link[1], href: link[2] });
            index += link[0].length;
            continue;
        }

        plain += source[index];
        index += 1;
    }

    flush();
    return out;
}

function cellsOf(line: string): readonly (readonly Span[])[] {
    return line
        .trim()
        .replace(/^\|/, "")
        .replace(/\|$/, "")
        .split("|")
        .map((cell) => spansOf(cell.trim()));
}

function depthOf(indent: string): number {
    return Math.min(Math.floor(indent.replace(/\t/g, "  ").length / indentPerLevel), maxDepth);
}

interface Pending {
    paragraph: string[];
    list: { ordered: boolean; items: Item[] } | null;
}

function settle(pending: Pending, blocks: Block[]): void {
    if (pending.paragraph.length > 0) {
        blocks.push({ kind: "paragraph", spans: spansOf(pending.paragraph.join(" ")) });
        pending.paragraph = [];
    }
    if (pending.list !== null) {
        blocks.push({ kind: "list", ordered: pending.list.ordered, items: pending.list.items });
        pending.list = null;
    }
}

function item(pending: Pending, blocks: Block[], isOrdered: boolean, next: Item): void {
    if (pending.paragraph.length > 0) {
        settle(pending, blocks);
    }
    if (pending.list !== null && pending.list.ordered !== isOrdered) {
        settle(pending, blocks);
    }
    if (pending.list === null) {
        pending.list = { ordered: isOrdered, items: [] };
    }
    pending.list.items.push(next);
}

export function blocksOf(source: string): readonly Block[] {
    const lines = source.replaceAll("\r\n", "\n").split("\n");
    const blocks: Block[] = [];
    const pending: Pending = { paragraph: [], list: null };

    let index = 0;
    while (index < lines.length) {
        const line = lines[index];

        const open = fence.exec(line);
        if (open !== null) {
            settle(pending, blocks);
            const body: string[] = [];
            index += 1;
            while (index < lines.length && fence.exec(lines[index]) === null) {
                body.push(lines[index]);
                index += 1;
            }
            index += 1;
            blocks.push({ kind: "code", language: open[1], text: body.join("\n") });
            continue;
        }

        if (line.trim() === "") {
            settle(pending, blocks);
            index += 1;
            continue;
        }

        const titled = heading.exec(line);
        if (titled !== null) {
            settle(pending, blocks);
            blocks.push({ kind: "heading", level: titled[1].length, spans: spansOf(titled[2].trim()) });
            index += 1;
            continue;
        }

        if (rule.test(line)) {
            settle(pending, blocks);
            blocks.push({ kind: "rule" });
            index += 1;
            continue;
        }

        if (line.includes("|") && index + 1 < lines.length && divider.test(lines[index + 1]) && lines[index + 1].includes("-")) {
            settle(pending, blocks);
            const head = cellsOf(line);
            const rows: (readonly (readonly Span[])[])[] = [];
            index += 2;
            while (index < lines.length && lines[index].includes("|") && lines[index].trim() !== "") {
                rows.push(cellsOf(lines[index]));
                index += 1;
            }
            blocks.push({ kind: "table", head, rows });
            continue;
        }

        const quoted = quote.exec(line);
        if (quoted !== null) {
            settle(pending, blocks);
            blocks.push({ kind: "quote", spans: spansOf(quoted[1].trim()) });
            index += 1;
            continue;
        }

        const numbered = ordered.exec(line);
        if (numbered !== null) {
            item(pending, blocks, true, { depth: depthOf(numbered[1]), marker: `${numbered[2]}.`, spans: spansOf(numbered[3]) });
            index += 1;
            continue;
        }

        const listed = bullet.exec(line);
        if (listed !== null) {
            item(pending, blocks, false, { depth: depthOf(listed[1]), marker: "", spans: spansOf(listed[2]) });
            index += 1;
            continue;
        }

        if (pending.list !== null && line.startsWith(" ")) {
            const last = pending.list.items[pending.list.items.length - 1];
            pending.list.items[pending.list.items.length - 1] = {
                ...last,
                spans: [...last.spans, { kind: "text", text: ` ${line.trim()}` }],
            };
            index += 1;
            continue;
        }

        if (pending.list !== null) {
            settle(pending, blocks);
        }
        pending.paragraph.push(line.trim());
        index += 1;
    }

    settle(pending, blocks);
    return blocks;
}
