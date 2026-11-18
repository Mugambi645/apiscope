import type { Api, Endpoint, Language, RequestSpec } from "./types.ts";

/** Builds the concrete request for an endpoint from the values the user typed. */
export function buildRequest(
  api: Pick<Api, "base_url">,
  ep: Endpoint,
  values: Record<string, string>,
  extraHeaders: Record<string, string> = {},
): RequestSpec {
  let path = ep.path;
  const query = new URLSearchParams();
  for (const p of ep.params ?? []) {
    const v = (values[p.name] ?? "").trim() || p.example;
    if (p.in === "path") path = path.replaceAll(`{${p.name}}`, encodeURIComponent(v));
    else if (v) query.set(p.name, v);
  }
  const qs = query.toString();
  const url = api.base_url.replace(/\/$/, "") + path + (qs ? `?${qs}` : "");
  const headers: Record<string, string> = { Accept: "application/json", ...extraHeaders };
  return { method: ep.method, url, headers };
}

const q = (s: string) => JSON.stringify(s); // JSON string literals are valid in every target language we emit for ASCII input
const headerEntries = (h: Record<string, string>) => Object.entries(h);

function curl(r: RequestSpec): string {
  const parts = [`curl -sS -X ${r.method} ${shellQuote(r.url)}`];
  for (const [k, v] of headerEntries(r.headers)) parts.push(`  -H ${shellQuote(`${k}: ${v}`)}`);
  if (r.body) parts.push(`  --data ${shellQuote(r.body)}`);
  return parts.join(" \\\n");
}

function shellQuote(s: string): string {
  return `'${s.replaceAll("'", `'\\''`)}'`;
}

function javascript(r: RequestSpec, typed: boolean): string {
  const headers = headerEntries(r.headers)
    .map(([k, v]) => `      ${q(k)}: ${q(v)},`)
    .join("\n");
  const body = r.body ? `\n    body: ${q(r.body)},` : "";
  const t = typed ? ": unknown" : "";
  return `// ${r.method} ${r.url}
(async () => {
  const res = await fetch(${q(r.url)}, {
    method: ${q(r.method)},
    headers: {
${headers}
    },${body}
  });
  console.log("status", res.status);
  const text = await res.text();
  try {
    const data${t} = JSON.parse(text);
    console.log(JSON.stringify(data, null, 2).slice(0, 2000));
  } catch {
    console.log(text.slice(0, 2000));
  }
})();
`;
}

function python(r: RequestSpec): string {
  const headers = headerEntries(r.headers)
    .map(([k, v]) => `    ${q(k)}: ${q(v)},`)
    .join("\n");
  const data = r.body ? `\n    data=${q(r.body)}.encode(),` : "";
  return `# ${r.method} ${r.url}
import json
import urllib.request

req = urllib.request.Request(
    ${q(r.url)},
    method=${q(r.method)},
    headers={
${headers}
    },${data}
)
with urllib.request.urlopen(req, timeout=10) as res:
    print("status", res.status)
    text = res.read().decode()
    try:
        print(json.dumps(json.loads(text), indent=2)[:2000])
    except ValueError:
        print(text[:2000])
`;
}

function go(r: RequestSpec): string {
  const headers = headerEntries(r.headers)
    .map(([k, v]) => `\treq.Header.Set(${q(k)}, ${q(v)})`)
    .join("\n");
  const body = r.body ? `strings.NewReader(${q(r.body)})` : "nil";
  const imports = r.body ? `"strings"\n\t` : "";
  return `// ${r.method} ${r.url}
package main

import (
	"fmt"
	"io"
	"net/http"
	${imports}"time"
)

func main() {
	req, err := http.NewRequest(${q(r.method)}, ${q(r.url)}, ${body})
	if err != nil {
		panic(err)
	}
${headers}
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 2000))
	fmt.Println("status", res.StatusCode)
	fmt.Println(string(b))
}
`;
}

function rust(r: RequestSpec): string {
  const headers = headerEntries(r.headers)
    .map(([k, v]) => `        .header(${q(k)}, ${q(v)})`)
    .join("\n");
  const body = r.body ? `\n        .body(${q(r.body)})` : "";
  const method = r.method === "POST" ? "post" : "get";
  return `// ${r.method} ${r.url}
// Cargo.toml: reqwest = { version = "0.12", features = ["blocking"] }
use std::time::Duration;

fn main() -> Result<(), reqwest::Error> {
    let client = reqwest::blocking::Client::builder()
        .timeout(Duration::from_secs(10))
        .build()?;
    let res = client
        .${method}(${q(r.url)})
${headers}${body}
        .send()?;
    println!("status {}", res.status());
    let text = res.text()?;
    println!("{}", &text[..text.len().min(2000)]);
    Ok(())
}
`;
}

export function generate(lang: Language, r: RequestSpec): string {
  switch (lang) {
    case "curl":
      return curl(r);
    case "javascript":
      return javascript(r, false);
    case "typescript":
      return javascript(r, true);
    case "python":
      return python(r);
    case "go":
      return go(r);
    case "rust":
      return rust(r);
  }
}

export const LANGUAGES: { id: Language; label: string }[] = [
  { id: "javascript", label: "JavaScript" },
  { id: "typescript", label: "TypeScript" },
  { id: "python", label: "Python" },
  { id: "go", label: "Go" },
  { id: "rust", label: "Rust" },
  { id: "curl", label: "cURL" },
];
