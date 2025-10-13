"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Prompt, createPrompt, updatePrompt } from "@/services/prompt";
import { useErrorHandling } from "@/lib/error-handling";

export interface CreateEditPromptModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  prompt?: Prompt | null;
  onSaved?: () => void | Promise<void>;
}

const extractPlaceholders = (text: string): string[] => {
  if (!text) return [];
  const re = /\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g;
  const set = new Set<string>();
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) !== null) {
    set.add(m[1]);
  }
  return Array.from(set);
};

export function CreateEditPromptModal({ open, onOpenChange, prompt, onSaved }: CreateEditPromptModalProps) {
  const isEdit = !!prompt;
  const { withErrorHandling } = useErrorHandling();

  const [name, setName] = useState("");
  const [systemPrompt, setSystemPrompt] = useState("");
  const [userPrompt, setUserPrompt] = useState("");
  const [placeholders, setPlaceholders] = useState<string[]>([]);
  const [createdAt, setCreatedAt] = useState<string>("");
  const [updatedAt, setUpdatedAt] = useState<string>("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (prompt) {
      setName(prompt.name);
      setSystemPrompt(prompt.systemPrompt || "");
      setUserPrompt(prompt.userPrompt || "");
      setPlaceholders(prompt.placeholders || []);
      setCreatedAt(prompt.createdAt || "");
      setUpdatedAt(prompt.updatedAt || "");
    } else {
      setName("");
      setSystemPrompt("");
      setUserPrompt("");
      setPlaceholders([]);
      setCreatedAt("");
      setUpdatedAt("");
    }
  }, [prompt, open]);

  // Derive placeholders from current textareas
  useEffect(() => {
    const ph = Array.from(
      new Set([...
        extractPlaceholders(systemPrompt),
        ...extractPlaceholders(userPrompt),
      ])
    );
    setPlaceholders(ph);
  }, [systemPrompt, userPrompt]);

  const canSave = name.trim().length > 0 && (systemPrompt.trim().length > 0 || userPrompt.trim().length > 0);

  const handleSave = async () => {
    if (!canSave) return;
    setSaving(true);
    try {
      await withErrorHandling(async () => {
        if (isEdit && prompt) {
          await updatePrompt({
            id: prompt.id,
            name: name.trim(),
            systemPrompt,
            userPrompt,
            placeholders,
            createdAt: prompt.createdAt,
            updatedAt: prompt.updatedAt,
          } as any);
        } else {
          await createPrompt({
            name: name.trim(),
            systemPrompt,
            userPrompt,
            placeholders,
          });
        }
        if (onSaved) await onSaved();
        onOpenChange(false);
      }, { successMessage: isEdit ? "Prompt updated" : "Prompt created", showSuccess: true });
    } finally {
      setSaving(false);
    }
  };

  const formatDate = (d?: string) => {
    if (!d) return "";
    const date = new Date(d);
    return date.toLocaleDateString("en-US", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" });
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !saving && onOpenChange(o)}>
      <DialogContent className="max-w-5xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Prompt" : "Create Prompt"}</DialogTitle>
          <DialogDescription>
            Define system and user prompts. Use placeholders like {"{{title}}"} to mark dynamic parts.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid gap-2">
            <label className="text-sm font-medium">Name</label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Short name" />
          </div>

          <div className="grid gap-2">
            <label className="text-sm font-medium">System Prompt</label>
            <Textarea value={systemPrompt} onChange={(e) => setSystemPrompt(e.target.value)} rows={6} placeholder="You are a helpful assistant..." />
          </div>

          <div className="grid gap-2">
            <label className="text-sm font-medium">User Prompt</label>
            <Textarea value={userPrompt} onChange={(e) => setUserPrompt(e.target.value)} rows={6} placeholder="Write me text on: {{title}} topic" />
          </div>

          <div className="grid gap-1">
            <div className="text-sm font-medium">Detected Placeholders</div>
            {placeholders.length === 0 ? (
              <div className="text-xs text-muted-foreground">Type placeholders as {"{{name}}"} in the prompts to see them here.</div>
            ) : (
              <div className="flex flex-wrap gap-1">
                {placeholders.map((ph) => (
                  <Badge key={ph} className="bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300 border-transparent">{`{{${ph}}}`}</Badge>
                ))}
              </div>
            )}
          </div>

          {isEdit && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs text-muted-foreground">
              <div>Created: <span className="font-medium text-foreground/80">{formatDate(createdAt)}</span></div>
              <div>Updated: <span className="font-medium text-foreground/80">{formatDate(updatedAt)}</span></div>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={saving}>Cancel</Button>
            <Button onClick={handleSave} disabled={!canSave || saving}>{isEdit ? "Save Changes" : "Create"}</Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default CreateEditPromptModal;
