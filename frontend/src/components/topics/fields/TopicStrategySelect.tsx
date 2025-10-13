"use client";

import React from "react";
import { Label } from "@/components/ui/label";
import { TOPIC_STRATEGIES } from "@/constants/topics";
import type { TopicStrategy } from "@/constants/topics";

export interface TopicStrategySelectProps {
  value: TopicStrategy;
  onChange: (value: TopicStrategy) => void;
  disabled?: boolean;
  label?: string;
  id?: string;
  className?: string;
}

export function TopicStrategySelect({ value, onChange, disabled = false, label = "Strategy", id = "strategy", className }: TopicStrategySelectProps) {
  return (
    <div className={`space-y-2 ${className ?? ""}`}>
      <Label htmlFor={id}>{label}</Label>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value as TopicStrategy)}
        disabled={disabled}
        className="flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
      >
        {TOPIC_STRATEGIES.map((s) => (
          <option key={s} value={s} className="capitalize">
            {s}
          </option>
        ))}
      </select>
    </div>
  );
}

export default TopicStrategySelect;
