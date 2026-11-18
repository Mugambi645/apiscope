export type Param = {
  name: string;
  in: "query" | "path";
  required: boolean;
  example: string;
  description?: string;
};

export type Endpoint = {
  name: string;
  method: "GET" | "POST";
  path: string;
  summary: string;
  params?: Param[];
};

export type Health = {
  slug: string;
  checks: number;
  uptime_pct: number;
  p50_ms: number;
  last_ok: boolean;
  last_verified?: string;
  https: boolean;
  cors_verified: boolean;
};

export type Api = {
  slug: string;
  name: string;
  description: string;
  categories: string[];
  tags?: string[];
  regions?: string[];
  auth: "none" | "apikey" | "oauth";
  cors: boolean;
  license?: string;
  base_url: string;
  docs_url: string;
  spec_url?: string;
  endpoints: Endpoint[];
  health: Health;
};

export type Preview = {
  url: string;
  title: string;
  description: string;
  favicon: string;
};

export type RequestSpec = {
  method: string;
  url: string;
  headers: Record<string, string>;
  body?: string;
};

export type Language = "curl" | "javascript" | "typescript" | "python" | "go" | "rust";

export const RUNNABLE: Language[] = ["javascript", "python"];
