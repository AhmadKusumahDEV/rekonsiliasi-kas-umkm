# Audit Fitur Impor dan AI — 4 November 2026

Laporan ini merangkum audit backend, integrasi frontend, dan CI untuk
perubahan impor, auto-match, dan klien Gemini.

## Ringkasan

Inti backend sudah berjalan: impor CSV toleran header, skor pencocokan, dan
simpan otomatis. Namun belum siap disebut selesai karena unggah XLSX,
antrean job, layar Verifikasi, dan pemanggilan Gemini belum ada.

## Status pengujian

| Pemeriksaan | Hasil |
| --- | --- |
| `go build ./...` | Lulus |
| `go vet ./...` | Lulus |
| `gofmt -l` | Tidak ada file |
| `golangci-lint` v2.15.0 | 0 issue |
| `go test ./...` dengan PostgreSQL | Lulus |
| `go test -race ./internal/...` dengan PostgreSQL | Lulus |
| `go test` tanpa DB | Lulus, tetapi enam integration test di-skip |
| CI GitHub terakhir | Lulus pada commit `c0f7cb2`, sebelum perubahan ini |

Perubahan di worktree belum di-commit, jadi belum ada run CI untuk snapshot
ini. Pemeriksaan lokal setara CI sudah lulus.

## Temuan yang sudah diperbaiki

| Temuan | Tingkat | Perbaikan |
| --- | --- | --- |
| Kolom `id` diabaikan pada impor fleksibel | Tinggi | Alias `id` ditambahkan |
| File yang semua barisnya dilewati menghasilkan error, warnings hilang | Tinggi | Kembalikan pratinjau kosong beserta warnings |
| Bobot nama terlalu kecil sehingga nama berbeda tetap lolos | Tinggi | Bobot menjadi referensi 0,35; nominal 0,25; kanal 0,15; tanggal 0,15; nama 0,10 |
| Kandidat 0,60–0,94 diabaikan saat cek ambiguitas | Tinggi | Semua kandidat di atas 0,60 sekarang memblokir simpan otomatis |
| `1,250.00` terbaca 1 rupiah | Tinggi | Deteksi separator desimal berdasarkan posisi terakhir |
| Kurung tidak seimbang menjadi nominal positif | Tinggi | Ditolak dengan error |
| Referensi bank tidak diekstrak dari keterangan | Tinggi | Pola INV/TRX/REF diambil dari keterangan |
| Kanal bank kosong dianggap `lainnya` | Tinggi | Kanal disimpulkan dari keterangan (TRF/QRIS) |
| Auto-match bisa menyentuh transaksi historis | Tinggi | Hanya pasangan yang melibatkan batch impor terbaru |
| Count auto-match dilaporkan walau transaksi rollback | Sedang | Return 0 saat gagal |
| Threshold `NaN` lolos validasi | Sedang | NaN/Inf ditolak, kembali ke 0,95 |
| Pratinjau fleksibel tidak memperhitungkan data workspace | Sedang | Memakai `PreviewImportForWorkspace`, `skipped` disertakan |
| Gemini mengikuti redirect dan bisa membocorkan header key | Sedang | Redirect diblokir |
| Verdict ganda untuk pasangan yang sama | Rendah | Duplikat dibuang |

## Temuan yang masih terbuka

| Temuan | Tingkat | Dampak |
| --- | --- | --- |
| Header tabel bank hanya dibaca dari baris pertama | Tinggi | Mutasi dengan info rekening di atas tabel belum bisa diimpor |
| Auto-match aktif secara default | Tinggi | Perubahan perilaku endpoint lama; pertimbangkan default mati saat rilis |
| Gagal auto-match tidak dicatat | Sedang | Sulit didiagnosis; perlu log atau status job |
| Batas 40.000 kombinasi berhenti tanpa peringatan | Rendah | Pencocokan dilewati pada data besar |
| CSV kanonik belum disanitasi dari formula injection | Sedang | Aman selama tidak diunduh sebagai file |
| Integration test memakai satu database bersama | Sedang | Sempat flaky saat beberapa proses berjalan |
| Dokumentasi testkit menyebut env lama | Rendah | Membingungkan saat setup |
| Belum ada fixture XLSX/CSV nyata | Tinggi | Adapter bank belum teruji |

