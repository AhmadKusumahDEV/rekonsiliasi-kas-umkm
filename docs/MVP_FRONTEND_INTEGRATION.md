# Handoff Backend → Frontend — MVP Rekonsiliasi Kas UMKM

Tanggal: 1 November 2026

Branch backend: `feat/mvp-gap-backend`

API live saat dokumen ini dibuat: `https://temukas.madgo.my.id`

Frontend live saat dokumen ini dibuat: `https://efa.madgo.my.id/app`

Dokumen ini fokus pada ide #01 **Rekonsiliasi Kas UMKM**. Perubahan backend
belum menjadi kontrak live sampai branch di-merge, image dibangun ulang, dan
migration dijalankan. Jangan menganggap endpoint baru sudah ada di server
sebelum langkah deploy itu selesai.

## 1. Status pekerjaan backend

Status persetujuan: dokumen ini adalah draft handoff untuk review Project
Owner. Tidak ada bukti PO approval pada repo; jangan menganggap pilot data
nyata sudah disetujui.

Yang siap dipakai dari branch backend:

| Kemampuan | Shape API | Alasan |
| --- | --- | --- |
| Batas impor dari server | `limits` pada `GET /api/v1/config` | Frontend tidak perlu menyalin angka 500/1 MiB/10.000 |
| Antrean selisih | `GET /api/v1/umkm/exceptions` | Menampilkan transaksi yang belum selesai atau belum punya keputusan |
| Jejak keputusan terhubung entitas | field baru pada `events` di `GET /api/v1/umkm/state` | Riwayat dapat membuka import/allocation yang tepat |
| Riwayat audit berpaginasi | `GET /api/v1/umkm/audit` | Halaman Riwayat tidak terpotong pada 120 event |
| Alasan pasangan | `reason` pada suggestions (sudah ada sebelumnya) | Pemilik melihat alasan, bukan sekadar label “cocok” |
| Penanda kandidat | `candidate_ids`, `candidate_count`, `suggestion_available` | Frontend tahu kandidat mana yang bisa dipilih dan kapan tombol konfirmasi aman ditampilkan |

Yang sudah ada sebelumnya dan tidak perlu dibuat ulang: register/login/sesi,
CSRF, workspace isolation, preview dan commit CSV, suggestions deterministik,
alokasi termasuk fee dan pembayaran sebagian, undo, cancel import, export CSV,
audit, serta SSE workspace version.

Yang sengaja **belum** dibuat: purge retensi otomatis, endpoint hapus workspace,
backup/restore otomatis, data-policy API, attestation data nyata, billing,
AI/OCR, dan integrasi bank. Semua itu membutuhkan keputusan produk atau
prosedur operasional; jangan menampilkan tombol yang mengaku sudah tersedia.
Implementasi saat ini hanya owner; PDF menyebut kemungkinan staf pencatatan
sebagai pengguna, sehingga undangan/peran staf tetap di luar scope iterasi ini.

## 2. Bukti dari dokumen

Acuan dokumen:

- `PBL_Week1_Tiga_Ide_penyesuaian.pdf` (PBL Week 1), ide #01.
- `Solution_Brief_Mini_Proposal_-_Rekonsiliasi_Kas_UMKM_-_Evan_William.pdf`
  (Solution Brief & Mini Proposal).

Kutipan dan konsekuensi yang dipakai untuk scope ini:

| Bukti | Isi yang relevan | Status di backend |
| --- | --- | --- |
| PBL halaman 7 | Preview, idempotency, persetujuan pemilik, alokasi tidak melebihi sisa, undo menjaga riwayat, isolasi akun | Sudah ada; endpoint exceptions membantu UI menampilkan antrean |
| PBL halaman 8 | Baris invalid ditolak sebelum tersimpan; referensi ganda diperiksa; impor ulang memakai penanda unik; ekspor menetralkan formula | Sudah ada; 500 baris berasal dari PDF, sedangkan 1 MiB dan 10.000 entri adalah guardrail implementasi yang kini diekspos server |
| PBL halaman 8 | “Data nyata hanya dengan izin”, “retensi uji coba diusulkan 30 hari dan perlu persetujuan”, backup dan prosedur hapus diuji sebelum data nyata | Belum diimplementasikan; tetap menjadi keputusan produk/operasional |
| PBL halaman 9 | Harga Rp49.000/usaha/bulan adalah hipotesis; 20 pelanggan dan titik impas bukan hasil uji pasar | Bukan fitur MVP dan tidak boleh ditampilkan sebagai tagihan nyata |
| Solution Brief halaman 3 | Alur upload CSV, preview, matching, keputusan manusia; kandidat otomatis hanya pasangan unik satu-ke-satu di kedua sisi | Sudah ada; `suggestion_available` sekarang menandai pasangan yang aman |
| Solution Brief halaman 3 | Batas 500 baris per file; tanpa bank, payment gateway, AI, OCR | Sudah ada; angka limit diekspos lewat config |
| Solution Brief halaman 5 | Alasan pasangan, antrean selisih, jejak keputusan, retensi/backup/hapus, TC01–TC10 | Tiga kemampuan pertama tersedia; sisanya ditunda |
| Solution Brief halaman 6 | Hipotesis revenue dan rencana adopsi | Bukan kontrak backend MVP |

