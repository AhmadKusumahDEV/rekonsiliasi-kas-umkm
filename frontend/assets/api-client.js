import { apiURL } from "./api-config.js?v=team2";

export function createAPIClient({ getCSRF, onUnauthorized, onForbidden, fetchImpl = globalThis.fetch }) {
  return async function api(path, options = {}) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 32000);
    const method = (options.method || "GET").toUpperCase();
    const headers = new Headers(options.headers || {});
    const csrf = getCSRF();
    if (!["GET", "HEAD"].includes(method) && csrf) headers.set("X-CSRF-Token", csrf);
    let body = options.body;
    if (body !== undefined && !(body instanceof FormData)) {
      headers.set("Content-Type", "application/json");
      body = JSON.stringify(body);
    }
    try {
      const response = await fetchImpl(apiURL(path), {
        ...options, method, body, headers,
        credentials: "include",
        signal: controller.signal,
      });
      const json = (response.headers.get("content-type") || "").includes("application/json");
      const data = json ? await response.json().catch(() => null) : null;
      if (!response.ok) {
        const error = new Error(data?.message || `Permintaan gagal (${response.status}). Coba lagi.`);
        error.status = response.status;
        error.requestId = data?.request_id;
        if (response.status === 401) onUnauthorized?.(path);
        if (response.status === 403 && onForbidden) {
          const message = await Promise.resolve().then(() => onForbidden(path)).catch(() => null);
          if (message) error.message = message;
        }
        throw error;
      }
      if (data === null) throw new Error("Respons server tidak sesuai. Coba muat ulang halaman.");
      return data;
    } catch (error) {
      if (error instanceof TypeError) {
        throw new Error("Koneksi ke server terputus. Periksa data terbaru sebelum mengirim ulang.");
      }
      if (error.name === "AbortError") {
        throw new Error("Server belum merespons. Data mungkin tersimpan; muat ulang sebelum mengirim lagi.");
      }
      throw error;
    } finally {
      clearTimeout(timeout);
    }
  };
}
