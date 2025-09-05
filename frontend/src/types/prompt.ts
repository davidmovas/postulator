export interface Prompt {
  id: number;
  name: string;
  system: string;
  user: string;
  is_default: boolean;
  is_active: boolean;
  created_at?: string; // ISO
  updated_at?: string; // ISO
}
