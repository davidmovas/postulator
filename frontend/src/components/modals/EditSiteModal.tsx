"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Site, updateSite, setSitePassword } from "@/services/site";
import { useErrorHandling } from "@/lib/error-handling";

export interface EditSiteModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  site: Site | null;
  onSaved?: () => void | Promise<void>;
}

export function EditSiteModal({ open, onOpenChange, site, onSaved }: EditSiteModalProps) {
  const { withErrorHandling } = useErrorHandling();

  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [wpUsername, setWpUsername] = useState("");
  const [password, setPassword] = useState("");
  const [nameTouched, setNameTouched] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (site) {
      setName(site.name || "");
      setUrl(site.url || "");
      setWpUsername(site.wpUsername || "");
      setPassword("");
      setNameTouched(false);
    }
  }, [site, open]);

  const normalizeUrl = (value: string): string => {
    if (!value) return value;
    try {
      const hasProtocol = /^https?:\/\//i.test(value);
      const u = new URL(hasProtocol ? value : `http://${value}`);
      return u.toString().replace(/\/$/, "");
    } catch (e) {
      return value;
    }
  };

  const extractDomain = (value: string): string => {
    try {
      const hasProtocol = /^https?:\/\//i.test(value);
      const u = new URL(hasProtocol ? value : `http://${value}`);
      let host = u.hostname.toLowerCase();
      if (host.startsWith("www.")) host = host.slice(4);
      return host;
    } catch (e) {
      return "";
    }
  };

  const handleUrlChange = (value: string) => {
    setUrl(value);
    if (!nameTouched && !name.trim()) {
      const domain = extractDomain(value);
      if (domain) setName(domain);
    }
  };

  const isSaveDisabled = useMemo(() => {
    return !site || !url.trim() || !wpUsername.trim();
  }, [site, url, wpUsername]);

  const onSave = async () => {
    if (!site) return;
    setIsSaving(true);
    const payload = {
      id: site.id,
      name: name.trim() || extractDomain(url.trim()) || site.name,
      url: normalizeUrl(url.trim()),
      wpUsername: wpUsername.trim(),
    };

    const ok = await withErrorHandling(async () => {
      await updateSite(payload);
      if (password.trim()) {
        await setSitePassword(site.id, password.trim());
      }
    }, { successMessage: 'Site updated', showSuccess: true });

    setIsSaving(false);
    if (ok !== null) {
      onOpenChange(false);
      if (onSaved) await onSaved();
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { onOpenChange(false); } }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit site</DialogTitle>
          <DialogDescription>
            Update site details. To change password, enter a new one below (optional).
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 pt-2">
          <div className="space-y-2">
            <Label htmlFor="edit-url">URL</Label>
            <Input
              id="edit-url"
              placeholder="https://example.com"
              value={url}
              onChange={(e) => handleUrlChange(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-name">Name</Label>
            <Input
              id="edit-name"
              placeholder="example.com"
              value={name}
              onChange={(e) => { setName(e.target.value); setNameTouched(true); }}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-wpuser">User</Label>
            <Input
              id="edit-wpuser"
              placeholder="wordpress username"
              value={wpUsername}
              onChange={(e) => setWpUsername(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-password">Password (optional)</Label>
            <Input
              id="edit-password"
              type="password"
              placeholder="Enter new password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter className="pt-4">
          <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={isSaving}>Cancel</Button>
          <Button onClick={onSave} disabled={isSaveDisabled || isSaving}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
