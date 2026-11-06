# Upload CSV dan Excel — frontend

5 November 2026. Branch `feat/excel-upload-frontend` dimulai dari main `4d8c8c0` (PR #30). Perubahan panduan pada PR #29 dibawa dengan merge biasa agar integrasi ini tidak menghapus `/panduan/`. PR #29 kemudian masuk main (`fd44191`); branch Excel diselaraskan lagi dengan main sebelum dikirim.

## Perilaku yang diterapkan

- Dua slot menerima CSV dan XLSX. File kosong, XLS lama, PDF, gambar, dan file di atas 10 MiB ditolak sebelum unggah.
- File asli dikirim sebagai `FormData` dengan `file` dan `kind` ke `POST /api/v1/umkm/imports/files`. Browser menentukan boundary multipart. Cookie sesi serta `X-CSRF-Token` tetap dipakai.
- XLSX tidak dibaca melalui `FileReader`, `.text()`, `.arrayBuffer()`, atau `TextDecoder` pada frontend. Konversi tetap di backend.
- Progres memakai event unggah XMLHttpRequest yang nyata. Setelah unggah selesai, indikator beralih ke pemrosesan server; tidak ada persentase rekaan.
- `422 details.sheets` membuka dialog pilihan wajib. File yang sama dikirim ulang dengan `sheet`. Pembatalan tidak menyimpan data; ganti file membuang pilihan sheet lama. **Pilih sheet lain** tersedia sebelum commit.
- Pratinjau menampilkan nama/ukuran file, format, sheet, baris header, total, cuplikan, dan warnings. CSV kanonik tetap mengikuti batas server 1 MiB/500 baris.
- Penyimpanan memakai `response.csv`, penjualan lebih dulu, lalu penerimaan. Hasil yang sudah berhasil tidak dikirim ulang ketika slot lain gagal. Konflik 409 membuang pratinjau yang belum tersimpan dan meminta pemeriksaan ulang.
- Hasil menampilkan `imported`, `auto_matched`, `ai_matched`, `skipped`, dan copy lima `ai_status`. AI `failed` menjadi catatan tindak lanjut, bukan kegagalan impor.
- Antrean Verifikasi dan endpoint alokasi tetap sama. Bagian **Sudah beres otomatis** menampilkan alokasi aktif dari backend, alasan, serta tindakan pembatalan yang sudah ada. Label berdasarkan prefix alasan yang memang disimpan backend: `Cocok otomatis (skor ...)` dan `Disarankan AI (skor aturan ...)`. Skor internal disembunyikan dari copy UI; frontend tidak membuat keputusan atau menghitung keyakinan sendiri.
- Tidak ada pemanggilan Gemini atau key AI pada frontend. Versi aset `excel1`; fingerprint PWA tetap berubah otomatis saat build.

## Hasil pengujian

| Pemeriksaan | Hasil |
| --- | --- |
| Unit/build frontend | 31 lulus |
| Browser upload Excel | 18 skenario lulus: multipart asli, sheet, batal/ganti file, 375/768/1440 px, 400/413/422/409/503, 401/403, lima status AI, metadata rusak, batas file, dan double submit |
| Regresi browser aplikasi | Handoff 17, MVP 15, audit 13 lulus |
| Preview backend live | 12 kasus lulus: CSV lama, XLSX 20/21 baris, dua-sheet 422 dan pilihan Rekap, file terenkripsi/rusak/XLS/PDF, lebih dari 20 sheet, lebih dari 500 baris, dan lebih dari 10 MiB |
| Transport di bawah batas | Workbook valid 2.102.935 byte menghasilkan 200 |
| E2E frontend ke backend live | Upload dan commit XLSX 20 + 21 baris, Verifikasi, pilihan sheet nyata pada 375 px, dan commit CSV lama satu baris; total akhir 42 entri sintetis |
| Pembacaan XLSX di client | 0; test memasang guard yang akan menggagalkan `.text()`/`.arrayBuffer()` pada File XLSX |
| Error JavaScript pada E2E live | 0 |

Seluruh operasi tulis live memakai workspace QA baru dengan data sintetis. Script menolak menjalankan pembersihan pada email/workspace selain QA tersebut. Workspace demo bersama dan data pengguna lain tidak diubah. Credential/session pengujian hanya ada di `.cache/` yang diabaikan Git.

Status AI yang benar-benar diamati pada data live adalah `no_candidates`; lima cabang status, termasuk `ok` dan `failed`, diuji lewat fixture. Laporan ini tidak mengklaim Gemini live menghasilkan pasangan pada dataset QA tersebut.

## Cara E2E live dijalankan

Backend live sudah menerima endpoint baru. Namun CORS saat diuji hanya mengizinkan origin produksi EFA, bukan hostname preview Vercel. Untuk menguji branch tanpa mengganti produksi, Playwright menyajikan **hanya aset statis branch** pada origin `https://efa.madgo.my.id`; request API tetap dikirim langsung ke backend live dengan cookie QA dan CSRF yang asli. Endpoint tidak di-mock pada pengujian ini.

Preview Vercel perlu ditambahkan ke `allowed_origins` backend agar login/upload lintas origin dari preview tersebut berjalan. Jangan melonggarkan CORS ke semua origin dengan credential. Produksi tetap menggunakan origin yang sudah diizinkan. Alamat preview final tercantum pada PR dan `docs/ui-guide/excel-upload/KIRIM_KE_TIM.txt`.

## Screenshot dan bukti

Folder: [`ui-guide/excel-upload/`](ui-guide/excel-upload/).

- `screenshots/01-pratinjau-xlsx-desktop.png`: dua workbook, 20/21 baris.
- `screenshots/02-hasil-impor-desktop.png`: hasil commit dan status AI.
- `screenshots/03-verifikasi-desktop.png`: hasil Verifikasi dari API live.
- `screenshots/05-pilih-sheet-mobile.png`: dialog 422 pada 375 px.
- `screenshots/06-sheet-terpilih-mobile.png`: hasil sheet Rekap.
- `screenshots/07-verifikasi-mobile.png`: Verifikasi pada HP.
- `provenance.json`: hasil request nyata, batas pengujian, dan tidak adanya perubahan data bersama.
- `fixtures/`: workbook sintetis yang dipakai; file oversize/padding hanya dibuat di cache.

Screenshot memuat data QA sintetis, bukan data pelanggan. Panduan pengguna serta pipeline screenshot di-update mengikuti multipart. Video panduan lama tetap menjelaskan jalur CSV yang masih valid; artikel menambahkan langkah XLSX/sheet.

## Berkas utama

`frontend/assets/import-flow.js` mengelola state slot dan sheet. `api-client.js` menangani multipart/progres serta mempertahankan error details. `import-status.js` menyimpan batas transport, copy status/error, dan label hasil otomatis. Tes browser berada di `frontend/tests/browser/excel-upload.cjs`; skenario backend live dan capture ada di `docs/ui-guide/excel-upload/live-e2e.cjs`.

Tidak ada perubahan source backend dibanding main. Perbaikan kecil build mengganti HTML output yang tersalin dengan atribut Hidden di Windows sebelum menambahkan tag analytics; source artikel tetap utuh.

## Deployment produksi — 6 November 2026

Atas instruksi Evan, deployment Excel yang sudah diuji dipromosikan ke `https://efa.madgo.my.id`. Vercel menunjuk `dpl_Ai3vfFJXEyVkqK4yjGkbjgL3bHxo`, status Ready. PR #31 tetap terbuka; deployment tidak melakukan merge GitHub.

Pemeriksaan pascadeploy memakai aset produksi dan API asli, tanpa route override atau mock: login melalui form, pratinjau workbook penjualan 20 baris dan mutasi bank 21 baris, Verifikasi, serta 422/pemilihan sheet pada 375 px lulus. Karena file 20/21 baris pernah diimpor pada QA sebelumnya, backend mengenalinya sebagai duplikat dan melewatinya; hasil ini bukan impor baru. Sheet Rekap menghasilkan satu baris pratinjau. Tidak ada commit impor pada pemeriksaan pascadeploy; data QA tetap 42 entri. Tidak ada error JavaScript dan tidak ada pembacaan XLSX client.

Bukti serta screenshot produksi berada di [`ui-guide/excel-upload/production/`](ui-guide/excel-upload/production/). Batas CORS hostname preview yang disebut di atas tidak berlaku untuk domain produksi yang sudah diizinkan.
