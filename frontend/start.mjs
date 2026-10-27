// Local launcher owns only the two children it starts; it never kills occupied ports.
import { spawn } from "node:child_process";
import { mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import net from "node:net";

const frontend = dirname(fileURLToPath(import.meta.url));
const root = dirname(frontend);
const backend = join(root, "backend");
const apiPort = Number(process.env.SERVER_PORT || 8080);
const uiPort = Number(process.env.PORT || 3000);
const children = new Set();
let stopping = false;
function stop(code = 0) {
  if (stopping) return;
  stopping = true;
  for (const child of children) child.kill();
  process.exitCode = code;
}
for (const signal of ["SIGINT", "SIGTERM"]) process.on(signal, () => stop());
async function available(port) {
  await new Promise((resolve, reject) => {
    const socket = net.createServer();
    socket.once("error", () => reject(new Error("Port " + port + " sedang dipakai. Tutup server milikmu atau pilih port lain.")));
    socket.listen(port, "127.0.0.1", () => socket.close(resolve));
  });
}
function start(command, args, cwd, env) {
  const child = spawn(command, args, { cwd, env, stdio: "inherit", windowsHide: true });
  children.add(child);
  child.on("error", error => { console.error(error.message); stop(1); });
  child.on("exit", code => { children.delete(child); if (!stopping) stop(code || 0); });
  return child;
}
try {
  if (apiPort === uiPort) throw new Error("PORT dan SERVER_PORT harus berbeda.");
  await available(apiPort);
  await available(uiPort);
  await mkdir(join(root, ".cache"), { recursive: true });
  const binary = join(root, ".cache", process.platform === "win32" ? "temukas-api.exe" : "temukas-api");
  const build = spawn("go", ["build", "-o", binary, "./cmd/api"], { cwd: backend, stdio: "inherit", windowsHide: true });
  children.add(build);
  const code = await new Promise((resolve, reject) => {
    build.once("error", reject);
    build.once("exit", resolve);
  });
  children.delete(build);
  if (code !== 0 || stopping) throw new Error("Build backend belum berhasil.");
  start(binary, [], backend, { ...process.env, SERVER_HOST: "127.0.0.1", SERVER_PORT: String(apiPort) });
  start(process.execPath, ["dev.mjs"], frontend, {
    ...process.env, PORT: String(uiPort), API_UPSTREAM: "http://127.0.0.1:" + apiPort,
  });
  console.log("Ctrl+C menghentikan kedua server. Konfigurasi database dibaca dari backend/configs/local.yaml.");
} catch (error) {
  console.error(error.message);
  stop(1);
}
