"use client";

import { useCallback, useState, useEffect } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import Link from "@tiptap/extension-link";
import Underline from "@tiptap/extension-underline";
import TextAlign from "@tiptap/extension-text-align";
import Placeholder from "@tiptap/extension-placeholder";
import { Button } from "@/components/ui/button";
import {
    Bold,
    Italic,
    Underline as UnderlineIcon,
    Strikethrough,
    Code,
    Heading1,
    Heading2,
    Heading3,
    List,
    ListOrdered,
    Quote,
    Redo,
    Undo,
    Link as LinkIcon,
    Image as ImageIcon,
    AlignLeft,
    AlignCenter,
    AlignRight,
    AlignJustify,
    Minus,
    Pilcrow,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { InsertMediaModal } from "./insert-media-modal";
import { InsertLinkModal } from "./insert-link-modal";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";

interface BlockEditorProps {
    content: string;
    onChange: (html: string) => void;
    placeholder?: string;
    disabled?: boolean;
}

interface ToolbarButtonProps {
    icon: React.ReactNode;
    label: string;
    onClick: () => void;
    disabled?: boolean;
    active?: boolean;
}

function ToolbarButton({ icon, label, onClick, disabled, active }: ToolbarButtonProps) {
    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={onClick}
                    disabled={disabled}
                    className={cn(
                        "h-8 w-8 p-0",
                        active && "bg-accent text-accent-foreground"
                    )}
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

export function BlockEditor({
    content,
    onChange,
    placeholder = "Start writing your article...",
    disabled = false,
}: BlockEditorProps) {
    const [showImageModal, setShowImageModal] = useState(false);
    const [showLinkModal, setShowLinkModal] = useState(false);

    const editor = useEditor({
        extensions: [
            StarterKit.configure({
                heading: {
                    levels: [1, 2, 3],
                },
            }),
            Underline,
            Image.configure({
                HTMLAttributes: {
                    class: "rounded-lg max-w-full h-auto",
                },
            }),
            Link.configure({
                openOnClick: false,
                HTMLAttributes: {
                    class: "text-primary underline",
                },
            }),
            TextAlign.configure({
                types: ["heading", "paragraph"],
            }),
            Placeholder.configure({
                placeholder,
            }),
        ],
        content,
        editable: !disabled,
        immediatelyRender: false,
        onUpdate: ({ editor }) => {
            onChange(editor.getHTML());
        },
        editorProps: {
            attributes: {
                class: "prose prose-sm dark:prose-invert max-w-none min-h-[400px] p-4 focus:outline-none",
            },
        },
    });

    // Sync content from props when it changes externally (e.g., AI generation)
    useEffect(() => {
        if (editor && content !== editor.getHTML()) {
            editor.commands.setContent(content);
        }
    }, [content, editor]);

    const handleInsertImage = useCallback((url: string, alt?: string) => {
        if (editor) {
            editor.chain().focus().setImage({ src: url, alt: alt || "" }).run();
        }
        setShowImageModal(false);
    }, [editor]);

    const handleInsertLink = useCallback((url: string, text?: string) => {
        if (editor) {
            if (text && editor.state.selection.empty) {
                editor.chain().focus().insertContent(`<a href="${url}">${text}</a>`).run();
            } else {
                editor.chain().focus().setLink({ href: url }).run();
            }
        }
        setShowLinkModal(false);
    }, [editor]);

    if (!editor) {
        return null;
    }

    return (
        <TooltipProvider delayDuration={300}>
            <div className="border rounded-lg overflow-hidden bg-background">
                {/* Toolbar */}
                <div className="flex items-center gap-1 p-2 border-b bg-muted/30 flex-wrap">
                    {/* Undo/Redo */}
                    <ToolbarButton
                        icon={<Undo className="h-4 w-4" />}
                        label="Undo"
                        onClick={() => editor.chain().focus().undo().run()}
                        disabled={disabled || !editor.can().undo()}
                    />
                    <ToolbarButton
                        icon={<Redo className="h-4 w-4" />}
                        label="Redo"
                        onClick={() => editor.chain().focus().redo().run()}
                        disabled={disabled || !editor.can().redo()}
                    />

                    <ToolbarSeparator />

                    {/* Text formatting */}
                    <ToolbarButton
                        icon={<Bold className="h-4 w-4" />}
                        label="Bold"
                        onClick={() => editor.chain().focus().toggleBold().run()}
                        disabled={disabled}
                        active={editor.isActive("bold")}
                    />
                    <ToolbarButton
                        icon={<Italic className="h-4 w-4" />}
                        label="Italic"
                        onClick={() => editor.chain().focus().toggleItalic().run()}
                        disabled={disabled}
                        active={editor.isActive("italic")}
                    />
                    <ToolbarButton
                        icon={<UnderlineIcon className="h-4 w-4" />}
                        label="Underline"
                        onClick={() => editor.chain().focus().toggleUnderline().run()}
                        disabled={disabled}
                        active={editor.isActive("underline")}
                    />
                    <ToolbarButton
                        icon={<Strikethrough className="h-4 w-4" />}
                        label="Strikethrough"
                        onClick={() => editor.chain().focus().toggleStrike().run()}
                        disabled={disabled}
                        active={editor.isActive("strike")}
                    />
                    <ToolbarButton
                        icon={<Code className="h-4 w-4" />}
                        label="Code"
                        onClick={() => editor.chain().focus().toggleCode().run()}
                        disabled={disabled}
                        active={editor.isActive("code")}
                    />

                    <ToolbarSeparator />

                    {/* Headings */}
                    <ToolbarButton
                        icon={<Pilcrow className="h-4 w-4" />}
                        label="Paragraph"
                        onClick={() => editor.chain().focus().setParagraph().run()}
                        disabled={disabled}
                        active={editor.isActive("paragraph")}
                    />
                    <ToolbarButton
                        icon={<Heading1 className="h-4 w-4" />}
                        label="Heading 1"
                        onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
                        disabled={disabled}
                        active={editor.isActive("heading", { level: 1 })}
                    />
                    <ToolbarButton
                        icon={<Heading2 className="h-4 w-4" />}
                        label="Heading 2"
                        onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
                        disabled={disabled}
                        active={editor.isActive("heading", { level: 2 })}
                    />
                    <ToolbarButton
                        icon={<Heading3 className="h-4 w-4" />}
                        label="Heading 3"
                        onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
                        disabled={disabled}
                        active={editor.isActive("heading", { level: 3 })}
                    />

                    <ToolbarSeparator />

                    {/* Lists */}
                    <ToolbarButton
                        icon={<List className="h-4 w-4" />}
                        label="Bullet List"
                        onClick={() => editor.chain().focus().toggleBulletList().run()}
                        disabled={disabled}
                        active={editor.isActive("bulletList")}
                    />
                    <ToolbarButton
                        icon={<ListOrdered className="h-4 w-4" />}
                        label="Numbered List"
                        onClick={() => editor.chain().focus().toggleOrderedList().run()}
                        disabled={disabled}
                        active={editor.isActive("orderedList")}
                    />

                    <ToolbarSeparator />

                    {/* Alignment */}
                    <ToolbarButton
                        icon={<AlignLeft className="h-4 w-4" />}
                        label="Align Left"
                        onClick={() => editor.chain().focus().setTextAlign("left").run()}
                        disabled={disabled}
                        active={editor.isActive({ textAlign: "left" })}
                    />
                    <ToolbarButton
                        icon={<AlignCenter className="h-4 w-4" />}
                        label="Align Center"
                        onClick={() => editor.chain().focus().setTextAlign("center").run()}
                        disabled={disabled}
                        active={editor.isActive({ textAlign: "center" })}
                    />
                    <ToolbarButton
                        icon={<AlignRight className="h-4 w-4" />}
                        label="Align Right"
                        onClick={() => editor.chain().focus().setTextAlign("right").run()}
                        disabled={disabled}
                        active={editor.isActive({ textAlign: "right" })}
                    />
                    <ToolbarButton
                        icon={<AlignJustify className="h-4 w-4" />}
                        label="Justify"
                        onClick={() => editor.chain().focus().setTextAlign("justify").run()}
                        disabled={disabled}
                        active={editor.isActive({ textAlign: "justify" })}
                    />

                    <ToolbarSeparator />

                    {/* Block elements */}
                    <ToolbarButton
                        icon={<Quote className="h-4 w-4" />}
                        label="Blockquote"
                        onClick={() => editor.chain().focus().toggleBlockquote().run()}
                        disabled={disabled}
                        active={editor.isActive("blockquote")}
                    />
                    <ToolbarButton
                        icon={<Minus className="h-4 w-4" />}
                        label="Horizontal Rule"
                        onClick={() => editor.chain().focus().setHorizontalRule().run()}
                        disabled={disabled}
                    />

                    <ToolbarSeparator />

                    {/* Insert elements */}
                    <ToolbarButton
                        icon={<LinkIcon className="h-4 w-4" />}
                        label="Insert Link"
                        onClick={() => setShowLinkModal(true)}
                        disabled={disabled}
                        active={editor.isActive("link")}
                    />
                    <ToolbarButton
                        icon={<ImageIcon className="h-4 w-4" />}
                        label="Insert Image"
                        onClick={() => setShowImageModal(true)}
                        disabled={disabled}
                    />
                </div>

                {/* Editor Content */}
                <EditorContent editor={editor} />
            </div>

            {/* Modals */}
            <InsertMediaModal
                open={showImageModal}
                onOpenChange={setShowImageModal}
                onInsert={handleInsertImage}
                title="Insert Image"
                placeholder="https://example.com/image.jpg"
            />

            <InsertLinkModal
                open={showLinkModal}
                onOpenChange={setShowLinkModal}
                onInsert={handleInsertLink}
            />
        </TooltipProvider>
    );
}
