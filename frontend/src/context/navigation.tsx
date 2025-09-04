"use client";
import * as React from "react";

export type Section = "dashboard" | "jobs" | "sites" | "titles" | "settings";

export interface NavigationState {
  section: Section;
  setSection: (s: Section) => void;
}

const NavigationContext = React.createContext<NavigationState | null>(null);

export function useNavigation() {
  const ctx = React.useContext(NavigationContext);
  if (!ctx) throw new Error("useNavigation must be used within <NavigationProvider>");
  return ctx;
}

export function NavigationProvider({
  initial = "dashboard",
  children,
}: {
  initial?: Section;
  children: React.ReactNode;
}) {
  const [section, setSection] = React.useState<Section>(initial);
  const value = React.useMemo(() => ({ section, setSection }), [section]);
  return <NavigationContext.Provider value={value}>{children}</NavigationContext.Provider>;
}
