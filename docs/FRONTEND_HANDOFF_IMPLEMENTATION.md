# Integrasi frontend impor dan Verifikasi

Tanggal: 4 November 2026. Acuan implementasi: main `44d628a`, sembilan dokumen handoff yang dimulai dari `FRONTEND_TASKS.md`.

## Keputusan kontrak

`FRONTEND_TASKS.md` dan `FRONTEND_VERIFICATION_CONTRACT.md` menjadi acuan endpoint dan scope. Bagian deteksi otomatis, XLSX/multipart, dan marketplace campuran pada dokumen riset lama tidak diaktifkan. Jenis file ditentukan slot, bukan ditebak frontend. Frontend tidak menghubungi Gemini; angka `ai_matched` ditampilkan apa adanya dari backend dan bukan klaim bahwa AI termasuk demo MVP.

## Hasil implementasi

| Kebutuhan | Implementasi |
| --- | --- |
| Dua slot opsional | Satu halaman dengan Penjualan dan Mutasi bank. Minimal satu dipilih. CSV UTF-8; XLSX/PDF ditolak dengan petunjuk. |
| Flexible preview | Setiap file dikirim `{kind,csv}` ke `/api/v1/umkm/imports/flexible`. `preview` adalah array. Count, total, skipped dan seluruh warnings terlihat sebelum simpan. Warnings di-escape sebagai teks. |
| Commit kanonik | `/umkm/imports` selalu menerima `response.csv`; tidak memakai CSV mentah. Penjualan lebih dulu, baru penerimaan. |
| Hasil dan retry | `imported`, `auto_matched`, `ai_matched`, skipped ditampilkan terpisah. File sukses tidak diulang saat file kedua gagal. Request tidak diulang otomatis setelah network/403/timeout. Checksum backend tetap otoritas idempotensi. |
| Verifikasi | Satu route `#verifikasi`; `#reconcile` dan `#exceptions` menjadi alias. Pasangan unik dan kandidat tunggal timbal balik dikelompokkan satu kartu. Kandidat ambigu tetap dipilih eksplisit. |
| Fee hint | Label, sumber, confidence, alasan dan nominal saran tampil. Prefill hanya jika counterpart dari snapshot/antrean benar-benar tunggal dua arah dan saldo menjelaskan fee; pilihan berubah menghapus prefill biaya. Tidak ada alokasi sebelum pengguna menyimpan. |
| Remaining | Sisa pada kartu memakai `remaining` API; snapshot yang tidak konsisten tidak dianggap nol. Sisa ringkasan/transaksi dihitung dari ledger aktif dengan memperhatikan `settled` dan kategori; API tidak memberi field remaining pada semua state entries. Alokasi akhir tetap divalidasi server. |
| Tunai/non-operasional | Tunai settled tidak menjadi tagihan atau pilihan alokasi. Non-operasional tidak masuk antrean dan pilihan, terlihat pada bagian baca-saja terpisah. Ringkasan penerimaan operasional tidak memasukkan non-operasional. |
| Reset sandi | `#forgot` generik, `#reset?token=...` membaca hash dan segera membersihkan token dari URL; token hanya di memori. Konfirmasi password, error invalid/expired, dan cooldown 429 tersedia. Sukses kembali login. |
| Kosongkan data | Owner melihat danger zone. Password dan HAPUS wajib; tombol terkunci selama request. Error membuang konfirmasi sehingga harus diketik ulang. Sukses menutup dialog, reset draft impor, memuat state dan kembali Ringkasan. Akun/workspace/audit tetap sesuai kontrak server. |
| PWA | Shortcut Verifikasi, alias root lama termasuk forgot/reset, serta versi aset diperbarui. Worker masih memakai hash build dan tidak menyimpan API, token, transaksi atau ekspor. |

## Direktori yang berubah

- `frontend/assets/app.js`: integrasi route, render, auth, clear-data, SSE dan lifecycle.
- `frontend/assets/import-flow.js`: draft per slot, validasi CSV, preview dan commit berurutan. Draft dibersihkan saat keluar sesi.
- `frontend/assets/verification.js`: kategori yang dapat dialokasikan, grouping kartu, perlakuan fee hint dan alias route.
- `frontend/assets/validation.js`: penjagaan pilihan alokasi terhadap cash/settled/non-operasional.
- `frontend/assets/design.js`: shortcut ringkasan dan total operasional.
- Template lima kolom dan contoh mutasi berada di `frontend/assets/`; dua dataset backend bersih disalin ke `docs/panduan-demo/` untuk latihan.

## Bukti pengujian

- 25 tes Node frontend lulus.
- 17 skenario browser handoff lulus: canonical payload dan urutan, satu slot, debit-saja, warnings, file biner/UTF-8, retry parsial, double submit, grouping/fee, alias, kegagalan antrean, 403/409, forgot/429, reset sampai login memakai password baru, invalid/mismatch token/password, clear data dan sesi habis.
- 15 regresi browser MVP dan 13 regresi audit lama diperbarui ke kontrak baru dan lulus. Pengujian memakai fixture sintetis lokal.
- Impor, Verifikasi dan Pengaturan diperiksa pada 375, 768, 1440 px tanpa overflow halaman atau JavaScript pageerror pada fixture.
- `go test -short ./internal/service ./internal/echohttp` lulus; integrasi PostgreSQL sengaja dilewati mode short. Tidak ada perubahan kode backend pada branch ini.
- Pratinjau API HTTPS live diuji memakai akun demo dan CSV sintetis: penjualan, mutasi bank dan debit-saja semua 200; csv/preview/warnings sesuai kontrak. Tidak commit impor, mengirim reset email, mengubah password atau mengosongkan data pada workspace live.

Menjalankan pengujian:

```text
node --test frontend/tests/*.test.mjs
node frontend/dev.mjs
```

Di terminal lain, dengan Playwright dan Chrome tersedia:

```text
node frontend/tests/browser/handoff.cjs
node frontend/tests/browser/mvp.cjs
node frontend/tests/browser/audit.cjs
```

Gunakan PORT/HANDOFF_URL/MVP_URL/AUDIT_URL bila port bukan 3058. File hasil berada di `test-results/`, bukan artefak data produksi di Git.

## Batas verifikasi

Uji reset sampai login dan pengosongan data di browser memakai API fixture, bukan provider email atau database produksi. Backend telah memiliki test integrasi sendiri; test tersebut tidak dijalankan ulang pada database live. Uji pengiriman reset email end-to-end memerlukan akun uji yang dikuasai tim. Hindari tombol kosongkan pada workspace bersama untuk presentasi.

Analytics hanya mengukur halaman publik. Respons endpoint collection sebelumnya 204 dan file Search Console 200 membuktikan pemasangan teknis; status ownership/indexing dan metrik pada property GA tetap perlu dilihat pemilik akun Google. Tidak menyamakan file verifikasi terpasang dengan sudah terindeks di pencarian.

## Status rilis
PR frontend siap ditinjau. Deployment frontend baru tertunda karena CLI Vercel logout; perlu login kembali ke akun yang memiliki workspace hash-a09b. Perubahan tidak diklaim sudah live. Setup publik Google yang sudah live dicek ulang: verifikasi 200 tanpa redirect, tag 200, collection Analytics 204.
