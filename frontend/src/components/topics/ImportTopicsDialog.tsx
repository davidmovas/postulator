"use client";

import React, { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
// Note: Using native select here to avoid hydration issues with different Select implementations
import { useErrorHandling } from "@/lib/error-handling";
import { importAndAssignToSite } from "@/services/topic";
import { syncCategories } from "@/services/site";
import { ALLOWED_IMPORT_EXTENSIONS, DEFAULT_TOPIC_STRATEGY, TOPIC_STRATEGIES } from "@/constants/topics";
import { RefreshCw, FolderOpen } from "lucide-react";
import { CanResolveFilePaths, ResolveFilePaths } from "@/wailsjs/wailsjs/runtime";

export interface ImportTopicsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  siteId: number | null;
  onImported?: () => void | Promise<void>;
}

export function ImportTopicsDialog({ open, onOpenChange, siteId, onImported }: ImportTopicsDialogProps) {
  const { withErrorHandling, showError } = useErrorHandling();

  const [filePath, setFilePath] = useState<string>("");

  // Try to resolve a native system path from a File object.
  // Wails exposes ResolveFilePaths that mutates the File with a .path/nativePath.
  // In some environments this can be async. We poll briefly to detect it.
  const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
  const resolvePathFromFile = async (f: File): Promise<string> => {
    let p = (f as any).path || (f as any).nativePath || "";
    if (p) return p as string;
    try {
      if (CanResolveFilePaths()) {
        ResolveFilePaths([f]);
        // Poll up to ~1s in 50ms steps
        for (let i = 0; i < 20; i++) {
          await sleep(50);
          p = (f as any).path || (f as any).nativePath || "";
          if (p) return p as string;
        }
      }
    } catch {}
    return "";
  };
  // TEMP: category will be 1 until categories are wired; TODO note for future implementation
  const [categoryId, setCategoryId] = useState<string>("1");
  const [strategy, setStrategy] = useState<string>(DEFAULT_TOPIC_STRATEGY);
  const [isImporting, setIsImporting] = useState(false);

  // Selected file name for display
  const [fileName, setFileName] = useState<string>("");

  const reset = () => {
    setFilePath("");
    setFileName("");
    setCategoryId("1");
    setStrategy(DEFAULT_TOPIC_STRATEGY);
  };

  const handleImport = async () => {
    if (!siteId || !filePath.trim()) return;
    setIsImporting(true);
    const ok = await withErrorHandling(async () => {
      await importAndAssignToSite(filePath.trim(), siteId, parseInt(categoryId, 10) || 1, strategy);
    }, { successMessage: "Topics imported and assigned", showSuccess: true });
    setIsImporting(false);
    if (ok !== null) {
      onOpenChange(false);
      reset();
      if (onImported) await onImported();
    }
  };

  const handleSyncCategories = async () => {
    if (!siteId) return;
    await withErrorHandling(async () => {
      // We expose syncCategories via topic service? Currently in site service; fallback to site service if needed.
      // Placeholder call; if not available here, the page should call site.syncCategories.
      await syncCategories(siteId);
    }, { successMessage: "Categories sync requested", showSuccess: true });
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { onOpenChange(false); reset(); } else { onOpenChange(true); } }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Import topics</DialogTitle>
          <DialogDescription>
            Choose a file to import topics and assign them to this site. Supported: {ALLOWED_IMPORT_EXTENSIONS.join(", ")}. 
            NOTE: Category selection is temporary and defaults to 1 until categories are implemented.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-2">
          {/* File picker with drag & drop */}
          <div className="space-y-2">
            <Label>File</Label>
            <div
              onDragOver={(e) => { e.preventDefault(); e.stopPropagation(); }}
              onDrop={async (e) => {
                e.preventDefault();
                const f = e.dataTransfer.files?.[0];
                if (f) {
                  const p = await resolvePathFromFile(f);
                  setFileName(f.name);
                  if (p) {
                    setFilePath(p);
                  } else {
                    setFilePath("");
                    showError("Не удалось получить путь к файлу. Перетащите файл из проводника в окно приложения или попробуйте ещё раз.");
                  }
                }
              }}
              className="border border-dashed rounded-md p-4 text-sm text-muted-foreground hover:bg-accent/40 cursor-pointer"
              onClick={() => {
                const input = document.getElementById("import-file-input") as HTMLInputElement | null;
                input?.click();
              }}
            >
              {filePath ? (
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-foreground font-medium">{fileName || filePath}</div>
                    {filePath && <div className="text-xs text-muted-foreground">{filePath}</div>}
                  </div>
                  <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); setFilePath(""); setFileName(""); }}>Clear</Button>
                </div>
              ) : (
                <div className="flex items-center justify-between">
                  <div>
                    <div>Drag & drop the file here</div>
                    <div className="text-xs">Supported: {ALLOWED_IMPORT_EXTENSIONS.join(", ")}</div>
                  </div>
                  <Button size="sm" variant="outline" onClick={(e) => { e.stopPropagation(); const input = document.getElementById("import-file-input") as HTMLInputElement | null; input?.click(); }}>
                    <FolderOpen className="h-4 w-4 mr-2" /> Browse...
                  </Button>
                </div>
              )}
            </div>
            <input
              id="import-file-input"
              type="file"
              className="hidden"
              accept={ALLOWED_IMPORT_EXTENSIONS.map((ext) => `.${ext}`).join(",")}
              onChange={async (e) => {
                const f = e.target.files?.[0];
                if (f) {
                  const p = await resolvePathFromFile(f);
                  setFileName(f.name);
                  if (p) {
                    setFilePath(p);
                  } else {
                    setFilePath("");
                    showError("Не удалось получить путь к файлу. Перетащите файл из проводника в окно приложения или попробуйте ещё раз.");
                  }
                }
              }}
            />
          </div>


          {/* Category + Sync in one row */}
          <div className="flex items-end gap-3">
            <div className="flex-1 space-y-2">
              <Label htmlFor="category">Category (temporary)</Label>
              <Input id="category" type="number" value={categoryId} onChange={(e) => setCategoryId(e.target.value)} />
            </div>
            <div className="pb-0">
              <Button variant="secondary" onClick={handleSyncCategories} disabled={!siteId || isImporting}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Sync Categories
              </Button>
            </div>
          </div>

          {/* Strategy */}
          <div className="space-y-2">
            <Label htmlFor="strategy">Strategy</Label>
            <select
              id="strategy"
              value={strategy}
              onChange={(e) => setStrategy(e.target.value)}
              className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="" disabled hidden>Select strategy</option>
              {TOPIC_STRATEGIES.map((s) => (
                <option key={s} value={s} className="capitalize">{s}</option>
              ))}
            </select>
          </div>
        </div>

        <DialogFooter className="pt-4">
          <Button variant="ghost" onClick={() => { onOpenChange(false); reset(); }} disabled={isImporting}>Cancel</Button>
          <Button onClick={handleImport} disabled={!filePath.trim() || !siteId || isImporting}>Import</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default ImportTopicsDialog;
