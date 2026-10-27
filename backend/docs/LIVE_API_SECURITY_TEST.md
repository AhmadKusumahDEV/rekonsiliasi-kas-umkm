# Live API Security Test

| Field | Nilai |
|---|---|
| Target | `https://temukas.madgo.my.id` |
| Tanggal | 27 Oktober 2026 |
| Metode | HTTPS publik, tanpa SSH |
| Scope | Semua endpoint `/api/v1`, health, root, auth/CSRF, IDOR, CORS, cookie, error handling, export |
| Data uji | Dua workspace uji terisolasi, hanya data yang dibuat sendiri |

## Ringkasan

64 pemeriksaan dijalankan: 61 lulus, 3 tidak sesuai ekspektasi. Tidak ada
kerentanan critical yang membocorkan data lintas workspace atau melewati CSRF.
Temuan utama adalah cookie session belum memiliki atribut `Secure` saat aplikasi
berjalan di belakang nginx/Cloudflare, belum ada rate limit login, dan route
`/api/v1/*` yang tidak terdaftar mengembalikan 401 alih-alih 404.

## Endpoint yang diuji

| Endpoint | Hasil | Catatan |
|---|---|---|
| `GET /` | 200 | JSON identitas Nginx |
| `GET /health` | 200 | OK |
| `GET /health/live` | 200 | API hidup |
| `GET /health/ready` | 200 | Database siap |
| `GET /api/v1/config` | 200 | Brand dan role benar |
| `POST /api/v1/auth/register` | 201 | Akun/workspace uji dibuat |
| `POST /api/v1/auth/login` | 200 | Login valid |
| `GET /api/v1/auth/me` | 200 | Cookie session valid |
| `PUT /api/v1/auth/password` | 200 | Password berubah |
| `POST /api/v1/auth/logout` | 200 | Session dihapus |
| `GET /api/v1/umkm/state` | 200 | State workspace sendiri |
| `GET /api/v1/umkm/suggestions` | 200 | Satu pasangan unik |
| `POST /api/v1/umkm/imports/preview` | 200 | CSV valid |
| `POST /api/v1/umkm/imports` | 201 | Sale dan receipt tersimpan |
| `POST /api/v1/umkm/imports/:id/cancel` | 200 | Batch dibatalkan |
| `POST /api/v1/umkm/allocations` | 201 | Alokasi tersimpan |
| `POST /api/v1/umkm/allocations/:id/undo` | 200 | Alokasi dibalik |
| `GET /api/v1/umkm/export` | 200 | CSV dengan BOM dan formula dinetralkan |

## Temuan

### F-01 — Cookie session tidak memiliki `Secure` — Medium

- **Bukti live:** `Set-Cookie: rku_session=...; Path=/; Max-Age=43200; HttpOnly; SameSite=Strict` tidak memuat `Secure`.
- **Bukti source:** `internal/echohttp/auth_handler.go` memakai `Secure: c.Request().TLS != nil`.
- **Dampak:** Di deployment saat ini TLS berhenti di Cloudflare/nginx dan request ke API adalah HTTP. Browser tidak diberi instruksi untuk membatasi cookie hanya pada HTTPS. Dampaknya lebih rendah selama origin API tidak diekspos langsung, tetapi ini melemahkan defense-in-depth dan bisa menjadi masalah jika ada proxy/port lain yang terbuka.
- **Catatan live:** HTTP publik mengembalikan 301 ke HTTPS dan edge mengirim HSTS, sehingga exploitabilitas praktis lebih rendah; atribut `Secure` tetap harus diperbaiki.
- **Rekomendasi:** set `Secure: true` untuk production, atau percayai `X-Forwarded-Proto` dari nginx secara eksplisit dan gunakan `https` sebagai kondisi. Jangan hanya mengandalkan `r.TLS` di belakang reverse proxy.

### F-02 — Belum ada rate limit login/register — Medium

