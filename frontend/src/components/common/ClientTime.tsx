"use client";
import * as React from "react";

export interface ClientTimeProps {
  iso?: string;
  /**
   * Optional formatter. If not provided, will use toLocaleString() on client.
   */
  format?: (d: Date) => string;
  className?: string;
}

/**
 * Client-only time text to avoid SSR/CSR hydration mismatches due to locale/timezone.
 * Renders empty string on server, then formats on client.
 */
export function ClientTime({ iso, format, className }: ClientTimeProps) {
  const [text, setText] = React.useState<string>("");

  React.useEffect(() => {
    if (!iso) {
      setText("—");
      return;
    }
    const d = new Date(iso);
    const t = format ? format(d) : d.toLocaleString();
    setText(t);
  }, [iso, format]);

  return (
    <span suppressHydrationWarning className={className}>
      {text}
    </span>
  );
}

export default ClientTime;
