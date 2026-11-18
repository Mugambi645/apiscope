"use client";

import { useEffect, useMemo, useState } from "react";
import type { Api, Language } from "@/lib/types";
import { RUNNABLE } from "@/lib/types";
import { buildRequest, generate, LANGUAGES } from "@/lib/codegen";
import { prettyBody } from "@/lib/format";

type TryResult = { status: number; status_text: string; headers: Record<string, string>; body: string; truncated: boolean; latency_ms: number };
type RunResult = { stdout: string; stderr: string; exit_code: number; duration_ms: number; timed_out: boolean; truncated: boolean };

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error((data as { error?: string }).error ?? `request failed (${res.status})`);
  return data as T;
}

export default function Playground({ api }: { api: Api }) {
  const [epIndex, setEpIndex] = useState(0);
  const ep = api.endpoints[epIndex];
  const [values, setValues] = useState<Record<string, string>>({});
  const [keyName, setKeyName] = useState(api.auth === "none" ? "" : "Authorization");
  const [keyValue, setKeyValue] = useState("");
  const [tab, setTab] = useState<"try" | Language>("try");

  const extra = useMemo(() => (keyName && keyValue ? { [keyName]: keyValue } : {}), [keyName, keyValue]);
  const request = useMemo(() => buildRequest(api, ep, values, extra), [api, ep, values, extra]);

  // Generated code is editable; we only regenerate when the user has not edited it.
  const [edited, setEdited] = useState<Record<string, string>>({});
  const generated = tab === "try" ? "" : generate(tab, request);
  const code = tab === "try" ? "" : (edited[tab] ?? generated);
  useEffect(() => setEdited({}), [epIndex]);

  const [tryRes, setTryRes] = useState<TryResult | null>(null);
  const [runRes, setRunRes] = useState<RunResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  async function tryIt() {
    setBusy(true); setError(""); setTryRes(null);
    try {
      setTryRes(await postJSON<TryResult>("/api/proxy", request));
    } catch (e) { setError((e as Error).message); } finally { setBusy(false); }
  }

  async function run() {
    if (tab === "try") return;
    setBusy(true); setError(""); setRunRes(null);
    try {
      setRunRes(await postJSON<RunResult>("/api/run", { language: tab, code }));
    } catch (e) { setError((e as Error).message); } finally { setBusy(false); }
  }

  async function copy() {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  const runnable = tab !== "try" && RUNNABLE.includes(tab);

  return (
    <section className="playground" aria-label="Playground">
      <div className="row">
        <label>
          Endpoint
          <select value={epIndex} onChange={(e) => { setEpIndex(Number(e.target.value)); setValues({}); setTryRes(null); setRunRes(null); }}>
            {api.endpoints.map((e, i) => (
              <option key={e.name} value={i}>{e.method} {e.name}</option>
            ))}
          </select>
        </label>
        <code className="url">{request.url}</code>
      </div>
      <p className="muted">{ep.summary}</p>

      {(ep.params ?? []).length > 0 && (
        <div className="params">
          {(ep.params ?? []).map((p) => (
            <label key={p.name}>
              {p.name} <span className="muted">({p.in}{p.required ? ", required" : ""})</span>
              <input value={values[p.name] ?? ""} placeholder={p.example} onChange={(e) => setValues({ ...values, [p.name]: e.target.value })} />
            </label>
          ))}
        </div>
      )}

      {api.auth !== "none" && (
        <div className="params">
          <label>Auth header name<input value={keyName} onChange={(e) => setKeyName(e.target.value)} /></label>
          <label>
            Auth header value
            <input type="password" autoComplete="off" value={keyValue} onChange={(e) => setKeyValue(e.target.value)} />
          </label>
          <p className="muted">Your key stays in this browser tab and is sent only with the request you make. It is never stored.</p>
        </div>
      )}

      <div className="tabs" role="tablist">
        <button role="tab" aria-selected={tab === "try"} onClick={() => setTab("try")}>Try it</button>
        {LANGUAGES.map((l) => (
          <button key={l.id} role="tab" aria-selected={tab === l.id} onClick={() => setTab(l.id)}>{l.label}</button>
        ))}
      </div>

      {tab === "try" ? (
        <div>
          <button className="primary" disabled={busy} onClick={tryIt}>{busy ? "Sending…" : "Send request"}</button>
          {tryRes && (
            <div className="output">
              <p>
                <strong className={tryRes.status < 400 ? "ok" : "err"}>{tryRes.status} {tryRes.status_text}</strong>
                {" · "}{tryRes.latency_ms} ms{tryRes.truncated ? " · truncated at 1 MB" : ""}
              </p>
              <pre>{prettyBody(tryRes.body)}</pre>
            </div>
          )}
        </div>
      ) : (
        <div>
          <textarea
            aria-label={`${tab} code`} spellCheck={false} rows={Math.min(24, code.split("\n").length + 1)}
            value={code} onChange={(e) => setEdited({ ...edited, [tab]: e.target.value })}
          />
          <div className="row">
            {runnable ? (
              <button className="primary" disabled={busy} onClick={run}>{busy ? "Running…" : "Run in sandbox"}</button>
            ) : (
              <span className="muted">Sandbox execution for this language is planned; copy the snippet and run it locally.</span>
            )}
            <button onClick={copy}>{copied ? "Copied" : "Copy"}</button>
            {edited[tab] !== undefined && <button onClick={() => { const n = { ...edited }; delete n[tab]; setEdited(n); }}>Reset</button>}
          </div>
          {runRes && (
            <div className="output">
              <p>exit {runRes.exit_code}{runRes.timed_out ? " · timed out" : ""} · {runRes.duration_ms} ms{runRes.truncated ? " · output truncated" : ""}</p>
              {runRes.stdout && <pre>{runRes.stdout}</pre>}
              {runRes.stderr && <pre className="err">{runRes.stderr}</pre>}
            </div>
          )}
        </div>
      )}
      {error && <p className="error" role="alert">{error}</p>}
    </section>
  );
}