- **Bukti live:** 5 login gagal berturut-turut menghasilkan `401, 401, 401, 401, 401`; tidak ada `429`.
- **Bukti source:** handler login/register tidak memiliki rate limiter; dokumentasi PRD/FRD juga mencatat rate limit sebagai gap.
- **Dampak:** credential stuffing dan brute force dapat dilakukan tanpa hambatan dari aplikasi. Cloudflare dapat menambahkan lapisan tambahan, tetapi tidak boleh diandalkan sebagai satu-satunya kontrol.
- **Rekomendasi:** rate limit per IP + email dengan jendela waktu, backoff, dan respons 429; untuk multi-instance gunakan Redis. Tambahkan monitoring lonjakan 401 pada endpoint login.

### F-03 — Route API tidak terdaftar mengembalikan 401 — Low

- **Bukti live:** `GET /api/v1/does-not-exist` tanpa session mengembalikan 401, bukan 404.
- **Penyebab:** middleware session dipasang pada group `/api/v1` dengan prefix kosong; request unknown di group tersebut melewati middleware sebelum router menghasilkan 404.
- **Dampak:** bukan bypass autentikasi dan tidak membocorkan data, tetapi menyulitkan observability, dokumentasi, dan membingungkan klien.
- **Rekomendasi:** daftarkan route privat pada sub-group eksplisit, atau tangani 404/405 secara konsisten sebelum middleware sesi untuk route yang tidak ada.

### F-04 — Header keamanan duplikat/berbeda — Low

- **Bukti live:** respons API memuat dua `X-Frame-Options`, dua `X-Content-Type-Options`, dua `Permissions-Policy`, dan dua `Referrer-Policy` (`no-referrer` dari API dan `strict-origin-when-cross-origin` dari nginx).
- **Dampak:** tidak menimbulkan bypass langsung, tetapi browser/proxy dapat mengambil nilai yang tidak diharapkan dan audit header menjadi sulit.
- **Rekomendasi:** pilih satu layer sebagai pemilik header keamanan; biasanya nginx untuk header transport/edge dan API untuk CSP API yang spesifik. Hindari duplikasi.

### F-05 — Informasi build versi `dev` — Info

- `/health/live` menampilkan `version: "dev"`. Tidak ada secret, tetapi build production sebaiknya diisi lewat `-ldflags` dengan commit SHA.

## Pemeriksaan yang lulus

- Endpoint privat tanpa cookie mengembalikan 401.
- POST tanpa `X-CSRF-Token` dan dengan token salah mengembalikan 403.
- Session fixation dengan cookie acak ditolak 401.
- Cookie memakai `HttpOnly` dan `SameSite=Strict`.
- Password berubah; session lama dan password lama tidak berlaku.
- Logout menghapus session; `/auth/me` setelah logout 401.
- Login email tidak dikenal dan password salah memakai pesan yang sama.
- Registrasi dengan role `admin` ditolak 400.
- IDOR lintas workspace ditolak 404; workspace B tidak melihat entry A.
- CSV export menetralkan formula spreadsheet (`'=1+1`) dan memakai BOM/attachment.
- CORS menolak origin asing dan mengizinkan origin sendiri.
- Error JSON tidak membocorkan `pgx`, SQL, stack trace, atau hash password.
- Input email berisi SQL injection ditolak validasi 400.
- HSTS dan header keamanan edge tersedia dari nginx/Cloudflare.

## Batasan

- Tidak dilakukan serangan DoS, load test, atau brute force agresif.
- Hanya 5 percobaan login gagal; Cloudflare WAF tidak diuji secara menyeluruh.
- Tidak ada akses SSH atau pembacaan database pada pengujian ini.
- Dua akun/workspace uji dengan email berprefix `security-audit-...@example.test` dibuat dan belum dihapus. Tidak ada data pengguna lain yang dibaca atau diubah.
- Temuan source yang terkonfirmasi sudah dimasukkan langsung ke temuan di atas; audit subagent terpisah tidak disertakan karena tidak selesai.
