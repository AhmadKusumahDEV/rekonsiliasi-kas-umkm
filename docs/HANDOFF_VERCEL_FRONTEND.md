# Serah Terima: Frontend Vercel efa.madgo.my.id ke Backend Temukas

Dokumen ini untuk pengelola project Vercel `efa.madgo.my.id`. Tujuannya
menjelaskan perubahan yang diperlukan agar frontend di Vercel memakai backend
tim di `https://temukas.madgo.my.id`, bukan fungsi `/api/...` milik prototipe.

Tanggal: 28 Oktober 2026.

## 1. Ringkasan

Saat ini `efa.madgo.my.id` adalah aplikasi monolit di Vercel: HTML/CSS/JS,
fungsi Go di `/api/...`, database sendiri, dan sinkronisasi memakai polling
(`realtime: "poll"`). Backend tim adalah proses Go/Echo terpisah di
`temukas.madgo.my.id` dengan PostgreSQL, endpoint `/api/v1/...`, dan SSE.

Yang berubah adalah **lapisan API di frontend**. Tampilan, routing hash, dan
alur halaman tidak perlu dibongkar. Backend sudah menyediakan seluruh endpoint
yang dibutuhkan frontend.

## 2. Kondisi sekarang dan target

| Aspek | Sekarang (Vercel) | Target (Temukas) |
| --- | --- | --- |
| Base API | `/api/...` di domain sendiri | `https://temukas.madgo.my.id/api/v1` |
| Cookie sesi | `pbl_umkm` | `rku_session` |
| Database | Milik project Vercel | PostgreSQL tim |
| Realtime | Polling | SSE `GET /api/v1/umkm/events` |
| Error | `{"error": "..."}` | `{"code", "status", "message", "request_id"}` |

## 3. Yang sudah disiapkan di backend

- Origin `https://efa.madgo.my.id` diizinkan untuk CORS dan handshake SSE.
- Cookie `rku_session` bersifat `HttpOnly`, `SameSite=Strict`, `Path=/`, dan
  `Secure` ketika request melewati HTTPS (`X-Forwarded-Proto: https`).
- Sesi berlaku 12 jam.
- CSRF wajib untuk semua request non-GET melalui header `X-CSRF-Token`.
- Backend menolak origin yang tidak terdaftar. Origin harus ditulis persis,
  tanpa trailing slash.

Konfigurasi backend yang dipakai:

```env
CORS_ALLOWED_ORIGINS=https://temukas.madgo.my.id,https://efa.madgo.my.id
```

## 4. Perubahan yang diperlukan di frontend

### 4.1 Satu konstanta base URL

Semua request harus memakai satu sumber alamat. Jangan menyebar domain di
banyak file.

```js
export const API_BASE = "https://temukas.madgo.my.id";
```

Berlaku untuk `fetch`, `EventSource`, dan tautan unduhan.

### 4.2 Pemetaan endpoint

| Prototipe (sekarang) | Backend tim |
| --- | --- |
| `GET /api/config` | `GET /api/v1/config` |
| `POST /api/register` | `POST /api/v1/auth/register` |
| `POST /api/login` | `POST /api/v1/auth/login` |
| `GET /api/me` | `GET /api/v1/auth/me` |
| `POST /api/logout` | `POST /api/v1/auth/logout` |
| `POST /api/password` | `PUT /api/v1/auth/password` |
| `GET /api/state` | `GET /api/v1/umkm/state` |
| `GET /api/suggestions` | `GET /api/v1/umkm/suggestions` |
| `POST /api/imports/preview` | `POST /api/v1/umkm/imports/preview` |
| `POST /api/imports` | `POST /api/v1/umkm/imports` |
| `POST /api/imports/{id}/cancel` | `POST /api/v1/umkm/imports/{id}/cancel` |
| `POST /api/allocations` | `POST /api/v1/umkm/allocations` |
| `POST /api/allocations/{id}/undo` | `POST /api/v1/umkm/allocations/{id}/undo` |
| `GET /api/export` | `GET /api/v1/umkm/export` |
| Polling `/api/version` | SSE `GET /api/v1/umkm/events` |

Perhatikan perubahan method: ganti password memakai `PUT`, bukan `POST`.

### 4.3 Bentuk respons yang berbeda

`GET /api/v1/umkm/state` mengembalikan:

```json
{
  "workspace": { "id": "...", "name": "...", "code": "...", "version": 4 },
  "state": { "entries": [], "imports": [], "allocations": [] },
  "users": [],
  "events": []
}
```

`GET /api/v1/umkm/suggestions` mengembalikan **array langsung**, bukan objek.

`GET /api/v1/config` tidak memiliki `realtime`:

```json
{ "kind": "umkm", "brand": "Easy Financial Audit", "tagline": "...", "roles": ["owner"] }
```

Error selalu memakai bentuk berikut. Frontend harus membaca `message`, bukan
`error`.

```json
{
  "code": 409,
  "status": "conflict",
  "message": "Alokasi melebihi sisa tagihan...",
  "request_id": "..."
}
```

### 4.4 Cookie, credentials, dan CSRF

- Semua `fetch` wajib memakai `credentials: "include"`.
- CSRF diambil dari body respons login/register atau `GET /api/v1/auth/me`,
  disimpan di memori, lalu dikirim sebagai `X-CSRF-Token` pada request non-GET.
- Jangan menyimpan token di `localStorage`. Cookie sesi sudah `HttpOnly`.

Contoh helper:

