"use client";

import React, { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Site } from "@/services/site";

export interface SetSitePasswordDialogProps {
  open: boolean;
  site: Site | null;
  onOpenChange: (open: boolean) => void;
  onSubmit: (password: string) => void | Promise<void>;
  loading?: boolean;
}

export function SetSitePasswordDialog({ open, site, onOpenChange, onSubmit, loading = false }: SetSitePasswordDialogProps) {
  const [value, setValue] = useState("");

  useEffect(() => {
    if (!open) setValue("");
  }, [open, site]);

  const handleConfirm = async () => {
    const pw = value.trim();
    if (!pw) return;
    await onSubmit(pw);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Set password {site ? `for ${site.name}` : ""}</DialogTitle>
          <DialogDescription>
            For security, password is managed separately. Enter a new password for this site.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-3 pt-1">
          <div className="space-y-2">
            <label htmlFor="pw-input" className="text-sm font-medium">Password</label>
            <Input
              id="pw-input"
              type="password"
              placeholder="Enter password"
              value={value}
              onChange={(e) => setValue(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={loading}>Cancel</Button>
          <Button onClick={handleConfirm} disabled={!value.trim() || !site || loading}>Set Password</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default SetSitePasswordDialog;
