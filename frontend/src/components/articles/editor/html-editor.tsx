"use client";

import { useCallback, useRef, useState } from "react";
import Editor, { OnMount, OnChange } from "@monaco-editor/react";
import { useTheme } from "next-themes";
import { Button } from "@/components/ui/button";
import {
    Bold,
    Italic,
    Underline,
    Heading1,
    Heading2,
    Heading3,
    List,
    ListOrdered,
    Quote,
    Link as LinkIcon,
    Image as ImageIcon,
    Code,
    Table,
    Minus,
} from "lucide-react";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { InsertMediaModal } from "./insert-media-modal";
import { InsertLinkModal } from "./insert-link-modal";
import type { editor } from "monaco-editor";

interface HtmlEditorProps {
    content: string;
    onChange: (html: string) => void;
    onFormat?: (html: string) => void;
    disabled?: boolean;
}

interface ToolbarButtonProps {
    icon: React.ReactNode;
    label: string;
    onClick: () => void;
    disabled?: boolean;
}

function ToolbarButton({ icon, label, onClick, disabled }: ToolbarButtonProps) {
    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={onClick}
                    disabled={disabled}
                    className="h-8 w-8 p-0"
                >
                    {icon}
                </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom" className="text-xs">
                {label}
            </TooltipContent>
        </Tooltip>
    );
}

function ToolbarSeparator() {
    return <div className="w-px h-6 bg-border mx-1" />;
}

// Format HTML with proper indentation
function formatHtml(html: string): string {
    const tab = "  "; // 2 spaces
    let result = "";
    let indent = 0;

    // Remove existing whitespace between tags
    const cleanHtml = html.replace(/>\s+</g, "><").trim();

    // Split by tags
    const tokens = cleanHtml.split(/(<[^>]+>)/g).filter(Boolean);

    const selfClosingTags = new Set([
        "img", "br", "hr", "input", "meta", "link", "area", "base", "col", "embed", "source", "track", "wbr"
    ]);

    for (const token of tokens) {
        if (token.startsWith("</")) {
            // Closing tag
            indent = Math.max(0, indent - 1);
            result += tab.repeat(indent) + token + "\n";
        } else if (token.startsWith("<")) {
            // Opening tag or self-closing
            const tagMatch = token.match(/^<(\w+)/);
            const tagName = tagMatch ? tagMatch[1].toLowerCase() : "";
            const isSelfClosing = selfClosingTags.has(tagName) || token.endsWith("/>");

            result += tab.repeat(indent) + token + "\n";

            if (!isSelfClosing) {
                indent++;
            }
        } else {
            // Text content
            const trimmed = token.trim();
            if (trimmed) {
                result += tab.repeat(indent) + trimmed + "\n";
            }
        }
    }

    return result.trim();
}

