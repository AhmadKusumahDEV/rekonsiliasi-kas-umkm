# API Live — Rekonsiliasi Kas UMKM

Base URL production:

```text
https://temukas.madgo.my.id
```

Semua endpoint domain memakai prefix `/api/v1`. Kontrak ini adalah route yang
sedang live per 27 Oktober 2026. Path `/api/...` dari prototipe tidak berlaku di
backend ini.

## Ringkasan route

| Method | Path | Auth | CSRF | Status sukses |
|---|---|---|---|---|
| GET | `/` | Publik | — | 200, metadata Nginx |
| GET | `/health` | Publik | — | 200, health Nginx |
| GET | `/health/live` | Publik | — | 200, liveness API |
| GET | `/health/ready` | Publik | — | 200/503, readiness API |
| GET | `/api/v1/config` | Publik | — | 200 |
| POST | `/api/v1/auth/register` | Publik | — | 201 |
| POST | `/api/v1/auth/login` | Publik | — | 200 |
| GET | `/api/v1/auth/me` | Cookie | — | 200 |
| POST | `/api/v1/auth/logout` | Cookie | Ya | 200 |
| PUT | `/api/v1/auth/password` | Cookie | Ya | 200 |
| GET | `/api/v1/umkm/state` | Cookie | — | 200 |
| GET | `/api/v1/umkm/suggestions` | Cookie | — | 200 |
| POST | `/api/v1/umkm/imports/preview` | Cookie | Ya | 200 |
| POST | `/api/v1/umkm/imports` | Cookie | Ya | 201 |
| POST | `/api/v1/umkm/imports/{id}/cancel` | Cookie | Ya | 200 |
| POST | `/api/v1/umkm/allocations` | Cookie | Ya | 201 |
| POST | `/api/v1/umkm/allocations/{id}/undo` | Cookie | Ya | 200 |
| GET | `/api/v1/umkm/export` | Cookie | — | 200 CSV |
| GET | `/api/v1/umkm/events` | Cookie | — | 200 SSE |

`Cookie` berarti browser harus mengirim cookie session. `CSRF` berarti request
non-GET harus membawa header `X-CSRF-Token` dari response login/register atau
`GET /api/v1/auth/me`.

## Alur auth

1. `GET /api/v1/config` untuk metadata aplikasi.
2. `GET /api/v1/auth/me` untuk mengecek session. Jika 401, tampilkan login.
3. `POST /api/v1/auth/register` atau `POST /api/v1/auth/login`; cookie session
   dipasang oleh response dan body berisi `user` serta `csrf`.
4. Simpan `csrf` di memory frontend. Jangan membaca cookie `rku_session` dari
   JavaScript karena cookie `HttpOnly`.
5. Kirim `X-CSRF-Token` pada setiap request non-GET setelah login.
6. `POST /api/v1/auth/logout` atau `PUT /api/v1/auth/password` akan
   membersihkan/menghapus session; frontend harus kembali ke login.

Cookie session bernama `rku_session`, dengan `HttpOnly`, `SameSite=Strict`,
dan `Secure` saat HTTPS. Ketiga atribut telah diverifikasi dari login browser
EFA ke API tim pada 28 Oktober 2026. Frontend pada subdomain HTTPS EFA memakai
credentials include; konfigurasi CORS/SSE dibatasi pada origin yang terdaftar.

## Contoh request/response

### Config

```http
GET /api/v1/config
```

```json
{
  "kind": "umkm",
  "brand": "Temu Kas",
  "tagline": "Setiap penerimaan, jelas pasangannya.",
  "roles": ["owner"]
}
```

Ini metadata publik untuk bootstrap frontend, bukan endpoint pengaturan server.

### Register

```http
POST /api/v1/auth/register
Content-Type: application/json
```

```json
{
  "name": "Nama Pemilik",
  "email": "owner@example.com",
  "password": "minimal-12-karakter",
  "workspace": "Nama Usaha",
  "role": "owner"
}
```

