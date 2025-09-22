export interface Topic {
  id: number;
  title: string;
  keywords?: string;
  category?: string;
  tags?: string;
  created_at?: string; // ISO
  updated_at?: string; // ISO
}
