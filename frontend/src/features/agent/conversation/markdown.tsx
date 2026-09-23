import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { useOpenExternal } from "../../../data/hooks/browser.js";
import { cx } from "../../../ui/index.js";
import type { Align, Block, Cell, Item, Span } from "./model/markdown.js";
import { blocksOf } from "./model/markdown.js";

const said = copy.agent.transcript;

const headingClasses: Readonly<Record<number, string>> = {
    1: "text-base font-semibold text-ink",
    2: "text-base font-semibold text-ink",
    3: "text-sm font-semibold text-ink",
    4: "text-xs font-semibold tracking-label text-ink-soft uppercase",
    5: "text-xs font-semibold tracking-label text-ink-soft uppercase",
    6: "text-xs font-semibold tracking-label text-ink-soft uppercase",
};

const alignClasses: Readonly<Record<Align, string>> = {
    left: "text-left",
    center: "text-center",
    right: "text-right",
};

function Link({ href, spans }: { href: string; spans: readonly Span[] }): ReactElement {
    const open = useOpenExternal();
    const reachable = href.startsWith("http://") || href.startsWith("https://");

    if (!reachable) {
        return (
            <span className="text-ink-soft" title={href}>
                <Spans spans={spans} />
            </span>
        );
    }
    return (
        <button
            type="button"
            className="text-accent underline decoration-dotted hover:decoration-solid"
            title={href}
            disabled={open.isPending}
            onClick={() => {
                open.mutate({ url: href });
            }}
        >
            <Spans spans={spans} />
        </button>
    );
}

function Spans({ spans }: { spans: readonly Span[] }): ReactElement {
    return (
        <>
            {spans.map((span, index) => {
                const key = `${span.kind}-${index}`;
                switch (span.kind) {
                    case "strong":
                        return (
                            <strong key={key} className="font-semibold text-ink">
                                <Spans spans={span.spans} />
                            </strong>
                        );
                    case "em":
                        return (
                            <em key={key} className="italic">
                                <Spans spans={span.spans} />
                            </em>
                        );
                    case "strike":
                        return (
                            <s key={key} className="text-ink-dim">
                                <Spans spans={span.spans} />
                            </s>
                        );
                    case "code":
                        return (
                            <code key={key} className="rounded-sm bg-inset px-1 font-mono text-xs text-ink">
                                {span.text}
                            </code>
                        );
                    case "link":
                        return <Link key={key} href={span.href} spans={span.spans} />;
                    case "break":
                        return <br key={key} />;
                    default:
                        return <span key={key}>{span.text}</span>;
                }
            })}
        </>
    );
}

function Marker({ ordered, at, start }: { ordered: boolean; at: number; start: number }): ReactElement {
    if (!ordered) {
        return <span aria-hidden={true} className="mt-2 ml-1.5 h-1 w-1 shrink-0 rounded-full bg-ink-faint" />;
    }
    return (
        <span className="w-5 shrink-0 text-right font-mono text-xs text-ink-dim tabular-nums">{start + at}.</span>
    );
}

function ListItem({ item, ordered, at, start }: { item: Item; ordered: boolean; at: number; start: number }): ReactElement {
    return (
        <li className="flex gap-2">
            <Marker ordered={ordered} at={at} start={start} />
            <div className="flex min-w-0 flex-1 flex-col gap-1">
                {item.spans.length > 0 ? (
                    <span>
                        <Spans spans={item.spans} />
                    </span>
                ) : null}
                {item.blocks.map((block, index) => (
                    <Rendered key={`n-${index}`} block={block} />
                ))}
            </div>
        </li>
    );
}