Bukti kondisi frontend saat dokumen ini dibuat, dari repository:

- `frontend/assets/app.js` masih menulis batas `500 baris`, `1 MB`, dan
  `1048576` secara hardcode di sekitar baris 323, 458, dan 946. Setelah backend
  dideploy, frontend harus membaca `config.limits`.
- Tidak ada pemanggilan `/api/v1/umkm/exceptions` di `frontend/assets/app.js`.
  Karena itu halaman antrean selisih belum tersedia meskipun endpoint-nya siap.
- Kartu suggestions di `frontend/assets/app.js` sekitar baris 285 belum
  menampilkan `reason`; informasi alasan masih hanya tersedia di response API.
- Riwayat di sekitar baris 337 hanya menampilkan teks audit. Field
  `entity_type`/`entity_id` belum dipakai untuk navigasi ke import atau
  allocation.
- Pengaturan menampilkan peringatan data sintetis, tetapi belum ada alur
  persetujuan atau pengelolaan retensi data nyata.

## 3. Kontrak API yang berubah atau perlu dipakai

Auth tetap sama: cookie `rku_session`, `credentials: "include"` untuk
frontend/API berbeda origin, dan `X-CSRF-Token` dari login/register atau
`GET /api/v1/auth/me` untuk setiap POST/PUT.

### 3.1 `GET /api/v1/config`

Response publik sekarang memiliki `limits`:

```json
{
  "kind": "umkm",
  "brand": "Easy Financial Audit",
  "tagline": "Setiap penerimaan, jelas pasangannya.",
  "roles": ["owner"],
  "limits": {
    "max_import_rows": 500,
    "max_csv_bytes": 1048576,
    "max_entries": 10000
  }
}
```

`max_import_rows` adalah jumlah baris data setelah header. `max_csv_bytes`
adalah ukuran byte CSV UTF-8. `max_entries` adalah batas seluruh entri dalam
satu workspace. Server tetap menjadi otoritas; validasi client hanya untuk
memberi umpan balik cepat.

### 3.2 `GET /api/v1/umkm/exceptions`

Endpoint protected, read-only, dan memakai cookie. Tidak memerlukan CSRF karena
GET. Response berupa array langsung:

```json
[
  {
    "entry": {
      "id": "0123456789abcdef0123456789abcdef",
      "kind": "sale",
      "date": "2026-10-01",
      "reference": "INV-1",
      "name": "Ayu",
      "amount": 150000,
      "channel": "transfer"
    },
    "status": "partial",
    "remaining": 50000,
    "candidate_count": 0,
    "suggestion_available": false,
    "reason": "Pembayaran sebagian; sisa tagihan Rp50000."
  }
]
```

Arti field:

- `status: "unmatched"`: belum punya keputusan atau belum punya kandidat.
- `status: "partial"`: sudah ada alokasi aktif, tetapi masih ada sisa saldo.
- `status: "ambiguous"`: ada kandidat yang belum unik di kedua sisi.
- `remaining`: sisa tagihan untuk penjualan, atau sisa penerimaan yang belum
  teralokasi. Alokasi yang sudah di-undo atau dibatalkan tidak dihitung.
- `candidate_count`: jumlah kandidat satu sisi dalam jendela tanggal yang sama
  (0–3 hari), tanpa menganggapnya otomatis cocok.
- `suggestion_available: true`: pasangan lolos aturan unik dua arah dan juga
  muncul di `GET /api/v1/umkm/suggestions`. Hanya nilai ini yang boleh
  memunculkan tombol “Konfirmasi usulan” tanpa pemilihan manual.
