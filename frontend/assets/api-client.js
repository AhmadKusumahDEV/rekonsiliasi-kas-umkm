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
    const multipart = body instanceof FormData;
    if (multipart) headers.delete('Content-Type');
    if (body !== undefined && !(body instanceof FormData)) {
      headers.set("Content-Type", "application/json");
      body = JSON.stringify(body);
    }
    try {
      const response = multipart && options.onUploadProgress && globalThis.XMLHttpRequest
        ? await uploadResponse(apiURL(path), {method,body,headers,signal:controller.signal,onProgress:options.onUploadProgress})
        : await fetchImpl(apiURL(path), {
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
        error.details = data?.details;
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

// XMLHttpRequest exposes real upload progress; response handling stays in api().
function uploadResponse(url, {method,body,headers,signal,onProgress}) {
  return new Promise((resolve,reject)=>{
    const xhr = new XMLHttpRequest();
    const abort = () => xhr.abort();
    xhr.open(method,url,true);
    xhr.withCredentials = true;
    headers.forEach((value,name)=>xhr.setRequestHeader(name,value));
    xhr.upload.addEventListener('progress',event=>onProgress({phase:'upload',loaded:event.loaded,total:event.lengthComputable?event.total:0}));
    xhr.upload.addEventListener('load',()=>onProgress({phase:'processing'}));
    xhr.addEventListener('load',()=>{
      if (!xhr.status) return reject(new TypeError('Network unavailable'));
      resolve(new Response(xhr.responseText,{status:xhr.status,headers:{'Content-Type':xhr.getResponseHeader('Content-Type')||''}}));
    });
    xhr.addEventListener('error',()=>reject(new TypeError('Network unavailable')));
    xhr.addEventListener('abort',()=>reject(new DOMException('Request aborted','AbortError')));
    xhr.addEventListener('loadend',()=>signal.removeEventListener('abort',abort));
    signal.addEventListener('abort',abort,{once:true});
    if(signal.aborted) { reject(new DOMException('Request aborted','AbortError')); return; }
    xhr.send(body);
  });
}