`role` boleh dihilangkan (default `owner`); untuk aplikasi ini hanya `owner`
yang diterima. Response `201`:

```json
{
  "user": {
    "id": "32-hex",
    "workspace_id": "32-hex",
    "app": "umkm",
    "name": "Nama Pemilik",
    "email": "owner@example.com",
    "role": "owner"
  },
  "csrf": "32-hex"
}
```

### Login

```http
POST /api/v1/auth/login
Content-Type: application/json
```

```json
{
  "email": "owner@example.com",
  "password": "minimal-12-karakter"
}
```

Response `200` berisi `user` dan `csrf`, sama seperti register.

### Me

```http
GET /api/v1/auth/me
Cookie: rku_session=...
```

Response `200` berisi `user` dan `csrf`. Gunakan `csrf` untuk request berikutnya.

### State

```http
GET /api/v1/umkm/state
Cookie: rku_session=...
```

```json
{
  "workspace": {
    "id": "32-hex",
    "name": "Nama Usaha",
    "code": "KODE123",
    "version": 4
  },
  "state": {
    "entries": [],
    "imports": [],
    "allocations": []
  },
  "users": [],
  "events": []
}
```

Entry memiliki field:

```json
{
  "id": "32-hex",
  "source_id": "S-1",
  "batch_id": "32-hex",
  "kind": "sale",
  "date": "2026-10-01",
  "reference": "INV-1",
  "name": "Ayu",
  "amount": 100000,
  "channel": "transfer",
  "note": "",
  "canceled": false,
  "created_at": "2026-10-01T00:00:00Z"
}
```

Import dan allocation memakai field yang sama dengan model prototipe:
`imports[].rows`, `imports[].checksum`, `allocations[].sale_id`,
`allocations[].receipt_id`, `allocations[].reversed`, dan seterusnya.

### Suggestions

```http
GET /api/v1/umkm/suggestions
Cookie: rku_session=...
```

Response `200` berupa array:

```json
[
  {
    "sale": { "id": "32-hex", "amount": 100000 },
    "receipt": { "id": "32-hex", "amount": 100000 },
    "reason": "Referensi, kanal, dan nominal sama; hanya satu kandidat..."
  }
]
```

### Import preview

```http
POST /api/v1/umkm/imports/preview
Content-Type: application/json
X-CSRF-Token: ...
Cookie: rku_session=...
```

```json
{
  "kind": "sale",
  "csv": "id,tanggal,referensi,nama,nominal,kanal,catatan\nS-1,2026-10-01,INV-1,Ayu,100000,transfer,"
}
```

Response `200`:

```json
{
  "rows": [],
  "count": 1,
  "total": 100000,
  "checksum": "sha256-hex"
}
```

### Commit import

```http
POST /api/v1/umkm/imports
Content-Type: application/json
X-CSRF-Token: ...
Cookie: rku_session=...
```

Body sama dengan preview. Response `201`:

```json
{
  "id": "32-hex",
  "imported": 1,
  "skipped": 0
}
```

Jika checksum sudah pernah diimpor, response tetap `201` dan berisi
`duplicate: true` serta `id` batch lama. Jika file pernah dibatalkan atau source
ID bertentangan, response `409`.

### Allocate

```http
POST /api/v1/umkm/allocations
Content-Type: application/json
X-CSRF-Token: ...
Cookie: rku_session=...
```

```json
{
  "sale_id": "32-hex",
  "receipt_id": "32-hex",
  "amount": 90000,
  "fee": 5000,
  "reason": "biaya layanan",
  "key": "32-hex"
}
```

`key` adalah idempotency key 32 karakter hex dan juga menjadi ID allocation.
Response `201` adalah objek allocation. Amount penjualan = amount penjualan -
(amount alokasi + fee); amount penerimaan = amount penerimaan - amount alokasi.