export function HtmlEditor({
    content,
    onChange,
    onFormat,
    disabled = false,
}: HtmlEditorProps) {
    const { resolvedTheme } = useTheme();
    const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
    const [showImageModal, setShowImageModal] = useState(false);
    const [showLinkModal, setShowLinkModal] = useState(false);

    const handleEditorMount: OnMount = (editor) => {
        editorRef.current = editor;
    };

    const handleEditorChange: OnChange = (value) => {
        onChange(value || "");
    };

    const insertAtCursor = useCallback((before: string, after: string = "") => {
        const editor = editorRef.current;
        if (!editor) return;

        const selection = editor.getSelection();
        if (!selection) return;

        const model = editor.getModel();
        if (!model) return;

        const selectedText = model.getValueInRange(selection);
        const newText = before + selectedText + after;

        editor.executeEdits("insert", [{
            range: selection,
            text: newText,
            forceMoveMarkers: true,
        }]);

        // Move cursor to after the inserted text
        const newPosition = {
            lineNumber: selection.startLineNumber,
            column: selection.startColumn + before.length + selectedText.length,
        };
        editor.setPosition(newPosition);
        editor.focus();
    }, []);

    const handleInsertImage = useCallback((url: string, alt?: string) => {
        insertAtCursor(`<img src="${url}" alt="${alt || ""}" />`);
        setShowImageModal(false);
    }, [insertAtCursor]);

    const handleInsertLink = useCallback((url: string, text?: string) => {
        if (text) {
            insertAtCursor(`<a href="${url}">${text}</a>`);
        } else {
            insertAtCursor(`<a href="${url}">`, "</a>");
        }
        setShowLinkModal(false);
    }, [insertAtCursor]);

    const handleFormat = useCallback(() => {
        const formatted = formatHtml(content);
        // Use onFormat if provided (doesn't mark as dirty), otherwise fall back to onChange
        if (onFormat) {
            onFormat(formatted);
        } else {
            onChange(formatted);
        }
    }, [content, onChange, onFormat]);

    const htmlSnippets = {
        bold: () => insertAtCursor("<strong>", "</strong>"),
        italic: () => insertAtCursor("<em>", "</em>"),
        underline: () => insertAtCursor("<u>", "</u>"),
        h1: () => insertAtCursor("<h1>", "</h1>"),
        h2: () => insertAtCursor("<h2>", "</h2>"),
        h3: () => insertAtCursor("<h3>", "</h3>"),
        ul: () => insertAtCursor("<ul>\n  <li>", "</li>\n</ul>"),
        ol: () => insertAtCursor("<ol>\n  <li>", "</li>\n</ol>"),
        quote: () => insertAtCursor("<blockquote>", "</blockquote>"),
        code: () => insertAtCursor("<code>", "</code>"),
        hr: () => insertAtCursor("<hr />\n"),
        table: () => insertAtCursor(
            "<table>\n  <thead>\n    <tr>\n      <th>Header</th>\n      <th>Header</th>\n    </tr>\n  </thead>\n  <tbody>\n    <tr>\n      <td>Cell</td>\n      <td>Cell</td>\n    </tr>\n  </tbody>\n</table>\n"
        ),
    };

    return (
        <TooltipProvider delayDuration={300}>
            <div className="border rounded-lg overflow-hidden bg-background">
                {/* Toolbar */}
                <div className="flex items-center gap-1 p-2 border-b bg-muted/30 flex-wrap">
                    <ToolbarButton
                        icon={<Bold className="h-4 w-4" />}
                        label="Bold"
                        onClick={htmlSnippets.bold}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<Italic className="h-4 w-4" />}
                        label="Italic"
                        onClick={htmlSnippets.italic}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<Underline className="h-4 w-4" />}
                        label="Underline"
                        onClick={htmlSnippets.underline}
                        disabled={disabled}
                    />

                    <ToolbarSeparator />

                    <ToolbarButton
                        icon={<Heading1 className="h-4 w-4" />}
                        label="Heading 1"
                        onClick={htmlSnippets.h1}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<Heading2 className="h-4 w-4" />}
                        label="Heading 2"
                        onClick={htmlSnippets.h2}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<Heading3 className="h-4 w-4" />}
                        label="Heading 3"
                        onClick={htmlSnippets.h3}
                        disabled={disabled}
                    />

                    <ToolbarSeparator />

                    <ToolbarButton
                        icon={<List className="h-4 w-4" />}
                        label="Bullet List"
                        onClick={htmlSnippets.ul}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<ListOrdered className="h-4 w-4" />}
                        label="Numbered List"
                        onClick={htmlSnippets.ol}
                        disabled={disabled}
                    />

                    <ToolbarSeparator />

                    <ToolbarButton
                        icon={<Quote className="h-4 w-4" />}
                        label="Blockquote"
                        onClick={htmlSnippets.quote}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<Code className="h-4 w-4" />}
                        label="Code"
                        onClick={htmlSnippets.code}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<Minus className="h-4 w-4" />}
                        label="Horizontal Rule"
                        onClick={htmlSnippets.hr}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<Table className="h-4 w-4" />}
                        label="Table"
                        onClick={htmlSnippets.table}
                        disabled={disabled}
                    />

                    <ToolbarSeparator />

                    <ToolbarButton
                        icon={<LinkIcon className="h-4 w-4" />}
                        label="Insert Link"
                        onClick={() => setShowLinkModal(true)}
                        disabled={disabled}
                    />
                    <ToolbarButton
                        icon={<ImageIcon className="h-4 w-4" />}
                        label="Insert Image"
                        onClick={() => setShowImageModal(true)}
                        disabled={disabled}
                    />

                    <div className="ml-auto">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleFormat}
                            disabled={disabled}
                            className="h-8 text-xs"
                        >
                            Format
                        </Button>
                    </div>
                </div>

                {/* Monaco Editor */}
                <Editor
                    height="450px"
                    language="html"
                    theme={resolvedTheme === "dark" ? "vs-dark" : "light"}
                    value={content}
                    onChange={handleEditorChange}
                    onMount={handleEditorMount}
                    options={{
                        minimap: { enabled: false },
                        fontSize: 14,
                        lineNumbers: "on",
                        wordWrap: "on",
                        tabSize: 2,
                        insertSpaces: true,
                        automaticLayout: true,
                        scrollBeyondLastLine: false,
                        readOnly: disabled,
                        formatOnPaste: true,
                        formatOnType: true,
                        autoIndent: "full",
                        folding: true,
                        foldingStrategy: "indentation",
                        renderWhitespace: "selection",
                        bracketPairColorization: {
                            enabled: true,
                        },
                    }}
                />
            </div>

            {/* Modals */}
            <InsertMediaModal
                open={showImageModal}
                onOpenChange={setShowImageModal}
                onInsert={handleInsertImage}
            />

            <InsertLinkModal
                open={showLinkModal}
                onOpenChange={setShowLinkModal}
                onInsert={handleInsertLink}
            />
        </TooltipProvider>
    );
}
