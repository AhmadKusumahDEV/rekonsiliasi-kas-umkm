// production calls the team API directly; localhost keeps the existing dev proxy.
const local = ["localhost", "127.0.0.1", "[::1]"].includes(globalThis.location?.hostname);
export const API_BASE = local ? "" : "https://temukas.madgo.my.id";

export function apiURL(path) {
  if (!path.startsWith("/api/v1/") || path.includes("..")) {
    throw new Error("Alamat API tidak valid.");
  }
  return API_BASE + path;
}
