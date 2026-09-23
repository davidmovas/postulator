import { copy } from "../../copy/index.js";
import type { SelectOption } from "../../ui/index.js";
import { flagLabel } from "./labels.js";
import type { SpecPath } from "./patch.js";
import type { SpecDraft } from "./spec.js";

export const unsetSource = "none";

interface Common {
    key: string;
    label: string;
    hint?: string;
    field: string;
    path: SpecPath;
}

export type Rule =
    | (Common & {
          kind: "switch";
          read: (draft: SpecDraft) => boolean;
          write: (value: boolean) => Partial<SpecDraft>;
      })
    | (Common & {
          kind: "number";
          min?: number;
          max?: number;
          step?: number;
          read: (draft: SpecDraft) => number;
          write: (value: number) => Partial<SpecDraft>;
          show?: (draft: SpecDraft) => string;
      })
    | (Common & {
          kind: "text";
          mono?: boolean;
          read: (draft: SpecDraft) => string;
          write: (value: string) => Partial<SpecDraft>;
      })
    | (Common & {
          kind: "select";
          options: readonly SelectOption<string>[];
          read: (draft: SpecDraft) => string;
          write: (value: string) => Partial<SpecDraft>;
          show: (draft: SpecDraft) => string;
      });

export interface RuleBlock {
    key: string;
    title: string;
    rules: readonly Rule[];
}

export function shown(rule: Rule, draft: SpecDraft): string {
    switch (rule.kind) {
        case "switch":
            return flagLabel(rule.read(draft));
        case "number":
            return rule.show === undefined ? String(rule.read(draft)) : rule.show(draft);
        case "select":
            return rule.show(draft);
        default: {
            const held = rule.read(draft);
            return held === "" ? copy.templates.layer.none : held;
        }
    }
}
