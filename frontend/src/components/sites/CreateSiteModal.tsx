"use client";

import React, { useMemo, useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { createSite, listSites, setSitePassword } from "@/services/site";
import { useErrorHandling } from "@/lib/error-handling";

export interface CreateSiteModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: () => void | Promise<void>; // optional callback to refresh list
}

export function CreateSiteModal({ open, onOpenChange, onCreated }: CreateSiteModalProps) {
  const { withErrorHandling } = useErrorHandling();

  const [step, setStep] = useState<"create" | "password">("create");

  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [wpUsername, setWpUsername] = useState("");
  const [password, setPassword] = useState("");

  const [nameTouched, setNameTouched] = useState(false);
  const [createdSiteId, setCreatedSiteId] = useState<number | null>(null);

  const resetState = () => {
    setStep("create");
    setName("");
    setUrl("");
    setWpUsername("");
    setPassword("");
    setNameTouched(false);
    setCreatedSiteId(null);
  };

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

  const isCreateDisabled = useMemo(() => {
    return !url.trim() || !wpUsername.trim();
  }, [url, wpUsername]);

  const onSubmitCreate = async () => {
    const payload = {
      name: name.trim() || extractDomain(url.trim()) || "New Site",
      url: normalizeUrl(url.trim()),
      wpUsername: wpUsername.trim(),
    };

    const siteId = await withErrorHandling(async () => {
      await createSite(payload);
      const all = await listSites();
      // Find by exact URL match; if multiple, take the one with max id
      const byUrl = (all || []).filter((s) => s.url === payload.url);
      let foundId: number | null = null;
      if (byUrl.length > 0) {
        foundId = byUrl.reduce((max, s) => (s.id > max ? s.id : max), byUrl[0].id);
      }
      setCreatedSiteId(foundId);
      if (onCreated) await onCreated();
      return foundId;
    }, { successMessage: "Site created", showSuccess: true });

    if (siteId !== null) {
      setStep("password");
    }
  };

  const onSubmitPassword = async () => {
    if (!createdSiteId) {
      onOpenChange(false);
      resetState();
      return;
    }
    const pw = password.trim();
    if (!pw) {
      onOpenChange(false);
      resetState();
      return;
    }
    await withErrorHandling(
      async () => {
        await setSitePassword(createdSiteId, pw);
      },
      { successMessage: "Password set for site", showSuccess: true }
    );
    onOpenChange(false);
    resetState();
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) {
      // closing
      onOpenChange(false);
      // give a small reset to not flash content if immediately reopened
      resetState();
    } else {
      onOpenChange(true);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        {step === "create" ? (
          <>
            <DialogHeader>
              <DialogTitle>Add new site</DialogTitle>
              <DialogDescription>
                Enter site details. We’ll auto-fill the name from the URL if left blank. You can edit it anytime.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 pt-2">
              <div className="space-y-2">
                <Label htmlFor="url">URL</Label>
                <Input
                  id="url"
                  placeholder="https://example.com"
                  value={url}
                  onChange={(e) => handleUrlChange(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  placeholder="example.com"
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value);
                    setNameTouched(true);
                  }}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="wpuser">User</Label>
                <Input
                  id="wpuser"
                  placeholder="wordpress username"
                  value={wpUsername}
                  onChange={(e) => setWpUsername(e.target.value)}
                />
              </div>
            </div>
            <DialogFooter className="pt-4">
              <Button variant="ghost" onClick={() => handleOpenChange(false)}>Cancel</Button>
              <Button onClick={onSubmitCreate} disabled={isCreateDisabled}>Create</Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>Set site password</DialogTitle>
              <DialogDescription>
                Site was created successfully. For security, password is set separately. You can do it now or later.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 pt-2">
              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Enter password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </div>
            </div>
            <DialogFooter className="pt-4">
              <Button variant="ghost" onClick={() => handleOpenChange(false)}>Skip</Button>
              <Button onClick={onSubmitPassword} disabled={!password.trim() || !createdSiteId}>Set Password</Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default CreateSiteModal;
