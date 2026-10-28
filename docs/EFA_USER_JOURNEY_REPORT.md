# Laporan Alur Pengguna EFA

Tanggal: 28 Oktober 2026.

- Aplikasi: `https://efa.madgo.my.id/app`
- Backend: `https://temukas.madgo.my.id`
- Akun: `pemilik@temukas.test` / `KasDemo!2026`
- Workspace: Roti Sore (Demo)

## Ringkasan

Alur inti pemilik usaha berjalan dari login sampai ekspor. Data yang tampil
berasal dari backend temukas, dan SSE memperbarui halaman tanpa reload.

Ada satu bug yang cukup mengganggu: setelah menyimpan pasangan, halaman masih
menampilkan data lama dan hanya memberi banner “Data baru · muat ulang”.
Pengguna bisa mengira penyimpanan gagal. Selain itu ada beberapa gap produk dan
kebersihan data yang perlu diputuskan.

## Metode

Dua subagent membaca source secara paralel:

1. Frontend: peta tujuh halaman, aksi, state, fitur tersembunyi, dan bug dari
   kode.
2. Backend: kemampuan owner per endpoint, aturan bisnis, batas, dan hal yang
   tidak tersedia.

Setelah itu alur dijalankan langsung di browser sebagai pengguna biasa.

## Persona dan tujuan

Pemilik UMKM yang mengurus pembukuan sendiri. Tujuannya menyatukan catatan
penjualan dan uang masuk, menemukan transaksi yang belum cocok, menjelaskan
selisih karena pembayaran sebagian atau biaya layanan, lalu menyimpan keputusan
yang bisa ditelusuri.

Aplikasi hanya memiliki satu peran, “Pemilik usaha”, dan satu workspace per
akun. Tidak ada jalur admin atau staf.

## Fitur yang dipakai pengguna

| Halaman | Aksi utama |
| --- | --- |
| Ringkasan | Melihat sisa tagihan, total penjualan/penerimaan, transaksi yang perlu dicocokkan |
| Transaksi | Mencari, memfilter, memilih 20/50/100 baris, melihat status batal |
| Cocokkan kas | Memeriksa usulan, alokasi manual, membatalkan pasangan |
| Impor CSV | Pratinjau, simpan, batalkan impor, unduh contoh CSV |
| Riwayat | Melihat audit aktivitas dan mencarinya |
| Laporan | Melihat ringkasan dan mengunduh rekap CSV |
| Pengaturan | Melihat akun/workspace, ganti kata sandi, logout |

Fitur yang mudah terlewat: pencarian dan pagination di tabel, alokasi manual,
undo pasangan, cancel impor, pratinjau CSV, ekspor, menu mobile, banner data
baru, fallback polling, tombol “Pasang EFA”, dan perlindungan formulir yang
belum disimpan.

## Hasil uji alur

| Langkah | Hasil |
| --- | --- |
| Login akun demo | Berhasil, masuk ke Ringkasan |
| Ringkasan | 5 penjualan Rp650.000, 4 penerimaan Rp590.000, 0/5 cocok |
| Cari “Kirana” di Transaksi | 3 dari 10 data |
| Filter “Dibatalkan” | 1 dari 10 data, yaitu `INV-SSE-1` |
| Ubah ukuran halaman ke 50 | Berhasil |
| Buka usulan INV-102 | Modal terisi penjualan, penerimaan, nominal 100.000, fee 0 |
| Simpan pasangan | Tersimpan, tetapi halaman masih menampilkan “Belum ada keputusan” |
| Klik banner “Data baru · muat ulang” | Keputusan muncul, status Aktif |
| Batalkan pasangan | Konfirmasi muncul, status menjadi Dibatalkan, sisa saldo pulih |
| Impor CSV contoh penjualan | Pratinjau: 5 baris valid, total Rp650.000 |
| Simpan file yang sama | “File sudah pernah diimpor. Tidak ada data ganda” |
| Riwayat | Menampilkan impor, konfirmasi pasangan, dan pembatalan |
| Ekspor CSV | Terunduh, 10 kolom, 10 baris data termasuk yang dibatalkan |
| Buka form ganti sandi | Field lama/baru tersedia, tanpa konfirmasi sandi baru |
| Menu mobile 390×844 | Tujuh rute tampil, termasuk logout |
| Logout lalu login ulang | Berhasil, kembali ke Ringkasan |

## Temuan

