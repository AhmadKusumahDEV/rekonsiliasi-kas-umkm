# Bukti upload Excel EFA

Screenshot pada `screenshots/` berasal dari frontend branch yang terhubung ke backend live. Semuanya memakai akun QA dan catatan sintetis. Lihat `provenance.json` untuk metode pengujian dan hasil request; tidak ada password, cookie, atau token di folder ini.

## Melihat hasil

1. Buka `02-hasil-impor-desktop.png` untuk jumlah baris, pasangan otomatis/AI, dan status AI.
2. Buka `03-verifikasi-desktop.png` untuk tindak lanjut.
3. Buka `05-pilih-sheet-mobile.png` dan `06-sheet-terpilih-mobile.png` untuk pemilihan sheet di HP.
4. File `KIRIM_KE_TIM.txt` berisi ringkasan yang dapat langsung dikirim bersama PR.

Folder `production/` berisi pemeriksaan langsung sesudah deployment 6 November 2026: aset produksi asli, API asli, dan tanpa mock. Lihat `smoke.json`; pemeriksaan ini tidak menyimpan impor baru.

## Data latihan

`fixtures/penjualan-20.xlsx`, `mutasi-bank-21.xlsx`, `dua-sheet.xlsx`, dan `csv-lama.csv` adalah data sintetis. Gunakan workspace latihan kosong. Penolakan diuji dengan workbook terenkripsi/rusak, workbook terlalu banyak sheet/baris, serta signature XLS/PDF. Fixture penolakan XLS/PDF bukan dokumen operasional.

Jalankan `python generate-fixtures.py` dengan `openpyxl`; tambahkan `msoffcrypto-tool` untuk membuat ulang workbook terenkripsi. Sandi file fixture terenkripsi adalah `EFA-test-only-2026`, bukan sandi akun aplikasi. File lebih dari 10 MiB dibuat di `.cache/` dan tidak di-commit.

## Menjalankan ulang

Tes tanpa backend: jalankan frontend lokal pada port 3058, lalu `node frontend/tests/browser/excel-upload.cjs` dari root. Tetapkan `PLAYWRIGHT_MODULE` bila memakai instalasi Playwright yang sudah ada.

`live-e2e.cjs` memerlukan credential QA privat di cache, menolak identitas selain QA, dan dapat mengosongkan **hanya workspace QA tersebut** sebelum mengulang. Jangan menyalin credential akun sungguhan ke file QA. Script tidak mengubah source/deployment backend atau frontend produksi.