### Undo/cancel

```http
POST /api/v1/umkm/allocations/32-hex/undo
X-CSRF-Token: ...
Cookie: rku_session=...
```

```http
POST /api/v1/umkm/imports/32-hex/cancel
X-CSRF-Token: ...
Cookie: rku_session=...
```

Keduanya mengembalikan `200 {"ok":true}`. Cancel import ditolak `409` bila
masih ada alokasi aktif pada batch tersebut.

### Export

```http
GET /api/v1/umkm/export
Cookie: rku_session=...
```

Response `text/csv; charset=utf-8` dengan attachment `umkm-rekap.csv`. CSV berisi
BOM UTF-8; sel yang dimulai dengan `=`, `+`, `-`, atau `@` dinetralkan.

### SSE events

```http
GET /api/v1/umkm/events
Cookie: rku_session=...
Accept: text/event-stream
```

Response `text/event-stream`. Setelah koneksi dibuka, server mengirim event
bernama `state` setiap kali workspace berubah:

```text
event: state
data: 42
```

Angka pada `data` adalah `workspace.version`. Cocokkan dengan
`state.data.workspace.version`; jika berbeda, panggil ulang
`GET /api/v1/umkm/state`. EventSource mengirim cookie secara otomatis untuk
origin yang sama. Gunakan `withCredentials: true` hanya bila frontend dan API
berada di origin berbeda dan konfigurasi CORS/cookie mendukungnya.

Server mengirim heartbeat berkala. Untuk logout/ganti password, backend
memutus stream user; frontend harus kembali ke halaman login. Tidak ada replay
event; setelah reconnect, selalu fetch state REST sebagai sumber kebenaran.

## Format error

Error dari API memakai envelope:

```json
{
  "code": 400,
  "status": "bad request",
  "message": "Data permintaan tidak valid: ...",
  "request_id": "..."
}
```

Gunakan `message` untuk toast/form error. Frontend prototipe lama membaca
`error`; itu harus diubah.

Status yang perlu ditangani: `400` validasi, `401` session/login, `403` CSRF,
`404` data tidak ada di workspace, `409` konflik saldo/duplikat, `413` body
terlalu besar, `500` gangguan server, dan `503` database tidak siap.

## Yang belum tersedia

- Endpoint `photos`, `pdf/extract`, `assignments`, `submissions`, dan `tickets`
  milik Tilik/Ruang tidak termasuk backend UMKM ini.
- Belum ada OpenAPI/Swagger atau Postman collection. Dokumen ini adalah kontrak
  yang sedang live.

## Catatan integrasi frontend

- Gunakan origin yang sama untuk menghindari kombinasi CORS + cookie
  `SameSite=Strict`.
- Origin yang diizinkan saat ini adalah `https://temukas.madgo.my.id`;
  `http://localhost:3000` dan `http://localhost:5173` tersedia untuk development.
- Untuk request yang memakai cookie, gunakan `credentials: "same-origin"` pada
  fetch jika frontend berada di origin yang sama.
- Jangan membaca cookie session dari JavaScript; cookie `HttpOnly`.
- Simpan CSRF hanya di memory; ambil ulang dari `/api/v1/auth/me` setelah reload.
- Path ID wajib 32 karakter hex tanpa huruf besar.
- CSV maksimal 1 MB dan 500 baris; body JSON API dibatasi 5 MB.

## Nginx untuk SSE

Tambahkan location khusus sebelum `location /` pada `temukas.conf`:

```nginx
location = /api/v1/umkm/events {
    proxy_pass $temukas_api;
    proxy_http_version 1.1;
    proxy_set_header Connection '';
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto https;
}
```

`proxy_buffering off` wajib agar event langsung sampai ke browser. Timeout SSE
lebih panjang dari timeout API biasa. Konfigurasi Echo juga tidak boleh memakai
`WriteTimeout` global yang lebih pendek dari durasi stream.
