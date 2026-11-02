# Integrasi frontend MVP

Pemeriksaan 2 November 2026, dari `main` revision `6785d08`.
Acuan implementasi: [handoff backend](MVP_FRONTEND_INTEGRATION.md).

## Kesesuaian dengan proposal

**Ya, kemampuan yang diintegrasikan termasuk scope ide rekonsiliasi kas UMKM
di proposal.** Ini hasil pencocokan dokumen, bukan persetujuan Project Owner
atau izin memakai data pelanggan nyata.

| Dokumen dan halaman | Scope | Implementasi frontend |
| --- | --- | --- |
| PBL Week 1 penyesuaian, 7 | Preview, idempotency, keputusan pemilik, batas alokasi, undo dan isolasi akun | Alur lama dipertahankan; error 409 memuat ulang data tanpa mengulang POST |
| PBL Week 1, 8 | CSV maksimal 500 baris, ekspor, referensi ganda ditinjau manusia | UI membaca batas dari config server dan menahan kandidat ambigu |
| Solution Brief Evan, 3 | Usulan hanya unik satu-ke-satu di kedua sisi; pembayaran sebagian/fee ditentukan pemilik | Reason tampil di kartu dan modal; tombol konfirmasi hanya dari usulan yang tersedia dan cocok dengan snapshot |
| Solution Brief Evan, 5 | Alasan pasangan, antrean selisih, jejak keputusan | Halaman Perlu diperiksa, remaining, kandidat terpotong, detail entitas audit dan riwayat berpaginasi |
| PBL Week 1, 8; Solution Brief, 5 | Izin data nyata, usulan retensi 30 hari, backup dan prosedur hapus diuji | Belum menjadi fitur. Tidak dibuat tombol, status backup, persetujuan atau purge palsu |
| PBL Week 1, 9; Solution Brief, 6 | Biaya dan harga merupakan hipotesis | Tidak diubah menjadi tagihan atau klaim pelanggan/penghematan nyata |

PDF yang dibaca: `PBL_Week1_Tiga_Ide_penyesuaian.pdf` (59 halaman, bagian ide
Evan pada halaman 7–9), dan
`Solution_Brief_Mini_Proposal_-_Rekonsiliasi_Kas_UMKM_-_Evan_William.pdf`
(8 halaman, terutama 3/5/6). Proposal menyebut React/Vite + Node/TypeScript.
Implementasi sekarang memakai HTML/CSS/JavaScript modular + Go/Echo dengan
PostgreSQL. Kesetaraan perilaku tidak berarti stack-nya sama. Dokumen PDF
tidak diedit dalam pekerjaan ini.

## Perubahan yang disambungkan

- `/api/v1/config`: `limits` menjadi sumber ukuran CSV, baris data, dan
  kapasitas ruang kerja. Config tanpa limits menonaktifkan impor, bukan
  memakai angka tebakan. Nilai live saat diperiksa: 500 baris, 1.048.576
  byte, 10.000 entri; angka ini tidak disalin sebagai fallback dalam kode UI.
- State, suggestions dan exceptions dimuat tanpa menjadikan keberhasilan
  satu endpoint bergantung pada keberhasilan endpoint lainnya. State tampil
  begitu tersedia. Antrean gagal dimuat menampilkan peringatan, bukan angka nol.
- `#exceptions`: satu kartu per entry, filter unmatched/partial/ambiguous,
  alasan, sisa tagihan/penerimaan, jumlah kandidat penuh, dan penjelasan
  saat daftar ID terpotong. Angka kartu bukan jumlah pasangan.
- Kandidat ambigu selalu membuka pilihan kosong yang wajib dipilih.
  Mode manual tersedia untuk kandidat di luar daftar 20 ID. Tidak ada ID
  hasil tebakan dari reason atau audit. Bila usulan unik hilang, UI memuat
  ulang data dan meminta pengguna memeriksa kembali.
