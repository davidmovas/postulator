import type { ReactElement } from "react";
import { useMemo } from "react";

import { cx } from "../../../ui/index.js";
import type { Block, Item, Span } from "./model/markdown.js";
import { blocksOf } from "./model/markdown.js";

const indentPx = 14;

const headingClasses: Readonly<Record<number, string>> = {
    1: "text-base font-semibold text-ink",
    2: "text-base font-semibold text-ink",
    3: "text-sm font-semibold text-ink",
    4: "text-xs font-semibold tracking-label text-ink-soft uppercase",
    5: "text-xs font-semibold tracking-label text-ink-soft uppercase",
    6: "text-xs font-semibold tracking-label text-ink-soft uppercase",
};

function Spans({ spans }: { spans: readonly Span[] }): ReactElement {
    return (
        <>
            {spans.map((span, index) => {
                const key = `${span.kind}-${index}`;
                switch (span.kind) {
                    case "strong":
                        return (
                            <strong key={key} className="font-semibold text-ink">
                                {span.text}
                            </strong>
                        );
                    case "em":
                        return (
                            <em key={key} className="italic">
                                {span.text}
                            </em>
                        );
                    case "code":
                        return (
                            <code key={key} className="rounded-sm bg-inset px-1 font-mono text-xs text-ink">
                                {span.text}
                            </code>
                        );
                    case "link":
                        return (
                            <span key={key} className="text-accent underline decoration-dotted" title={span.href}>
                                {span.text}
                            </span>
                        );
                    default:
                        return <span key={key}>{span.text}</span>;
                }
            })}
        </>
    );
}

function ListItem({ item, ordered }: { item: Item; ordered: boolean }): ReactElement {
    return (
        <li className="flex gap-2" style={{ paddingLeft: item.depth * indentPx }}>
            {ordered ? (
                <span className="w-5 shrink-0 text-right font-mono text-xs text-ink-dim tabular-nums">{item.marker}</span>
            ) : (
                <span aria-hidden={true} className="mt-2 ml-1.5 h-1 w-1 shrink-0 rounded-full bg-ink-faint" />
            )}
            <span className="min-w-0 flex-1">
                <Spans spans={item.spans} />
            </span>
        </li>
    );
}

function Table({ block }: { block: Extract<Block, { kind: "table" }> }): ReactElement {
    return (
        <div className="overflow-x-auto rounded-md border border-hairline">
            <table className="w-full border-collapse text-xs">
                <thead>
                    <tr className="border-b border-hairline bg-inset">
                        {block.head.map((cell, index) => (
                            <th key={`h-${index}`} className="px-2 py-1 text-left font-semibold text-ink-soft">
                                <Spans spans={cell} />
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {block.rows.map((row, rowIndex) => (
                        <tr key={`r-${rowIndex}`} className="border-b border-inset last:border-b-0">
                            {row.map((cell, cellIndex) => (
                                <td key={`c-${cellIndex}`} className="px-2 py-1 align-top text-ink-soft">
                                    <Spans spans={cell} />
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function Rendered({ block }: { block: Block }): ReactElement {
    switch (block.kind) {
        case "heading":
            return (
                <p className={cx("mt-1 first:mt-0", headingClasses[block.level])}>
                    <Spans spans={block.spans} />
                </p>
            );
        case "list":
            return (
                <ul className="flex flex-col gap-1">
                    {block.items.map((item, index) => (
                        <ListItem key={`i-${index}`} item={item} ordered={block.ordered} />
                    ))}
                </ul>
            );
        case "code":
            return (
                <pre className="overflow-x-auto rounded-md border border-hairline bg-inset px-2 py-1.5">
                    <code className="font-mono text-xs whitespace-pre text-ink">{block.text}</code>
                </pre>
            );
        case "quote":
            return (
                <blockquote className="border-l-2 border-edge pl-2.5 text-ink-dim italic">
                    <Spans spans={block.spans} />
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
                </p>
            );
    }
}

export interface MarkdownProps {
    text: string;
    trailing?: ReactElement | null;
}

export function Markdown({ text, trailing }: MarkdownProps): ReactElement {
    const blocks = useMemo(() => blocksOf(text), [text]);

    return (
        <div className="flex flex-col gap-2 text-sm leading-relaxed text-ink-soft">
            {blocks.map((block, index) => (
                <Rendered key={`b-${index}`} block={block} />
            ))}
            {trailing ?? null}
        </div>
    );
}