## Kesiapan backend

| Komponen | Status |
| --- | --- |
| Skor dan simpan otomatis | Selesai |
| Impor CSV toleran header | Selesai untuk header di baris pertama |
| Deteksi metadata di atas header | Belum |
| Unggah multipart XLSX | Belum |
| Antrean job dan status batch | Belum |
| Profil pemetaan tersimpan | Belum |
| Pemanggilan Gemini dari alur impor | Belum, klien sudah ada |
| Layar Verifikasi | Belum, frontend masih memakai halaman lama |

## Scope

Yang sudah masuk scope dan berjalan: CSV kanonik, CSV toleran header,
pencocokan deterministik, simpan otomatis, audit, pembatalan alokasi.

Yang sengaja ditunda: PDF, screenshot, OCR, multi-mata uang, XLSX, antrean
job, profil pemetaan, dan integrasi AI.

Scope ini masuk akal untuk demo karena tiga nilai utama sudah terlihat:
format lebih longgar, pasangan yakin tersimpan sendiri, antrean menyusut.

## Kenapa estimasi dua minggu

Angka dua minggu hanya realistis untuk potongan berikut: satu pengembang,
CSV dan XLSX, skor dan simpan otomatis, tanpa layar Verifikasi penuh dan tanpa
integrasi AI. Anggapannya: file uji sudah tersedia dan tidak ada perubahan
kontrak di tengah jalan.

Estimasi penuh 17–20 hari kerja berasal dari:

- 2 hari migrasi dan status batch,
- 2 hari CSV, 3–4 hari XLSX,
- 2–3 hari profil pemetaan dan pratinjau koreksi,
- 3 hari skor dan simpan otomatis,
- 2 hari Gemini,
- 3–4 hari frontend,
- plus pengujian dan perbaikan.

Yang membuatnya bisa melar: header bank yang tidak standar, file XLSX nyata,
kegagalan format, dan integrasi frontend yang menyentuh navigasi, PWA, dan
aksesibilitas.

## Kebutuhan akses testing

| Kebutuhan | Cara | Estimasi |
| --- | --- | --- |
| PostgreSQL | `docker compose up -d db-test`, port 55432 | 15–30 menit |
| Env test | `TEST_DATABASE_URL`, `TEST_REQUIRE_DB=1` | 5 menit |
| Database unik per run | Buat DB baru atau jalankan `-p 1` | 2–3 jam |
| Gemini | `GEMINI_API_KEY` di environment, bukan repo | 30–60 menit |
| Billing Gemini | Aktifkan di Google AI Studio | 1–2 hari kerja |
| Fixture bank/marketplace | File anonim, struktur asli | 1–2 hari |

Test Gemini di PR harus memakai mock. Panggilan live hanya lewat
`workflow_dispatch` dengan secret di environment terlindungi.

## Frontend

Rekomendasi audit frontend:

- Satu rute `#verifikasi`, dengan alias `#reconcile` dan `#exceptions`.
- Panggil `/imports/flexible` hanya bila header bukan kanonik.
- Simpan CSV kanonik hasil konversi, bukan CSV mentah, untuk commit.
- Tampilkan warnings dan `auto_matched` setelah impor.
- Filter antrean per item, bukan menggagalkan seluruh halaman.
- Estimasi 7–9 hari untuk satu frontend developer.

## Yang perlu dipikirkan ke depan

1. Kebijakan auto-apply: default mati atau nyala, dan siapa yang boleh
   membatalkan.
2. Privasi data keuangan yang dikirim ke Gemini, serta persetujuan pengguna.
3. Biaya dan kuota AI per workspace.
4. Format bank nyata: kumpulkan lima file anonim sebelum menulis adapter.
5. Strategi PDF: ditunda atau dijadwalkan.
6. Isolasi database test agar CI tidak flaky.
7. Fixture XLSX dan CSV di repo test.
8. Versi aset frontend dan cache PWA saat rute berubah.
9. Monitoring: tingkat auto-match, tingkat pembatalan, error impor.
10. Rencana rollback: feature flag untuk mematikan auto-match dan AI.

