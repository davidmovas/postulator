"use client";

import React from "react";
import { I18nProvider, Group } from "react-aria-components";
import { TimeField, DateInput, dateInputStyle } from "@/components/ui/datefield-rac";
import { parseTime } from "@internationalized/date";

type Props = {
  value: string; // HH:mm
  onChange: (value: string) => void;
  isInvalid?: boolean;
  className?: string; // additional classes for the wrapper input
  widthClassName?: string; // e.g., w-[120px]
};

/**
 * TimePicker24
 * A reusable 24-hour time picker built on react-aria TimeField with en-GB locale.
 * Avoids double borders by rendering DateInput with unstyled and applying the input chrome to Group.
 */
export default function TimePicker24({ value, onChange, isInvalid, className, widthClassName = "w-[120px]" }: Props) {
  return (
    <I18nProvider locale="en-GB">
      <TimeField
        value={parseTime(value)}
        granularity="minute"
        onChange={(v) => {
          if (!v) return;
          const h = String(v.hour ?? 0).padStart(2, "0");
          const m = String(v.minute ?? 0).padStart(2, "0");
          onChange(`${h}:${m}`);
        }}
        isInvalid={isInvalid}
      >
        <Group className={`${dateInputStyle} ${widthClassName} ${className ?? ""}`.trim()}>
          <DateInput unstyled />
        </Group>
      </TimeField>
    </I18nProvider>
  );
}
