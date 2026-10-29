import test from "node:test";
import assert from "node:assert/strict";
import http from "node:http";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { fileURLToPath } from "node:url";

test("static server isolates files and proxies session/CSRF requests", async () => {
  const api = http.createServer((req, res) => {
    res.setHeader("Content-Type", "application/json");
    res.setHeader("Set-Cookie", "rku_session=test; HttpOnly; SameSite=Strict");
    res.end(JSON.stringify({ method: req.method, csrf: req.headers["x-csrf-token"] }));
  });
  api.listen(0, "127.0.0.1");
  await once(api, "listening");
  const reserve = http.createServer();
  reserve.listen(0, "127.0.0.1");
  await once(reserve, "listening");
  const port = reserve.address().port;
  await new Promise(resolve => reserve.close(resolve));
  const child = spawn(process.execPath, [fileURLToPath(new URL("../dev.mjs", import.meta.url))], {
    env: { ...process.env, PORT: String(port), API_UPSTREAM: "http://127.0.0.1:" + api.address().port },
    stdio: ["ignore", "pipe", "pipe"],
  });
  try {
    await Promise.race([
      once(child.stdout, "data"),
      once(child, "exit").then(() => { throw new Error("dev server failed to start"); }),
      new Promise((_, reject) => { const t = setTimeout(() => reject(new Error("startup timeout")), 10000); t.unref(); }),
    ]);
    const base = "http://127.0.0.1:" + port;
    const home = await fetch(base);
    assert.equal(home.status, 200);
    assert.match(await home.text(), /landing\/assets\/landing\.js/);
    assert.match(await (await fetch(base + "/app")).text(), /assets\/app\.js/);
    assert.equal((await fetch(base + "/landing/assets/landing.css")).status, 200);
    assert.equal((await fetch(base + "/landing/assets/landing.js")).status, 200);
    assert.match((await fetch(base + "/sw.js")).headers.get("content-type"), /javascript/);
    assert.match((await fetch(base + "/pwa/manifest.webmanifest")).headers.get("content-type"), /manifest\+json/);
    assert.equal((await fetch(base + "/pwa/offline")).status, 200);
    assert.match(home.headers.get("content-security-policy"), /connect-src 'self'/);
    assert.equal((await fetch(base + "/assets/efa-ledger-4k.jpg")).status, 200);
    assert.equal((await fetch(base + "/assets/motion.js")).status, 200);
    const favicon = await fetch(base + "/assets/favicon.svg");
    assert.equal(favicon.headers.get("content-type"), "image/svg+xml");
    assert.match(await favicon.text(), /Easy Financial Audit/);
    assert.equal((await fetch(base + "/assets/style.css", {method:"HEAD"})).headers.get("content-type"), "text/css; charset=utf-8");
    for (const path of ["/.env", "/backend/configs/local.yaml", "/../.git/config", "/assets/../.env", "/landing/README.md", "/landing/assets/../../.env", "/pwa/README.md", "/pwa/service-worker.js", "/pwa/../.env"]) {
      assert.equal((await fetch(base + path)).status, 404);
    }
    assert.equal((await fetch(base, {method:"POST"})).status, 405);
    const response = await fetch(base + "/api/v1/auth/password", {
      method: "PUT", headers: { "X-CSRF-Token": "test-csrf" }, body: "{}",
    });
    assert.match(response.headers.get("set-cookie"), /HttpOnly/);
    assert.deepEqual(await response.json(), {method:"PUT", csrf:"test-csrf"});
  } finally {
    child.kill();
    await new Promise(resolve => api.close(resolve));
  }
});
