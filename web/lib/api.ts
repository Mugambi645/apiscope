import type { Api, Preview } from "./types.ts";

// Server components call the gateway directly; the browser uses the /api rewrite.
const GATEWAY = process.env.GATEWAY_URL || "http://localhost:8080";

export async function getApi(slug: string): Promise<Api | null> {
  const res = await fetch(`${GATEWAY}/api/apis/${encodeURIComponent(slug)}`, { next: { revalidate: 60 } });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`gateway returned ${res.status}`);
  return res.json();
}

export async function getPreview(slug: string): Promise<Preview | null> {
  try {
    const res = await fetch(`${GATEWAY}/api/apis/${encodeURIComponent(slug)}/preview`, { next: { revalidate: 3600 } });
    return res.ok ? res.json() : null;
  } catch {
    return null; // previews are best-effort
  }
}