- Suggestions menampilkan reason sebelum pemilik menyimpan keputusan.
- `#history`: mengambil audit dari endpoint cursor, 50 item per permintaan.
  Tombol Muat aktivitas lebih lama melampaui batas snapshot 120 terbaru.
  Pencarian tabel berlaku pada aktivitas yang sudah dimuat, bukan seluruh
  arsip server. Data lama dengan entity null tetap tampil tanpa tautan palsu.
- Detail audit membuka import/allocation berdasarkan type dan ID terstruktur,
  diverifikasi terhadap data workspace yang sudah dimuat. Detail ini baca-saja.
- Retry alokasi mempertahankan key dan payload form. 403 menyinkronkan CSRF
  tanpa memutar ulang POST; 409 menyegarkan data dan mempertahankan draft.
  Fee memerlukan alasan minimal lima karakter. 401 memakai pemulihan login
  yang sudah ada; 429/503 menampilkan pesan aman server.

## Bukti pemeriksaan

| Pemeriksaan | Hasil |
| --- | --- |
| 17 tes Node frontend | Lulus: termasuk limits dinamis, guard usulan dan target audit |
| 13 regresi browser audit sebelumnya | Lulus, fixture lokal |
| 11 skenario browser MVP baru | Lulus, fixture lokal |
| API HTTPS live tanpa cookie | Exceptions ditolak 401 |
| Config, state, suggestions, exceptions, audit dengan sesi demo | 200, shape sesuai; tidak mengubah transaksi |
| UI baru dengan API HTTPS nyata | 24 kombinasi halaman/ukuran, tanpa overflow atau pageerror; nol mutasi transaksi |

**Batas bukti UI HTTPS:** kode frontend lokal disajikan hanya di browser uji
terisolasi melalui intersepsi aset; API tetap server HTTPS sebenarnya.
Ini bukan deployment frontend produksi. Pengguna memberi izin login
baca-saja akun demo; pengujian tidak mengimpor, mengalokasikan, membatalkan,
atau mengganti kata sandi pada database produksi.

Skenario MVP lokal meliputi limit byte/baris dari config, config hilang,
usulan unik basi, CSRF 403, ambiguitas tanpa auto-pilih, remaining parsial,
exceptions 503, audit 130 event beserta detail, retry jaringan dengan key
sama, draft saat 409, dan responsivitas 375/768/1440 px. Data fixture sintetis.
Fixture 130 event tidak diklaim berasal dari akun demo live.

[Keluaran pengujian](mvp-preview/) memakai label sumber yang jelas. Screenshot
publik hanya memakai fixture sintetis lokal, bukan data akun HTTPS yang diuji.
Backend, migration 000004, generated sqlc, observability dan konfigurasi
hosting tim tidak diubah. Suite Go pada main tidak dijalankan ulang karena
pekerjaan ini hanya menyentuh frontend; hasil frontend tidak menggantikan
pengujian integritas transaksi backend.

## Menjalankan ulang

Dari root repository:

```powershell
node --test frontend/tests/*.test.mjs
$env:PORT='3058'; node frontend/dev.mjs
```

Di terminal lain, gunakan modul Playwright yang sudah terpasang pada mesin:

```powershell
$env:PLAYWRIGHT_MODULE='C:\lokasi\node_modules\playwright'
node frontend/tests/browser/mvp.cjs
node frontend/tests/browser/audit.cjs
```

Tes browser hanya menerima localhost/127.0.0.1, memalsukan seluruh request API
dengan fixture, dan menulis hasil ke `test-results`/bukti audit lokal.

## Status pengiriman

Hasil dikerjakan pada branch `feat/mvp-frontend-integration` setelah pull main.
Deployment produksi belum dilakukan: permintaan pekerjaan ini adalah
pull–integrasi–push, dan deployment memerlukan izin terpisah. Sebelum
deployment tim, review PR dan periksa kembali bahwa backend/migration live
masih sesuai kontrak. Status pilot data nyata tetap membutuhkan keputusan PO.
