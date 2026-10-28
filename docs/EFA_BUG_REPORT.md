# Laporan Bug Frontend EFA

Tanggal: 28 Oktober 2026.

- Aplikasi: `https://efa.madgo.my.id/app`
- Backend: `https://temukas.madgo.my.id`
- Akun uji: `pemilik@temukas.test` / `KasDemo!2026`
- Source yang dianalisis: frontend terbaru dari `origin/main` (`d7f3c91`)

## Ringkasan

Tiga subagent memeriksa auth/routing, data/mutasi, dan UI/aksesibilitas. Dari
30 kandidat, 9 bug berhasil divalidasi di browser atau lewat HTTP. Tidak ada
bug yang menghalangi login, penyimpanan, atau SSE. Masalah paling terasa ada
pada state halaman yang basi setelah menyimpan pasangan, pratinjau impor yang
menghitung duplikat, dan beberapa masalah aksesibilitas.

Dua klaim subagent **tidak terbukti** dan tidak dimasukkan sebagai bug:

- **Login CSRF high**: endpoint menerima JSON lintas origin lewat curl, tetapi
  form lintas situs mengirim `application/x-www-form-urlencoded` dan ditolak
  `400`. JSON lintas origin memerlukan preflight CORS yang tidak diizinkan.
  Tetap disarankan memeriksa header `Origin` sebagai pertahanan tambahan.
- **PWA tidak bisa dipasang**: deployment efa justru sudah memiliki
  `manifest.webmanifest`, `pwa/register.js`, ikon, dan alur install. Yang
  berbeda adalah source di repo belum memuat referensi PWA tersebut.

## Bug tervalidasi

### 1. Halaman tetap basi setelah menyimpan pasangan — Medium

Reproduksi:

1. Buka `#reconcile` dan klik **Periksa pasangan**.
2. Simpan pasangan.
3. Modal tertutup, tetapi halaman masih menampilkan “Belum ada keputusan” dan
   usulan lama.
4. Hanya banner **Data baru · muat ulang** yang muncul.
5. Setelah banner diklik, keputusan baru tampil.

Expected: halaman langsung menampilkan keputusan yang baru disimpan.
Actual: data baru sudah masuk, tetapi render ditunda sampai banner diklik.
Dampak: pengguna mengira penyimpanan gagal dan dapat mengulang aksi.
Perbaikan: render ulang setelah modal tertutup, atau jangan lewati render saat
banner ditampilkan.

### 2. `#login` dan `#register` saat sesi masih aktif — Medium

Reproduksi: login, lalu buka `/app#login` atau tekan Back sampai hash tersebut.

Expected: dialihkan ke Ringkasan.
Actual: muncul “Halaman tidak tersedia”.
Dampak: deep link dan tombol Back berakhir di jalan buntu.

### 3. Pratinjau impor menghitung baris duplikat — Medium

Reproduksi: workspace sudah berisi `P101`. Unggah CSV berisi `P101` identik dan
baris baru `S2` Rp50.000, lalu klik **Periksa CSV**.

Expected: pratinjau menandai `P101` duplikat dan total hanya Rp50.000.
Actual: pratinjau menampilkan **2 baris valid / total Rp200.000**, termasuk
`P101`. Commit baru memfilter duplikat di server.
Dampak: total pratinjau menyesatkan dan commit bisa gagal setelah pengguna
mengira file sudah benar.

### 4. Impor duplikat tetap menulis audit dan menaikkan versi — Low–Medium

Reproduksi: impor file yang sama dua kali. UI menampilkan “File sudah pernah
diimpor. Tidak ada data ganda”.

Actual: Riwayat mencatat “Impor transaksi / Data CSV tervalidasi” dan versi
workspace ikut naik meskipun tidak ada baris baru.
Dampak: riwayat dan SSE terlihat seperti ada perubahan data.

### 5. Klik area luar dialog tidak menutup dialog — Low–Medium

Reproduksi: buka dialog Impor atau Alokasi, lalu klik area gelap di luar panel.

Expected: dialog tertutup, dengan konfirmasi jika ada isian.
Actual: dialog tetap terbuka; hanya tombol Tutup atau Escape yang bekerja.
Dampak: dialog terasa macet, terutama di layar sentuh.

### 6. Fokus awal dialog jatuh ke tombol “Tutup” — Low (aksesibilitas)

Reproduksi: buka dialog Impor atau Alokasi dan periksa elemen yang langsung
terfokus.

Expected: fokus ke judul atau field pertama.
Actual: fokus ke tombol Tutup. Pembaca layar mengumumkan “Tutup” sebelum tujuan
dialog.

### 7. Label “Menyimpan…” dipakai untuk aksi yang tidak menyimpan — Low

Reproduksi: tekan **Masuk** atau **Periksa CSV**.

Expected: “Memproses…” atau “Memeriksa…”.
Actual: semua submit menampilkan “Menyimpan…”. Dampak: pengguna mengira data
sudah tersimpan padahal baru diperiksa.

### 8. Batas kata sandi dihitung byte, UI menyebut karakter — Low

Reproduksi: daftar dengan kata sandi 25 karakter Hanzi (75 byte).

Expected: diterima karena UI menyatakan 12–72 karakter.
Actual: backend menolak `400` karena memakai `len()` byte, bukan jumlah rune.
Dampak: pesan validasi menyesatkan untuk kata sandi non-ASCII.

### 9. Ekspor saat sesi habis menampilkan JSON mentah — Low

Reproduksi: sesi kedaluwarsa, lalu klik **Unduh rekap CSV**.

