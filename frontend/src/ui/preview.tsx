import type { ReactElement } from "react";
import { useMemo } from "react";

import { cx } from "./cx.js";

const policy = [
    "default-src 'none'",
    "img-src https: data:",
    "style-src 'unsafe-inline'",
    "font-src data:",
].join("; ");

const frameStyles = [
    "html{color-scheme:light}",
    "body{margin:0;padding:20px;background:#ffffff;color:#16181d;",
    "font:16px/1.6 -apple-system,'Segoe UI',system-ui,sans-serif;overflow-wrap:break-word}",
    "img,video,table{max-width:100%}",
    "table{border-collapse:collapse}",
].join("");

function frameDocument(bodyHtml: string): string {
    return [
        "<!doctype html><html><head><meta charset=\"utf-8\">",
        `<meta http-equiv="Content-Security-Policy" content="${policy}">`,
        `<style>${frameStyles}</style></head><body>`,
        bodyHtml,
        "</body></html>",
    ].join("");
}

export interface HtmlPreviewProps {
    bodyHtml: string;
    title: string;
    className?: string;
}

export function HtmlPreview({ bodyHtml, title, className }: HtmlPreviewProps): ReactElement {
    const srcDoc = useMemo(() => frameDocument(bodyHtml), [bodyHtml]);
    return (
        <iframe
            title={title}
            sandbox=""
            referrerPolicy="no-referrer"
            loading="lazy"
            srcDoc={srcDoc}
            className={cx("h-full w-full border-0 bg-white", className)}
        />
    );
}