- `candidate_ids`: daftar ID kandidat lawan yang berada dalam jendela pencocokan.
  Gunakan ID ini untuk membuka pilihan kandidat; jangan mengarang ID dari
  `reason` atau `entity_id` audit.
- `reason`: teks siap tampil dalam Bahasa Indonesia; frontend tetap boleh
  menambahkan penjelasan visual, tetapi jangan mengubah maknanya.

Endpoint ini mengembalikan satu item per entry. Satu pasangan yang masih
menunggu konfirmasi dapat muncul sebagai dua item (sale dan receipt). Jika UI
ingin menampilkan satu kartu per pasangan, kelompokkan berdasarkan pasangan ID
yang ditemukan di suggestions.

Contoh kegagalan: tanpa cookie → `401`; sesi kedaluwarsa → `401`; database
tidak siap → `503`. Response sukses tanpa selisih adalah `[]`, bukan `404`.

### 3.3 Suggestions

`GET /api/v1/umkm/suggestions` tidak berubah bentuknya, tetapi frontend wajib
menampilkan `reason`:

```json
[
  {
    "sale": { "id": "...", "amount": 150000 },
    "receipt": { "id": "...", "amount": 150000 },
    "reason": "Referensi, kanal, dan nominal sama; hanya satu kandidat di kedua sisi (maksimal 3 hari)."
  }
]
```

Jangan menyebut pasangan “pasti cocok” untuk item yang hanya ada di antrean
`ambiguous`. Suggestions hanya berisi kandidat unik dua arah.

### 3.4 Audit event

`events` pada `GET /api/v1/umkm/state` sekarang berisi:

```json
{
  "id": 42,
  "actor": "Ayu",
  "action": "Konfirmasi pasangan",
  "detail": "Alokasi penerimaan ke penjualan",
  "entity_type": "allocation",
  "entity_id": "0123456789abcdef0123456789abcdef",
  "actor_user_id": "fedcba9876543210fedcba9876543210",
  "created_at": "2026-11-01T08:00:00Z"
}
```

`entity_type` saat ini adalah `import` atau `allocation`. Data historis dapat
bernilai `null` untuk `entity_type`, `entity_id`, dan `actor_user_id`; jangan
menebak ID dari teks `detail`. State hanya mengambil 120 audit terbaru, jadi
tampilkan sebagai riwayat ringkas, bukan arsip lengkap.

Untuk riwayat lengkap, gunakan `GET /api/v1/umkm/audit?before_id=<id>&limit=50`.
`limit` default 50 dan maksimal 100; `before_id` adalah cursor dari item audit
tertua di halaman sebelumnya. Array kosong berarti tidak ada halaman berikutnya.

### 3.5 Endpoint lama yang tetap dipakai

| Method | Path | Peran |
| --- | --- | --- |
| POST | `/api/v1/auth/register` | Membuat owner dan workspace |
| POST | `/api/v1/auth/login` | Login |
| GET | `/api/v1/auth/me` | Restore session dan CSRF |
| POST | `/api/v1/umkm/imports/preview` | Validasi CSV tanpa menulis |
| POST | `/api/v1/umkm/imports` | Simpan batch |
| POST | `/api/v1/umkm/imports/{id}/cancel` | Batalkan batch |
| POST | `/api/v1/umkm/allocations` | Simpan keputusan |
| POST | `/api/v1/umkm/allocations/{id}/undo` | Balikkan keputusan |
| GET | `/api/v1/umkm/export` | Unduh CSV |
| GET | `/api/v1/umkm/events` | SSE perubahan workspace |
| GET | `/api/v1/umkm/audit` | Halaman audit dengan cursor |

Detail request/response lengkap tetap ada di
[`backend/docs/API_ROUTES.md`](../backend/docs/API_ROUTES.md).

Error memakai envelope:

```json
{
  "code": 409,
  "status": "conflict",
  "message": "Pesan yang aman ditampilkan ke pengguna.",
  "request_id": "..."
}
```

Tampilkan `message`. Jangan menampilkan HTML, stack trace, atau detail SQL.

## 4. Cara integrasi di frontend

### 4.1 Bootstrap dan batas impor

Ambil config sebelum memakai nilai batas. Jangan mengandalkan konstanta lokal:

```js
const config = await api("/api/v1/config");
const limits = config.limits;

function validateFile(file) {
  if (file.size > limits.max_csv_bytes) {
    throw new Error(`CSV maksimal ${limits.max_csv_bytes} byte.`);
  }
}
```