```js
async function api(path, { method = "GET", body, csrf = "" } = {}) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (csrf) headers["X-CSRF-Token"] = csrf;
  const response = await fetch(API_BASE + path, {
    method,
    headers,
    credentials: "include",
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const type = response.headers.get("content-type") || "";
  const data = type.includes("application/json") ? await response.json() : await response.text();
  if (!response.ok) {
    const error = new Error((data && data.message) || "Permintaan gagal");
    error.status = response.status;
    throw error;
  }
  return data;
}
```

### 4.5 SSE

```js
const stream = new EventSource(API_BASE + "/api/v1/umkm/events", {
  withCredentials: true,
});
stream.addEventListener("state", (event) => {
  const version = Number(event.data);
  if (version !== currentWorkspaceVersion) refreshState();
});
```

Event `state` hanya membawa nomor versi workspace, bukan seluruh data. Setelah
versi berubah, ambil ulang `GET /api/v1/umkm/state`. Tidak ada replay riwayat;
setiap reconnect harus mengambil snapshot.

Fallback polling tetap boleh dipakai: 15 detik saat stream tertutup, 60 detik
saat stream terbuka, dan hanya ketika tab terlihat.

### 4.6 Unduhan CSV

Gunakan URL absolut:

```js
const exportURL = API_BASE + "/api/v1/umkm/export";
```

Atribut `download` tidak berlaku lintas origin. Server sudah mengirim
`Content-Disposition: attachment`, jadi navigasi biasa atau `window.location`
cukup. Alternatif lain: `fetch` dengan `credentials: "include"`, lalu buat blob.

### 4.7 CSP

Jika hosting mengirim Content-Security-Policy, tambahkan backend ke
`connect-src`:

```text
connect-src 'self' https://temukas.madgo.my.id
```

Tanpa ini, browser akan memblokir fetch dan EventSource meskipun CORS benar.

### 4.8 Penanganan 401

401 pada `/auth/me` saat halaman pertama dibuka berarti belum login. Tampilkan
form login, bukan layar “aplikasi belum tersambung”. 401 pada request lain
berarti sesi berakhir; hentikan stream dan kembali ke login.

### 4.9 Hapus pemanggilan legacy

Setelah migrasi, tidak boleh ada lagi panggilan ke `/api/...` prototipe. Jika
masih ada, aplikasi akan mencampur dua database dan hasilnya membingungkan.

## 5. Hal yang tidak boleh dilakukan

- Jangan memakai rewrite Vercel dari `/api/v1/*` ke `temukas.madgo.my.id`.
  Rewrite berjalan di server Vercel dan tidak membawa cookie browser.
- Jangan memakai domain preview `*.vercel.app`. Cookie `SameSite=Strict`
  membutuhkan situs yang sama; gunakan `efa.madgo.my.id`.
- Jangan memakai `Access-Control-Allow-Origin: *` untuk request ber-credential.
- Jangan menyimpan password, token, atau cookie di `localStorage`.

## 6. Checklist pengujian

### 6.1 Preflight CORS

```bash
curl -i -X OPTIONS https://temukas.madgo.my.id/api/v1/umkm/state \
  -H "Origin: https://efa.madgo.my.id" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: content-type,x-csrf-token"
```

Harus ada `Access-Control-Allow-Origin: https://efa.madgo.my.id` dan
`Access-Control-Allow-Credentials: true`.

### 6.2 Register dan login

```bash
curl -i -c cookies.txt -X POST https://temukas.madgo.my.id/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo","email":"demo@example.test","password":"DemoPass!2026","workspace":"Demo"}'
```

Respons `201` berisi `user` dan `csrf`. Setelah itu:

```bash
curl -s -b cookies.txt https://temukas.madgo.my.id/api/v1/auth/me
```

Akun demo `pemilik@temukas.test` hanya bisa dipakai setelah seed dijalankan di
database tujuan. Sebelum itu, gunakan register.

### 6.3 State dan impor

```bash
curl -s -b cookies.txt https://temukas.madgo.my.id/api/v1/umkm/state
```

`POST /api/v1/umkm/imports/preview` menerima CSV sebagai teks di dalam JSON,
bukan multipart:

```json
{ "kind": "sale", "csv": "id,tanggal,referensi,nama,nominal,kanal,catatan\nP-1,2026-10-28,INV-1,Rina,10000,transfer,Contoh" }
```

### 6.4 SSE

Buka stream di terminal, lalu lakukan impor atau alokasi dari UI. Harus muncul:

```text
event: state
data: 2
```

Tanpa sesi, endpoint menjawab 401 dengan teks biasa. Itu normal untuk handshake
SSE.

### 6.5 Ekspor dan logout

- `GET /api/v1/umkm/export` mengunduh `umkm-rekap.csv`.
- `POST /api/v1/auth/logout` memakai CSRF dan mengakhiri sesi.

## 7. Catatan domain dan cookie

`efa.madgo.my.id` dan `temukas.madgo.my.id` berada di registrable domain yang
sama, `madgo.my.id`, dan sama-sama HTTPS. Karena itu cookie `SameSite=Strict`
tetap dikirim untuk request dari efa ke temukas. Ini tidak berlaku untuk domain
preview Vercel atau domain lain; pada kasus itu cookie harus `SameSite=None`
dan `Secure`, yang menambah risiko dan tidak disarankan.

## 8. Rollback

Perubahan ini tidak mengubah DNS. Jika integrasi gagal, kembalikan frontend ke
base API lama dan deploy ulang. Backend tim tidak perlu diubah. Jangan menghapus
deployment Vercel lama sebelum integrasi terbukti stabil.

## 9. Referensi

- Kontrak API lengkap: `backend/docs/API_ROUTES.md`
- Frontend yang sudah memakai `/api/v1`: folder `frontend/`
- Arsitektur aplikasi: `architecture.md`
- Kebijakan hosting dan proxy: `docs/HOSTING.md`
