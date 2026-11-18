"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import type { Api } from "@/lib/types";
import HealthBadge from "./HealthBadge";

type Category = { name: string; count: number };

export default function SearchClient() {
  const [q, setQ] = useState("");
  const [category, setCategory] = useState("");
  const [auth, setAuth] = useState("");
  const [corsOnly, setCorsOnly] = useState(false);
  const [region, setRegion] = useState("");
  const [items, setItems] = useState<Api[]>([]);
  const [cats, setCats] = useState<Category[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/categories").then((r) => r.json()).then(setCats).catch(() => {});
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    const t = setTimeout(async () => {
      const p = new URLSearchParams();
      if (q) p.set("q", q);
      if (category) p.set("category", category);
      if (auth) p.set("auth", auth);
      if (region) p.set("region", region);
      if (corsOnly) p.set("cors", "true");
      setLoading(true);
      try {
        const res = await fetch(`/api/apis?${p}`, { signal: ctrl.signal });
        if (!res.ok) throw new Error(`gateway returned ${res.status}`);
        setItems((await res.json()).items ?? []);
        setError("");
      } catch (e) {
        if ((e as Error).name !== "AbortError") setError("Could not load the catalog. Is the gateway running?");
      } finally {
        setLoading(false);
      }
    }, 200); // debounce typing
    return () => {
      clearTimeout(t);
      ctrl.abort();
    };
  }, [q, category, auth, region, corsOnly]);

  return (
    <>
      <div className="searchbar">
        <input
          aria-label="Search APIs"
          placeholder='Try "weather", "currency", "kenya" or "testing"'
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
      <div className="filters">
        <select aria-label="Category" value={category} onChange={(e) => setCategory(e.target.value)}>
          <option value="">All categories</option>
          {cats.map((c) => (
            <option key={c.name} value={c.name}>{c.name} ({c.count})</option>
          ))}
        </select>
        <select aria-label="Auth type" value={auth} onChange={(e) => setAuth(e.target.value)}>
          <option value="">Any auth</option>
          <option value="none">No key needed</option>
          <option value="apikey">API key</option>
          <option value="oauth">OAuth</option>
        </select>
        <select aria-label="Region" value={region} onChange={(e) => setRegion(e.target.value)}>
          <option value="">Any region</option>
          <option value="kenya">Kenya</option>
          <option value="africa">Africa</option>
        </select>
        <label className="check">
          <input type="checkbox" checked={corsOnly} onChange={(e) => setCorsOnly(e.target.checked)} />
          Works from the browser (CORS)
        </label>
      </div>

      {error && <p className="error" role="alert">{error}</p>}
      <p className="muted" aria-live="polite">{loading ? "Searching…" : `${items.length} APIs`}</p>

      <ul className="grid">
        {items.map((a) => (
          <li key={a.slug} className="card">
            <Link href={`/apis/${a.slug}`}>
              <h2>{a.name}</h2>
            </Link>
            <p>{a.description}</p>
            <div className="row">
              <HealthBadge health={a.health} />
              <span className="badge badge-muted">{a.auth === "none" ? "no key" : a.auth}</span>
              {a.cors && <span className="badge badge-muted">CORS</span>}
            </div>
            <div className="tags">
              {a.categories.map((c) => (
                <button key={c} className="tag" onClick={() => setCategory(c)}>{c}</button>
              ))}
            </div>
          </li>
        ))}
      </ul>
      {!loading && !error && items.length === 0 && <p>No APIs match. Loosen a filter or clear the search.</p>}
    </>
  );
}