Gunakan `limits.max_import_rows` untuk pratinjau dan pesan validasi.
`max_entries` dapat ditampilkan pada halaman impor atau pengaturan sebagai
batas ruang kerja. Validasi akhir tetap di server; jangan hanya mengandalkan
ukuran file di browser.

### 4.2 Memuat state, suggestions, dan exceptions

Setelah `/auth/me` sukses, muat ketiganya. Satu request gagal tidak boleh
menghapus state yang sudah berhasil dimuat:

```js
async function loadReconciliation() {
  const results = await Promise.allSettled([
    api("/api/v1/umkm/state"),
    api("/api/v1/umkm/suggestions"),
    api("/api/v1/umkm/exceptions"),
  ]);
  return {
    state: results[0].status === "fulfilled" ? results[0].value : null,
    suggestions: results[1].status === "fulfilled" ? results[1].value : null,
    exceptions: results[2].status === "fulfilled" ? results[2].value : null,
  };
}
```

Jika exceptions gagal, state dan suggestions yang sudah berhasil tetap boleh
ditampilkan dengan pesan bahwa antrean belum termuat. Jangan menampilkan angka
selisih nol saat request exceptions justru gagal.

`api` harus mengirim `credentials: "include"` dan menambahkan
`X-CSRF-Token` untuk request mutasi. Jangan membaca cookie `rku_session` dari
JavaScript karena cookie tersebut `HttpOnly`.

### 4.3 Membuat antrean selisih

Petakan entry ke suggestions dan tampilkan satu kartu per entry. Contoh aturan
aksi:

```js
function exceptionAction(item, suggestions) {
  if (item.status === "partial") return "Tambah alokasi";
  if (item.suggestion_available) return "Konfirmasi usulan";
  if (item.status === "ambiguous") return "Pilih kandidat";
  return "Alokasikan manual";
}

const suggestion = item.suggestion_available
  ? suggestions.find((candidate) =>
      candidate.sale.id === item.entry.id ||
      candidate.receipt.id === item.entry.id,
    )
  : null;

const choices = item.candidate_ids
  .map((id) => state.state.entries.find((entry) => entry.id === id))
  .filter(Boolean);
```

Jika `suggestion_available` benar tetapi pencarian suggestions tidak menemukan
pasangan, muat ulang state dan suggestions sebelum membuka modal. Ini bisa
terjadi karena ada perubahan dari tab lain. Jangan membuat ID alokasi baru dari
teks atau dari `entity_id` audit.

Tampilkan `reason` di kartu dan modal. Untuk `partial`, tampilkan `remaining`
sebagai sisa tagihan/sisa penerimaan dan jangan menandainya sebagai lunas.
Untuk `ambiguous`, tampilkan jumlah kandidat dan minta pemilik memilih; jangan
memilih kandidat pertama secara diam-diam.

### 4.4 Menyimpan dan menyegarkan data

Gunakan endpoint alokasi yang sudah ada:

```json
{
  "sale_id": "32-hex-lowercase",
  "receipt_id": "32-hex-lowercase",
  "amount": 50000,
  "fee": 0,
  "reason": "Pembayaran sebagian",
  "key": "32-hex-lowercase"
}
```

`key` harus idempoten dan 32 karakter hex lowercase. Setelah sukses, ambil
ulang state/suggestions/exceptions. SSE hanya memberi tahu bahwa versi
workspace berubah; REST state tetap sumber kebenaran. Tangani `401` dengan
login, `403` dengan mengambil CSRF baru, `409` dengan memuat ulang state, dan
`429` dengan pesan coba lagi nanti.

Retry dengan key dan isi yang sama aman: server mengembalikan allocation lama
tanpa menambah audit, versi, atau event SSE. Jangan mengganti key saat timeout;
itu dapat membuat alokasi kedua.

`reason` wajib diisi minimal 5 karakter bila ada `fee > 0` atau alokasi
meninggalkan sisa pada penjualan/penerimaan. Ini berlaku untuk pembayaran
sebagian, bukan hanya biaya layanan.

### 4.5 SSE, ekspor, dan audit

SSE tetap di `GET /api/v1/umkm/events` dengan cookie. Saat event `state`
masuk, bandingkan angkanya dengan `workspace.version`; jika berbeda, muat ulang
state. Jangan memakai urutan event sebagai sumber mutasi. Untuk ekspor, pakai
fetch/blob agar `401` tetap bisa ditangani; jangan hanya memasang `<a href>`
tanpa penanganan sesi.

