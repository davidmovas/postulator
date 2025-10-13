"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useErrorHandling } from "@/lib/error-handling";
import { AIProvider, createAIProvider, getAIModels, getAvailableModels, updateAIProvider, validateModel } from "@/services/aiProvider";
import { Select } from "@/components/ui/select";

export interface CreateEditAIProviderModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  provider?: AIProvider | null;
  onSaved?: () => void | Promise<void>;
}

export function CreateEditAIProviderModal({ open, onOpenChange, provider, onSaved }: CreateEditAIProviderModalProps) {
  const isEdit = !!provider;
  const { withErrorHandling, showSuccess } = useErrorHandling();

  const [name, setName] = useState("");
  const [providerName, setProviderName] = useState<string>("");
  const [model, setModel] = useState<string>("");
  const [customModel, setCustomModel] = useState<string>("");
  const [apiKey, setApiKey] = useState("");
  const [isActive, setIsActive] = useState<boolean>(true);
  const [createdAt, setCreatedAt] = useState<string>("");
  const [updatedAt, setUpdatedAt] = useState<string>("");

  const [modelsByProvider, setModelsByProvider] = useState<Record<string, string[]>>({});
  const [modelOptions, setModelOptions] = useState<string[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [saving, setSaving] = useState(false);

  // Load provider registry
  useEffect(() => {
    if (!open) return;
    (async () => {
      try {
        const m = await getAIModels();
        setModelsByProvider({ openai: m.openai, anthropic: m.anthropic, google: m.google });
      } catch {}
    })();
  }, [open]);

  // Initialize from edit or reset
  useEffect(() => {
    if (provider && open) {
      setName(provider.name);
      // Try to infer provider name by matching model presence
      setProviderName(prev => prev); // keep until models are loaded; adjust below
      setModel(provider.model);
      setCustomModel("");
      setApiKey(""); // optional on edit; empty means unchanged
      setIsActive(provider.isActive);
      setCreatedAt(provider.createdAt);
      setUpdatedAt(provider.updatedAt);
    } else if (open) {
      setName("");
      setProviderName("");
      setModel("");
      setCustomModel("");
      setApiKey("");
      setIsActive(true);
      setCreatedAt("");
      setUpdatedAt("");
    }
  }, [provider, open]);

  // When providerName changes, load available models
  useEffect(() => {
    if (!providerName) { setModelOptions([]); return; }
    setLoadingModels(true);
    (async () => {
      try {
        const opts = await getAvailableModels(providerName);
        setModelOptions(opts);
        // If editing and model belongs to this provider, keep it selected
        if (provider && opts.includes(provider.model)) {
          setModel(provider.model);
        } else if (!provider) {
          setModel(opts[0] || "");
        }
      } catch {
        setModelOptions([]);
      } finally {
        setLoadingModels(false);
      }
    })();
  }, [providerName]);

  // Infer provider name once models are known in edit mode
  useEffect(() => {
    if (isEdit && provider && provider.model && Object.keys(modelsByProvider).length > 0 && !providerName) {
      const entry = Object.entries(modelsByProvider).find(([, arr]) => arr?.includes(provider.model));
      if (entry) setProviderName(entry[0]);
    }
  }, [isEdit, provider, modelsByProvider, providerName]);

  const providersList = useMemo(() => Object.keys(modelsByProvider), [modelsByProvider]);

  const selectedModel = (customModel || model).trim();
  const canSave = name.trim().length > 0 && providerName && selectedModel.length > 0 && (!isEdit ? apiKey.trim().length > 0 : true);

  const handleValidate = async () => {
    if (!providerName || !selectedModel) return;
    await withErrorHandling(async () => {
      await validateModel(providerName, selectedModel);
      showSuccess("Model is valid", "Validation");
    }, { showSuccess: false });
  };

  const handleSave = async () => {
    if (!canSave) return;
    setSaving(true);
    try {
      // Validate model first; throw will be caught by withErrorHandling
      const ok = await withErrorHandling(async () => {
        await validateModel(providerName, selectedModel);
      }, { showSuccess: false });
      if (ok === null) { return; }

      await withErrorHandling(async () => {
        if (isEdit && provider) {
          await updateAIProvider({
            id: provider.id,
            name: name.trim(),
            model: selectedModel,
            isActive,
            apiKey: apiKey.trim() ? apiKey.trim() : undefined,
          });
        } else {
          await createAIProvider({
            name: name.trim(),
            apiKey: apiKey.trim(),
            model: selectedModel,
            isActive,
          });
        }
        if (onSaved) await onSaved();
        onOpenChange(false);
      }, { successMessage: isEdit ? "AI Provider updated" : "AI Provider created", showSuccess: true });
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
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit AI Provider" : "Create AI Provider"}</DialogTitle>
          <DialogDescription>
            Configure AI provider connection in a simple flow. Pick a provider, choose or type a model, validate, then add your API key.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Name */}
          <div className="grid gap-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Short display name" />
          </div>

          {/* Provider + Model (single column) */}
          <div className="grid gap-3">
            <div className="grid gap-2">
              <Label>Provider</Label>
              <Select value={providerName} onChange={(e) => setProviderName(e.target.value)} disabled={providersList.length === 0}>
                <option value="" disabled>Select provider</option>
                {providersList.map((p) => (
                  <option key={p} value={p}>{p}</option>
                ))}
              </Select>
            </div>

            <div className="grid gap-2">
              <Label>Model</Label>
              <Select value={model} onChange={(e) => setModel(e.target.value)} disabled={!providerName || loadingModels || modelOptions.length === 0}>
                {(!providerName || modelOptions.length === 0) && <option value="">{loadingModels ? "Loading..." : "Select provider first"}</option>}
                {modelOptions.map((m) => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </Select>
              <div className="text-[11px] text-muted-foreground">Or type a custom model and validate it.</div>
              <div className="flex items-center gap-2">
                <Input value={customModel} onChange={(e) => setCustomModel(e.target.value)} placeholder="Custom model (e.g. gpt-4o)" />
                <Button variant="outline" size="sm" onClick={handleValidate} disabled={!providerName || !selectedModel}>Validate</Button>
              </div>
            </div>
          </div>

          {/* API Key */}
          <div className="grid gap-2">
            <Label>API Key {isEdit && <span className="text-xs text-muted-foreground">(leave blank to keep unchanged)</span>}</Label>
            <Input value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="sk-..." type="password" />
          </div>

          {/* Edit meta */}
          {isEdit && (
            <div className="grid gap-1 text-xs text-muted-foreground">
              <div>Created: <span className="font-medium text-foreground/80">{formatDate(createdAt)}</span></div>
              <div>Updated: <span className="font-medium text-foreground/80">{formatDate(updatedAt)}</span></div>
            </div>
          )}

          {/* Footer row: left checkbox, right actions */}
          <div className="flex items-center justify-between gap-3 pt-2">
            <div className="flex items-center gap-2">
              <Checkbox id="active" checked={isActive} onCheckedChange={(v) => setIsActive(!!v)} />
              <label htmlFor="active" className="text-sm">Active</label>
            </div>
            <div className="flex gap-2">
              <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={saving}>Cancel</Button>
              <Button onClick={handleSave} disabled={!canSave || saving}>{isEdit ? "Save Changes" : "Create"}</Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default CreateEditAIProviderModal;
