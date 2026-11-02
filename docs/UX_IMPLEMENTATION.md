# Tindak lanjut temuan UX

Acuan: [rencana kerja UX](UX_FINDINGS_ACTION_PLAN.md), dari main `e85ee50`.
Perubahan ini berada di branch `feat/ux-action-plan`; belum di-deploy ke produksi.

## Perubahan frontend

| Temuan | Hasil |
| --- | --- |
| F-01 | Dua sisi dengan usulan unik yang sama menjadi satu kartu dan satu tombol. Grouping hanya berlaku bila kedua sisi sama-sama memenuhi kontrak usulan unik. Item partial/ambiguous tetap terpisah; bila usulan gagal dimuat, frontend tidak menebak pasangan. |
| F-02 | Kartu menampilkan referensi, nama, jenis, kanal, nominal, sisa, dan status kandidat dari snapshot entries. Kandidat terpotong tetap menampilkan jumlah penuh dan jalur manual; ID mentah tidak ditampilkan. Kandidat yang hilang diberi penjelasan. |
| F-03 | Kolom Tindakan membuka modal dengan sisi transaksi yang dipilih. Sisi lawan dan nominal tetap kosong sampai dipilih pengguna. Transaksi cocok/dibatalkan tidak memiliki tombol alokasi. Membuka/menutup modal tidak menyimpan data. |
| F-04 | Laporan memakai label “Penjualan belum teralokasi” dan penjelasan bahwa nilainya bukan kerugian, bisa belum dibayar, dibayar sebagian, atau menunggu pencairan. Penjelasan bahwa laporan bukan laba tetap ada. Angka dan rumus tidak berubah. |
| F-05 | Riwayat tanpa relasi terstruktur menjelaskan bahwa event lama belum menyimpan tautan ke impor/alokasi. Event baru tetap membuka entitas melalui ID terstruktur; tidak menebak ID dari teks. |
| F-09 | SSE error langsung mengubah status menjadi “Koneksi terbatas”. REST/polling yang berhasil tidak menyatakan SSE sudah pulih. Status bertahan saat pindah halaman; open SSE mengaktifkan status langsung dan memeriksa versi terbaru. Polling tetap aktif tanpa SSE. |
| F-10 | Main ditarik fast-forward sebelum mengubah frontend. Source sudah memuat exceptionsPage, loadAudit, limits, dan tautan entitas dari PR #20 yang telah digabung. Tidak mengambil salinan frontend lama. |

## Bagian yang tetap terpisah

- **F-06, reset sandi:** belum ada endpoint reset aman. Tidak membuat tautan dummy. Backend perlu token sekali pakai, expiry, rate limit, email, dan penanganan sesi.
- **F-07, retensi/hapus/restore:** menunggu kebijakan produk dan endpoint. Tidak membuat tombol yang seolah sudah berfungsi.
- **F-08, data demo:** pencarian source tidak menemukan `INV-SSE-1` pada seed Go/SQL. Ini catatan uji historis di workspace live menurut audit tim. Frontend tetap menampilkan transaksi dibatalkan; tidak menghapus atau menyembunyikan catatan berdasarkan nama tertentu. Untuk presentasi, siapkan workspace demo terpisah melalui pengelola data, bukan menimpa riwayat workspace bersama.

## Bukti dan cara uji

Seluruh pengujian perubahan menggunakan data sintetis lokal, tanpa menulis data produksi.

- 18 tes Node frontend lulus, termasuk helper grouping, validasi, CSRF, PWA, server dev, dan build.
- 15 skenario browser MVP mencakup grouping, kandidat, tindak lanjut kedua jenis transaksi, laporan, riwayat lama/baru, SSE error/poll/reconnect, pagination, retry, 403, dan 409.
- 13 skenario regresi browser audit lama lulus.
- Antrean dan riwayat diperiksa pada lebar 375, 768, dan 1440 px, tanpa overflow halaman atau JavaScript pageerror pada fixture.
- Screenshot yang disertakan adalah fixture sintetis, bukan data akun produksi.

Jalankan server statis dengan `PORT=3058` melalui `node frontend/dev.mjs`, lalu:

```text
node --test frontend/tests/*.test.mjs
node frontend/tests/browser/mvp.cjs
node frontend/tests/browser/audit.cjs
node tools/build-static.mjs
```

Dua tes browser memerlukan Playwright dan Chrome; bila modul tidak terpasang pada project, arahkan `PLAYWRIGHT_MODULE` ke instalasi Playwright lokal. Hasil JSON dan screenshot sementara masuk `test-results/` yang diabaikan Git. Polling gagal SSE sengaja ditunggu 15 detik dalam tes.

Batas pengujian: tidak memverifikasi email reset, kebijakan retensi, perubahan database demo, atau gangguan jaringan QUIC produksi. SSE diuji dengan event failure terkontrol; tidak mengklaim penyebab QUIC server telah diperbaiki. Endpoint dan backend tidak diubah.

Bukti sintetis: [antrean desktop](images/ux-followup/desktop-exceptions.png), [antrean mobile](images/ux-followup/mobile-exceptions.png), [hasil browser](images/ux-followup/browser-results.json).