Di riwayat, gunakan `entity_type`/`entity_id` untuk navigasi. Untuk data lama
yang field-nya `null`, tampilkan teks audit saja. `actor_user_id` berguna untuk
menyorot pelaku, tetapi jangan menampilkannya sebagai data sensitif mentah.
Untuk halaman berikutnya, panggil `GET /api/v1/umkm/audit?before_id=<id-tertua>&limit=50`;
state dashboard hanya berisi 120 audit terbaru.

## 5. Matriks penerimaan dan pengujian

| ID PDF | Skenario | Status backend | Tindak lanjut frontend |
| --- | --- | --- | --- |
| TC01 | Dua CSV valid, pratinjau | Ada | Tampilkan rows/count/total/skipped dari response |
| TC02 | Header/tanggal/nominal salah | Ada; 400 dengan baris | Tampilkan `message` dan pertahankan input |
| TC03 | Impor ulang file sama | Ada; duplicate/no-op | Jangan menampilkan sukses seolah ada baris baru |
| TC04 | Pasangan unik dua arah | Ada; suggestions + reason | Tampilkan alasan sebelum konfirmasi |
| TC05 | Dua transaksi bernilai sama | Ada; ambiguous | Jangan auto-pilih; minta keputusan |
| TC06 | Fee/parsial/gabungan | Ada | Tampilkan remaining dan validasi saldo |
| TC07 | Dua tab alokasi serentak | Ada; lock/version | Reload state saat 409 |
| TC08 | Batalkan pasangan | Ada; undo + audit | Konfirmasi dan reload |
| TC09 | Akses usaha lain/sesi habis | Ada; 401/404/403 | Kembali login atau tampilkan error aman |
| TC10 | Timeout/ekspor formula | Sebagian; export menetralkan formula | Tambahkan retry aman dan uji buka CSV |

Verifikasi backend lokal yang harus diulang setelah perubahan:

```powershell
cd backend
go build ./...
go vet ./...
go test -count=1 ./...
```

Integration test memakai database disposable, bukan database pengguna:

```powershell
docker compose up -d db-test
$env:TEST_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:55432/rku_test?sslmode=disable"
$env:TEST_REQUIRE_DB="1"
go test -race -count=1 ./...
```

Uji endpoint setelah backend baru live: config memiliki `limits`, exceptions
tanpa cookie menghasilkan `401`, exceptions dengan cookie menghasilkan array,
dan events memiliki field entity. Uji pasangan unik, kandidat ambigu, sisa
parsial, undo, dan ekspor. Hasil E2E frontend tetap diperlukan; backend tidak
membuktikan UI sudah mengikuti kontrak ini.

## 6. Fitur yang ditunda dan keputusan yang dibutuhkan

- **Retensi 30 hari**: angka pada PDF adalah usulan trial, bukan kebijakan
  final. Jangan membuat purge otomatis sebelum pemilik produk menetapkan
  durasi, data apa yang dihapus, dan audit penghapusan.
- **Hapus workspace**: PDF meminta prosedur diuji sebelum data nyata, bukan
  tombol self-service. Belum ada endpoint.
- **Backup/restore**: tanggung jawab operator hosting. Jangan menampilkan
  status backup palsu atau bucket/kredensial ke tenant.
- **Attestation data nyata**: peringatan UI belum cukup. Server-side consent
  bertanggal/nama/versi belum dibuat.
- **Billing/AI/OCR/bank**: bukan MVP. Harga Rp49.000 adalah hipotesis dan
  bukan harga yang sudah diuji.

Pemilik produk perlu memutuskan minimal: apakah data nyata diizinkan pada
pilot, berapa lama retensi, siapa yang boleh menghapus data, dan apakah
backup/restore diuji sebelum pilot. Tanpa keputusan itu, frontend hanya boleh
menampilkan alur data sintetis dan panduan izin.

## 7. Batas handoff

Dokumen ini menjelaskan kontrak backend yang siap diintegrasikan, bukan jaminan
bahwa frontend live sudah memakainya. Proposal PDF memakai Node/TypeScript;
backend ini memakai Go/Echo. Perbedaan stack ini harus dicatat di laporan PBL,
meskipun kontrak HTTP dan perilaku bisnisnya setara. Frontend tetap harus memperbarui
`frontend/assets/app.js`, menambah halaman antrean, mengganti hardcode limit,
dan menjalankan E2E di origin HTTPS yang terdaftar. Perubahan backend tidak
mengubah origin, nama cookie, atau format alokasi lama; migration baru hanya
menambah kolom audit, constraint pasangan, index, dan foreign key pelaku.