function Code({ language, text }: { language: string; text: string }): ReactElement {
    const [copied, setCopied] = useState(false);

    return (
        <div className="overflow-hidden rounded-md border border-hairline bg-inset">
            <div className="flex h-6 items-center justify-between border-b border-hairline px-2">
                <span className="text-2xs tracking-label text-ink-dim uppercase">
                    {language === "" ? said.plainCode : language}
                </span>
                <button
                    type="button"
                    className="text-2xs tracking-label text-ink-dim uppercase hover:text-ink-soft"
                    onClick={() => {
                        void navigator.clipboard.writeText(text).then(() => {
                            setCopied(true);
                        });
                    }}
                >
                    {copied ? said.codeCopied : said.copyCode}
                </button>
            </div>
            <pre className="overflow-x-auto px-2 py-1.5">
                <code className="font-mono text-xs whitespace-pre text-ink">{text}</code>
            </pre>
        </div>
    );
}

function Row({ cells, align, head }: { cells: readonly Cell[]; align: readonly Align[]; head: boolean }): ReactElement {
    return (
        <tr className={head ? "border-b border-hairline bg-inset" : "border-b border-inset last:border-b-0"}>
            {cells.map((cell, index) =>
                head ? (
                    <th
                        key={`h-${index}`}
                        className={cx("px-2 py-1 font-semibold text-ink-soft", alignClasses[align[index] ?? "left"])}
                    >
                        <Spans spans={cell} />
                    </th>
                ) : (
                    <td
                        key={`c-${index}`}
                        className={cx("px-2 py-1 align-top text-ink-soft", alignClasses[align[index] ?? "left"])}
                    >
                        <Spans spans={cell} />
                    </td>
                ),
            )}
        </tr>
    );
}

function Table({ block }: { block: Extract<Block, { kind: "table" }> }): ReactElement {
    return (
        <div className="overflow-x-auto rounded-md border border-hairline">
            <table className="w-full border-collapse text-xs">
                <thead>
                    <Row cells={block.head} align={block.align} head={true} />
                </thead>
                <tbody>
                    {block.rows.map((row, index) => (
                        <Row key={`r-${index}`} cells={row} align={block.align} head={false} />
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function Rendered({ block, trailing }: { block: Block; trailing?: ReactElement | null }): ReactElement {
    switch (block.kind) {
        case "heading":
            return (
                <p className={cx("mt-1 first:mt-0", headingClasses[block.level])}>
                    <Spans spans={block.spans} />
                    {trailing ?? null}
                </p>
            );
        case "list": {
            const items = block.items.map((item, index) => (
                <ListItem key={`i-${index}`} item={item} ordered={block.ordered} at={index} start={block.start} />
            ));
            return block.ordered ? (
                <ol className="flex flex-col gap-1">{items}</ol>
            ) : (
                <ul className="flex flex-col gap-1">{items}</ul>
            );
        }
        case "code":
            return <Code language={block.language} text={block.text} />;
        case "quote":
            return (
                <blockquote className="flex flex-col gap-1 border-l-2 border-edge pl-2.5 text-ink-dim italic">
                    {block.blocks.map((nested, index) => (
                        <Rendered key={`q-${index}`} block={nested} />
                    ))}
                </blockquote>
            );
        case "rule":
            return <hr className="border-hairline" />;
        case "table":
            return <Table block={block} />;
        default:
            return (
                <p>
                    <Spans spans={block.spans} />
                    {trailing ?? null}
                </p>
            );
    }
}

export interface MarkdownProps {
    text: string;
    trailing?: ReactElement | null;
}

function carries(block: Block): boolean {
    return block.kind === "paragraph" || block.kind === "heading";
}

export function Markdown({ text, trailing }: MarkdownProps): ReactElement {
    const blocks = useMemo(() => blocksOf(text), [text]);
    const last = blocks.length - 1;
    const inline = trailing !== undefined && trailing !== null && last >= 0 && carries(blocks[last]);

    return (
        <div className="flex flex-col gap-2 text-sm leading-relaxed text-ink-soft">
            {blocks.map((block, index) => (
                <Rendered key={`b-${index}`} block={block} trailing={inline && index === last ? trailing : null} />
            ))}
            {inline ? null : (trailing ?? null)}
        </div>
    );
}
