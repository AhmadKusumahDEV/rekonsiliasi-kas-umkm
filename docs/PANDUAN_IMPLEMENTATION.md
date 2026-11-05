# Panduan EFA — status implementasi

5 November 2026. Branch `feat/panduan-efa`, dibuat dari main `2d08f0c` setelah PR #27 dan #28 digabung tim. PR ini menambahkan bantuan; tidak mengubah endpoint, adapter CSV, keputusan pencocokan, atau backend.

## Yang tersedia

| Permintaan | Hasil |
| --- | --- |
| Starlight terpisah di `/panduan/` | `docs-site/`; build induk menyalin hasil ke `dist/panduan/` |
| Halaman masuk berbasis tugas | Enam pintu masuk utama, navigasi maksimal dua tingkat, pencarian lokal |
| Tutorial aplikasi | Impor, Verifikasi, Ringkasan/catat harian, Transaksi, Laporan, Pengaturan, reset sandi, kosongkan data |
| Struktur tutorial | Tujuan, sebelum mulai, tujuh langkah, error umum, dan halaman terkait |
| Konten pendukung | Mulai cepat, format CSV, status, istilah, cara pencocokan, selisih, FAQ; total 16 halaman |
| Video di akhir tiga artikel | Impor 65 detik; Verifikasi 55 detik; catat harian 55 detik |
| Akses video | Pilihan Baca langkah/Tonton video, MP4 720p tanpa suara, caption VTT, teks langkah pada rekaman, transkrip, kecepatan 0,75–1,5× |
| Hemat kuota | `preload="none"`, poster lokal, tanpa autoplay; masing-masing MP4 kurang dari 1 MB |
| Screenshot | Delapan PNG dengan nomor langkah dari pipeline `docs/ui-guide/capture-panduan.cjs` |
| Bantuan aplikasi | Tautan kontekstual pada halaman aplikasi dan akses akun; tab baru menjaga pekerjaan yang sedang dibuka |
| Landing | Tautan Panduan lengkap di Cara kerja dan footer; empat langkah singkat tetap dipertahankan |
| PWA dan versi | Shortcut Panduan, fingerprint konten/media ke `version.json` dan build ID service worker |

Video dan screenshot seluruhnya memakai fixture API sintetis pada localhost. CSV yang digunakan disertakan di `docs-site/public/examples/`, sehingga contoh dapat diikuti ulang. Tidak ada transaksi akun live, token reset, atau credential dalam aset. Rekaman mentah dan dependency tidak masuk Git.

**Catat harian mengikuti fitur yang tersedia:** satu baris per tanggal/kanal melalui CSV. Tidak ada formulir catat manual baru di Ringkasan. Penerimaan bukan laba, dan sisa tagihan bukan otomatis uang hilang. Tutorial tidak mengikuti contoh riset lama yang menyebut sheet Excel, tujuh kolom wajib, atau menu Cocokkan kas/Perlu diperiksa yang terpisah.

## Hasil uji

| Pemeriksaan | Hasil |
| --- | --- |
| Unit frontend dan public build | 25/25 lulus |
| Regresi browser handoff aplikasi | 17/17 lulus; seluruh write memakai fixture lokal |
| Responsif panduan | 16 halaman × 375/768/1440 px = 48 tampilan; tidak ada overflow dokumen |
| Navigasi HP | Menu buka/tutup lulus |
| Pencarian | Sinonim upload/rekonsiliasi/unduh/sandi, tanpa hasil, gagal muat + coba lagi, Ctrl+K, Escape, dan pengembalian fokus lulus |
| Video | 65/55/55 detik, 1280×720, caption 7/6/6 cue, kecepatan 1,5× lulus |
| Lazy video | Tidak ada request MP4 sebelum pemutaran |
| Fallback video | Kegagalan MP4 menampilkan pesan; langkah dan transkrip tetap tersedia |
| CSP hasil build | 0 pelanggaran; 0 page error pada 48 tampilan |
| Batas perubahan | Diff backend kosong; tidak ada perubahan pada import-flow.js, verification.js, api-client.js, atau validation.js |

Hasil terperinci lokal: `test-results/panduan.json`, screenshot responsif `test-results/panduan-1440.png` dan `panduan-375.png`. Pipeline yang dapat dijalankan ulang ada di [README panduan](../docs-site/README.md). Manifest screenshot menyimpan viewport, waktu, hash gambar, serta hash source frontend.

## Keputusan CSP dan pencarian

Percobaan awal mengonfirmasi Pagefind gagal mengompilasi WASM dengan `script-src 'self'`. Menambah hash skrip tidak memberi izin WASM. Agar tidak menambah `wasm-unsafe-eval`, `unsafe-eval`, CDN, atau domain baru, rilis ini memakai fallback pencarian JSON lokal dengan sinonim pengguna. Indeks Pagefind tetap dibuild sebagai artefak alternatif, tetapi runtime Pagefind tidak dimuat.

Starlight tetap menjadi mesin dokumentasi. Skrip bootstrap inline dipindahkan tanpa mengubah isinya ke file `_csp/<hash>.js`. Style statis dipindahkan ke CSS lokal. Header khusus `/panduan/` mempertahankan:

```text
default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:;
font-src 'self'; media-src 'self'; connect-src 'self'; object-src 'none';
frame-ancestors 'none'; base-uri 'none'; form-action 'self'
```

CSP landing/aplikasi tidak dilonggarkan. Panduan tidak mengirim data ke API aplikasi atau Gemini. Analytics halaman pemasaran tetap terpisah dari panduan dan akun.

## Batas pengujian

Uji dilakukan melalui browser otomatis dan pemeriksaan visual screenshot/frame video, bukan sesi usability dengan lima pemilik UMKM. Bukti email reset nyata, penghapusan data produksi, dan layanan AI tidak diuji ulang karena tidak diperlukan untuk perubahan panduan. Catatan bug aplikasi pada `FRONTEND_LIVE_AUDIT.md` tidak dinyatakan selesai oleh PR dokumentasi ini.

Shortcut PWA membuka halaman web panduan. Cache versi ikut berubah, tetapi video dan seluruh situs panduan tidak dijanjikan tersedia offline. Teks tetap menjadi sumber utama saat video gagal atau tidak ditonton.
