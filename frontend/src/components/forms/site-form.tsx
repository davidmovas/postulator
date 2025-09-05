"use client";
import * as React from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";

import type { Strategy } from "@/types/site";

export type SiteFormValues = {
  name: string;
  url: string;
  username: string;
  password: string;
  strategy: Strategy;
  is_active: boolean;
};

export function SiteForm({
  open,
  onOpenChange,
  initial,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initial?: SiteFormValues;
  onSubmit: (values: SiteFormValues) => void;
}) {
  const [values, setValues] = React.useState<SiteFormValues>(
    initial ?? { name: "", url: "", username: "", password: "", strategy: "round_robin", is_active: true }
  );

  React.useEffect(() => {
    setValues(initial ?? { name: "", url: "", username: "", password: "", strategy: "round_robin", is_active: true });
  }, [initial]);

  const submit = () => {
    onSubmit(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{initial ? "Edit Site" : "Add Site"}</DialogTitle>
          <DialogDescription>Configure WordPress connection parameters.</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3 py-2">
          <div className="grid gap-1">
            <Label htmlFor="name">Name</Label>
            <Input id="name" value={values.name} onChange={(e) => setValues({ ...values, name: e.target.value })} />
          </div>
          <div className="grid gap-1">
            <Label htmlFor="url">URL</Label>
            <Input id="url" type="url" value={values.url} onChange={(e) => setValues({ ...values, url: e.target.value })} />
          </div>
          <div className="grid gap-1 md:grid-cols-2 md:items-center md:gap-3">
            <div className="grid gap-1">
              <Label htmlFor="username">Username</Label>
              <Input id="username" value={values.username} onChange={(e) => setValues({ ...values, username: e.target.value })} />
            </div>
            <div className="grid gap-1">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" value={values.password} onChange={(e) => setValues({ ...values, password: e.target.value })} />
            </div>
          </div>
          <div className="grid gap-1">
            <Label htmlFor="strategy">Strategy</Label>
            <select
              id="strategy"
              className="h-9 w-full rounded-md border bg-background px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              value={values.strategy}
              onChange={(e) => setValues({ ...values, strategy: e.target.value as Strategy })}
            >
              <option value="unique">Unique</option>
              <option value="round_robin">Round robin</option>
              <option value="random">Random</option>
              <option value="random-all">Random All</option>
            </select>
          </div>
          <div className="flex items-center justify-between border-t pt-3 mt-1">
            <div className="flex flex-col gap-0.5">
              <Label htmlFor="is_active">Active</Label>
              <span className="text-xs text-muted-foreground">If disabled, site won&apos;t be used by schedules.</span>
            </div>
            <Switch id="is_active" checked={values.is_active} onCheckedChange={(c: boolean) => setValues({ ...values, is_active: c })} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={submit}>{initial ? "Save changes" : "Create site"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
