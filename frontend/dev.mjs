// Local static server and same-origin proxy; production can use Caddy/Nginx.
import http from "node:http";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = dirname(fileURLToPath(import.meta.url));
const upstream = new URL(process.env.API_UPSTREAM || "http://127.0.0.1:8080");
const port = Number(process.env.PORT || 3000);
if (upstream.protocol !== "http:")
  throw new Error("Local API_UPSTREAM must use http.");
const types = {
  ".html": "text/html; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".jpg": "image/jpeg",
  ".png": "image/png",
  ".woff2": "font/woff2",
  ".ttf": "font/ttf",
  ".csv": "text/csv; charset=utf-8",
  ".txt": "text/plain; charset=utf-8",
};
const server = http.createServer(async (req, res) => {
  const pathname = new URL(req.url, "http://localhost").pathname;
  if (pathname.startsWith("/api/v1/") || pathname.startsWith("/health/")) {
    const proxy = http.request(
      new URL(req.url, upstream),
      {
        method: req.method,
        headers: {
          ...req.headers,
          "x-forwarded-proto": "http",
          "x-forwarded-for": req.socket.remoteAddress || "",
        },
        timeout: 35000,
      },
      (response) => {
        res.writeHead(response.statusCode, response.headers);
        response.pipe(res);
      },
    );
    proxy.on("timeout", () => proxy.destroy(new Error("upstream timeout")));
    proxy.on("error", () => {
      if (!res.headersSent)
        res.writeHead(502, { "Content-Type": "application/json" });
      if (!res.writableEnded)
        res.end(
          JSON.stringify({
            message:
              "Backend belum tersambung. Jalankan API Go terlebih dahulu.",
          }),
        );
    });
    req.on("aborted", () => proxy.destroy());
    res.on("close", () => {
      if (!res.writableEnded) proxy.destroy();
    });
    req.pipe(proxy);
    return;
  }
  if (!["GET", "HEAD"].includes(req.method)) {
    res.writeHead(405);
    res.end();
    return;
  }
  const relative =
    pathname === "/" || pathname === "/index.html"
      ? "index.html"
      : /^\/assets\/[a-zA-Z0-9_.-]+$/.test(pathname) && !pathname.includes("..")
        ? pathname.slice(1)
        : "";
  if (!relative) {
    res.writeHead(404);
    res.end("Not found");
    return;
  }
  try {
    const body = await readFile(join(root, relative));
    const extension = relative.slice(relative.lastIndexOf("."));
    res.writeHead(200, {
      "Content-Type": types[extension] || "application/octet-stream",
      "Cache-Control": "no-cache",
      "X-Content-Type-Options": "nosniff",
      "X-Frame-Options": "DENY",
      "Referrer-Policy": "same-origin",
      "Content-Security-Policy":
        "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'",
    });
    res.end(req.method === "HEAD" ? undefined : body);
  } catch {
    res.writeHead(404);
    res.end("Not found");
  }
});
server.on("error", (error) => {
  console.error(error.message);
  process.exitCode = 1;
});
server.listen(port, "127.0.0.1", () =>
  console.log("Temu Kas: http://127.0.0.1:" + port),
);
for (const signal of ["SIGINT", "SIGTERM"])
  process.on(signal, () => server.close());