Expected: kembali ke login atau tampilkan pesan error di aplikasi.
Actual: tautan melakukan navigasi top-level; API menjawab `401` JSON sehingga
browser menampilkan JSON mentah. Dampak: pengguna keluar dari SPA tanpa alur
pemulihan.

## Temuan kode yang belum divalidasi di browser

Bagian ini berasal dari analisis source dan perlu diuji dengan skenario khusus.

| Temuan | Severity | Bukti | Catatan |
| --- | --- | --- | --- |
| Alokasi jatuh ke pasangan pertama bila ID target hilang | High | `app.js:407-410` | `sales.find(...) ‖ sales[0]`; risiko salah pasangan saat data berubah di tab lain |
| Refresh membuang state baru jika `suggestions` gagal | High | `app.js:342-349` | `state.data` baru diterapkan setelah kedua request sukses |
| Snapshot `state` dan `suggestions` tidak satu versi | Medium–High | `app.js:342-346` | Mutasi di antara dua request dapat menghasilkan usulan untuk entry yang tidak ada |
| Event SSE hilang saat refresh sedang berjalan | Medium | `app.js:620-626` | Event diabaikan jika `state.refreshing` true |
| CSRF lama setelah login ulang di tab lain | Medium | `app.js:84-95` | Hanya 401 yang ditangani, 403 tidak memicu sinkronisasi CSRF |
| Login sukses tetapi state gagal tetap menampilkan form login | Medium | `app.js:784-796` | Sesi aktif, event stream tidak dimulai |
| Sesi kedaluwarsa menutup form dan membuang isian | Medium | `app.js:86-94` | Tidak ada peringatan atau pemulihan isian |
| Batas 500 baris hanya diperiksa server | Medium | `app.js:322`, `:819` | Klien hanya memeriksa 1 MB |
| Batas nominal alokasi tidak dihitung ulang saat pilihan berubah | Medium | `app.js:422-455` | `max` tetap Rp1 triliun, bukan sisa yang berlaku |
| Kuota 10.000 dihitung sebelum deduplikasi dan termasuk entri batal | Medium | `umkm_service.go:175-181` | Impor delta kecil bisa ditolak di ambang batas |
| CSV non-UTF-8 diam-diam rusak | Medium | `app.js:802` | `file.text()` mengganti byte tidak valid dengan `U+FFFD` |
| Impor semua-duplikat membuat batch kosong | Low–Medium | `umkm_service.go:203-224` | Batch `row_count=0`, audit dan versi tetap naik |
| Pagination/pencarian hanya di klien; audit dipotong 120 | Medium | `app.js:186-201`, `umkm_service.go:43-46` | Payload besar dan riwayat lama tidak terlihat |
| Kontras fokus keyboard di permukaan gelap ~1,51:1 | Medium | `efa.css:62,73,95` | Di bawah rekomendasi 3:1 |
| Tombol lihat kata sandi tanpa `aria-pressed` | Low | `app.js:581,882` | Status terlihat tidak diumumkan |
| Hasil pratinjau CSV tidak diumumkan ke pembaca layar | Low | `app.js:808` | Tidak ada `aria-live` atau pemindahan fokus |
| Toast error hilang permanen setelah 4,5 detik | Low | `app.js:97` | Tidak ada tombol tutup atau riwayat error |
| Area scroll tabel tidak dapat difokuskan | Low | `style.css:468` | Tanpa `tabindex`, role, atau label |
| Target sentuh 30 px pada lebar 900 px | Low | `style.css:97`, `brand.css:527` | Di bawah 44 px |
| Hash tujuan hilang setelah login ulang | Low | `app.js:792` | Selalu kembali ke Ringkasan |
| Pesan jaringan berbahasa Inggris | Low | `api-client.js:34-45` | `Failed to fetch` diteruskan apa adanya |
| Menu auth mobile berpotensi terpotong di layar pendek | Low | `style.css:828-838` | Perlu uji tinggi 320–375 px |

## Catatan deployment

1. **Source dan deployment tidak sepenuhnya sama.** Deployment `/app` memuat
   `manifest.webmanifest`, `pwa/register.js`, dan ikon PWA, sedangkan
   `frontend/index.html` di `origin/main` belum memuat referensi tersebut.
   Deploy ulang dari repo berisiko menghilangkan PWA dan tombol “Pasang EFA”.
2. **Aset statis menolak User-Agent non-browser.** `GET /assets/app.js` tanpa
   UA browser mengembalikan `404 NOT_FOUND` dari Vercel; dengan UA browser
   mengembalikan `200`. Ini tidak mengganggu pengguna, tetapi mengganggu
   monitoring, curl, dan crawler.
3. **Artefak uji**: akun `qa-20261028-1@temukas.test`, batch `INV-SSE-1`
   (`c6afe4bc...`) yang sudah dibatalkan, dan beberapa entri audit dari
   pengujian masih ada di workspace demo.

## Prioritas perbaikan

1. Perbaiki render setelah modal alokasi ditutup (bug paling terasa).
2. Tambahkan guard pada alokasi agar tidak memakai `sales[0]`/`receipts[0]`
   ketika ID yang diminta tidak ditemukan.
3. Terapkan `state.data` lebih dulu, lalu muat suggestions secara terpisah.
4. Perbaiki pratinjau duplikat dan jangan menulis audit/versi untuk impor tanpa
   perubahan data.
5. Alihkan `#login`/`#register` saat sudah login ke Ringkasan.
6. Perbaiki fokus dialog, kontras fokus, dan pengumuman hasil untuk pembaca
   layar.
7. Samakan batas kata sandi (rune vs byte) antara UI dan backend.
8. Tangani error ekspor dengan fetch + blob agar sesi habis tidak menampilkan
   JSON mentah.
9. Samakan source deployment dengan repo, terutama untuk PWA.
