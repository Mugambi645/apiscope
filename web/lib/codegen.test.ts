import { test } from "node:test";
import assert from "node:assert/strict";
import { buildRequest, generate } from "./codegen.ts";
import type { Endpoint } from "./types.ts";

const ep: Endpoint = {
  name: "Holidays",
  method: "GET",
  path: "/api/v3/PublicHolidays/{year}/{country}",
  summary: "",
  params: [
    { name: "year", in: "path", required: true, example: "2026" },
    { name: "country", in: "path", required: true, example: "KE" },
    { name: "lang", in: "query", required: false, example: "" },
  ],
};
const api = { base_url: "https://date.nager.at/" };

test("buildRequest substitutes path params and falls back to examples", () => {
  assert.equal(buildRequest(api, ep, {}).url, "https://date.nager.at/api/v3/PublicHolidays/2026/KE");
  assert.equal(buildRequest(api, ep, { year: "2027", lang: "sw" }).url, "https://date.nager.at/api/v3/PublicHolidays/2027/KE?lang=sw");
});

test("buildRequest escapes values", () => {
  const r = buildRequest(api, ep, { country: "a b/c" });
  assert.ok(r.url.includes("a%20b%2Fc"));
});

test("every language embeds the url and headers", () => {
  const r = buildRequest(api, ep, {}, { "X-API-Key": "k'ey" });
  for (const lang of ["curl", "javascript", "typescript", "python", "go", "rust"] as const) {
    const code = generate(lang, r);
    assert.ok(code.includes("https://date.nager.at/api/v3/PublicHolidays/2026/KE"), `${lang} url`);
    assert.ok(code.includes("X-API-Key"), `${lang} header`);
  }
});

test("curl quotes single quotes safely", () => {
  const r = buildRequest(api, ep, {}, { "X-Key": "k'ey" });
  assert.ok(generate("curl", r).includes(`'X-Key: k'\\''ey'`));
});

test("post bodies are emitted", () => {
  const r = { method: "POST", url: "https://x.test/a", headers: {}, body: '{"a":1}' };
  assert.ok(generate("python", r).includes("data="));
  assert.ok(generate("go", r).includes("strings.NewReader"));
  assert.ok(generate("javascript", r).includes("body:"));
});
