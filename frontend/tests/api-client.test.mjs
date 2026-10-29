import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createAPIClient } from "../assets/api-client.js";
import { API_BASE, apiURL } from "../assets/api-config.js";

test("production requests use one team origin, credentials, and CSRF only for writes", async () => {
  const calls = [];
  const api = createAPIClient({
    getCSRF: () => "test-csrf",
    fetchImpl: async (url, options) => {
      calls.push({ url, ...options });
      return Response.json({ ok: true });
    },
  });
  await api("/api/v1/umkm/state");
  await api("/api/v1/auth/password", { method: "PUT", body: { password: "test-value" } });
  assert.equal(API_BASE, "https://temukas.madgo.my.id");
  assert.equal(calls[0].url, API_BASE + "/api/v1/umkm/state");
  assert.equal(calls[0].credentials, "include");
  assert.equal(calls[0].headers.has("X-CSRF-Token"), false);
  assert.equal(calls[1].credentials, "include");
  assert.equal(calls[1].headers.get("X-CSRF-Token"), "test-csrf");
  assert.equal(calls[1].headers.get("Content-Type"), "application/json");
  assert.equal(calls[1].method, "PUT");
});

test("team errors preserve message and request ID; 401 signals session expiry", async () => {
  let unauthorized = "";
  const api = createAPIClient({
    getCSRF: () => "",
    onUnauthorized: (path) => { unauthorized = path; },
    fetchImpl: async () => Response.json({ message: "Sesi berakhir.", request_id: "test-request" }, { status: 401 }),
  });
  await assert.rejects(api("/api/v1/auth/me"), (error) =>
    error.message === "Sesi berakhir." && error.status === 401 && error.requestId === "test-request",
  );
  assert.equal(unauthorized, "/api/v1/auth/me");
});

test("proxy HTML is not shown as a successful empty JSON response", async () => {
  const api = createAPIClient({ getCSRF: () => "", fetchImpl: async () => new Response("<html>proxy</html>") });
  await assert.rejects(api("/api/v1/config"), /Respons server tidak sesuai/);
  assert.throws(() => apiURL("/api/login"), /Alamat API tidak valid/);
  assert.throws(() => apiURL("https://other.test/api/v1/config"), /Alamat API tidak valid/);
});

test("403 recovers session metadata without replaying a financial write", async () => {
  let requests = 0;
  let recovered = 0;
  const api = createAPIClient({
    getCSRF: () => "old-csrf",
    fetchImpl: async () => { requests++; return Response.json({ message: "CSRF mismatch" }, { status: 403 }); },
    onForbidden: async () => { recovered++; return "Sesi diperbarui. Periksa isian, lalu kirim lagi."; },
  });
  await assert.rejects(api("/api/v1/umkm/allocations", { method: "POST", body: { amount: 100 } }),
    error => error.status === 403 && error.message.includes("Sesi diperbarui"));
  assert.equal(requests, 1);
  assert.equal(recovered, 1);
});

test("network failures give recovery guidance instead of raw browser text", async () => {
  const api = createAPIClient({ getCSRF: () => "", fetchImpl: async () => { throw new TypeError("Failed to fetch"); } });
  await assert.rejects(api("/api/v1/umkm/state"), /Periksa data terbaru sebelum mengirim ulang/);
});

test("Vercel publishes only the static frontend and permits direct team API connections", async () => {
  const config = JSON.parse(await readFile(new URL("../../vercel.json", import.meta.url), "utf8"));
  assert.equal(config.outputDirectory, "dist");
  assert.equal(config.buildCommand, "node tools/build-static.mjs");
  assert.equal(config.cleanUrls, true);
  assert.equal(config.framework, null);
  assert.equal(config.rewrites, undefined);
  assert.equal(config.functions, undefined);
  const csp = config.headers[0].headers.find((h) => h.key === "Content-Security-Policy").value;
  assert.match(csp, /connect-src 'self' https:\/\/temukas\.madgo\.my\.id;/);
  const source = await readFile(new URL("../assets/app.js", import.meta.url), "utf8");
  assert.match(source, /new EventSource\(apiURL\("\/api\/v1\/umkm\/events"\), \{ withCredentials: true \}\)/);
  assert.match(source, /apiURL\("\/api\/v1\/umkm\/export"\)/);
  assert.equal(source.includes('href="/api/v1/umkm/export"'), false);
});