### Bug: halaman tetap basi setelah menyimpan pasangan

Reproduksi:

1. Buka `#reconcile`.
2. Klik **Periksa pasangan** pada usulan.
3. Simpan pasangan.
4. Modal tertutup, tetapi daftar keputusan masih kosong dan usulan masih ada.
5. Hanya banner **Data baru · muat ulang** yang muncul.
6. Setelah banner diklik, barulah keputusan tampil.

Penyebabnya ada di alur refresh: ketika modal terbuka, data diperbarui tetapi
render halaman dilewati, lalu banner disembunyikan. Menutup modal tidak memicu
render ulang. Dampaknya pengguna bisa mengira penyimpanan gagal dan mengulang
aksi. Perbaikannya: render ulang setelah modal tertutup, atau jangan lewati
render ketika banner ditampilkan.

### Bug: pengguna yang sudah login membuka `#login`

Jika sesi masih aktif lalu membuka `/app#login`, aplikasi menampilkan “Halaman
tidak tersedia”, bukan mengalihkan ke Ringkasan. Ini terjadi saat pengujian dan
membingungkan karena pengguna sudah login.

### Perilaku menyesatkan: impor duplikat tetap menulis audit

Impor file yang sama tidak menambah data dan UI memberi pesan yang benar, tetapi
Riwayat mencatat aktivitas “Impor transaksi / Data CSV tervalidasi” dan versi
workspace ikut naik. Untuk pengguna, ini terlihat seperti ada perubahan padahal
tidak ada. Sebaiknya percobaan duplikat tidak menulis audit atau ditandai
sebagai “ditolak, tidak ada perubahan”.

### Gap produk

- Tidak ada reset/lupa kata sandi. Hanya ganti sandi dengan sandi lama.
- Hanya satu peran owner; tidak ada staf, undangan, atau multi-workspace.
- Tidak ada input transaksi manual, edit, atau hapus per entri.
- Tidak ada pagination/pencarian/filter di server; seluruh state dikirim ke
  browser.
- Kuota 10.000 entri tetap menghitung entri yang dibatalkan, dan tidak ada jalur
  pengelola untuk menaikkan kuota.
- Usulan otomatis hilang untuk entri yang sudah pernah dialokasikan sebagian,
  meskipun masih punya sisa. Pengguna harus memakai alokasi manual.
- Audit hanya 120 kejadian terbaru dan tidak menyimpan ID entitas terkait.

### Gap validasi klien

- Teks UI menjanjikan maksimal 500 baris, tetapi klien hanya memeriksa 1 MB.
- Batas nominal pada modal alokasi tidak dihitung ulang saat pilihan penjualan
  atau penerimaan diganti, dan `max` hanya Rp1 triliun, bukan sisa yang berlaku.
  Pengguna bisa mengirim kombinasi yang pasti ditolak server.
- Form ganti sandi tidak meminta konfirmasi sandi baru.

### Catatan kebersihan

- Batch uji `SSE-1` (`c6afe4bc...`) sudah dibatalkan tetapi masih tampil di
  Transaksi, Riwayat, dan ekspor sebagai baris batal.
- Akun uji `qa-20261028-1@temukas.test` masih ada.
- URL lama `https://efa.madgo.my.id/#reports` menampilkan landing page, bukan
  aplikasi. Aplikasi ada di `/app#reports`.
- `/health/live` masih menampilkan `version: "dev"`.

## Rekomendasi prioritas

1. Perbaiki render setelah modal alokasi ditutup. Ini bug yang paling terasa.
2. Alihkan pengguna yang sudah login dari `#login`/`#register` ke Ringkasan.
3. Jangan menulis audit atau menaikkan versi untuk impor duplikat tanpa
   perubahan data.
4. Tegakkan batas 500 baris di klien dan hitung ulang batas nominal saat
   pilihan transaksi berubah.
5. Tambahkan konfirmasi sandi baru.
6. Putuskan produk: reset sandi, staf/multi-user, input manual, dan pagination
   server.
7. Bersihkan artefak uji sebelum demo, atau siapkan workspace demo baru.

## Kesimpulan

Alur utama pemilik usaha sudah berfungsi dan terhubung ke backend temukas.
Masalah terbesar bukan pada integrasi, melainkan pada satu bug render setelah
menyimpan pasangan dan beberapa gap validasi/produk yang perlu diputuskan
sebelum dipakai di luar demo.
