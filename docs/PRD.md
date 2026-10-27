# Product Requirements Document — Temu Kas

## Tujuan

Pemilik usaha perlu menelusuri setiap penjualan sampai pembayaran diterima. Selisih bisa berasal dari pembayaran bertahap, tanggal pencairan, biaya layanan, atau satu pencairan untuk beberapa pesanan. Selisih tidak otomatis berarti kehilangan uang.

Temu Kas menempatkan penjualan dan penerimaan dalam satu ruang kerja, mengusulkan pasangan yang memenuhi aturan, lalu meminta konfirmasi pemilik.

## Pengguna dan alur

Pengguna aktif adalah pemilik UMKM. Registrasi membuat akun owner dan ruang kerja sendiri. Tidak ada pemilihan peran demo, mentor, atau teknisi.

1. Daftar dengan nama, email, kata sandi, dan nama usaha.
2. Unduh template CSV, isi penjualan/penerimaan, lalu periksa pratinjau.
3. Simpan dan lihat ringkasan seluruh transaksi aktif.
4. Periksa usulan atau buat alokasi manual.
5. Catat pembayaran sebagian, biaya layanan, dan penerimaan gabungan.
6. Tinjau riwayat, koreksi keputusan, dan ekspor laporan.

## Kebutuhan produk

| ID | Kebutuhan | Penerimaan |
| --- | --- | --- |
| P-01 | Isolasi usaha | ID dari usaha lain tidak membuka atau mengubah datanya |
| P-02 | Impor dapat diperiksa | Preview tidak menyimpan; satu kesalahan menolak seluruh file |
| P-03 | Hindari data ganda | File sama tidak menambah data; ID dengan isi berbeda ditolak |
| P-04 | Usulan dapat dijelaskan | Referensi, kanal, nominal, tanggal dan keunikan kedua sisi diperiksa |
| P-05 | Keputusan di tangan pemilik | Usulan tidak otomatis menjadi alokasi |
| P-06 | Pembayaran realistis | Alokasi sebagian/gabungan dan biaya tidak melewati sisa saldo |
| P-07 | Koreksi terlacak | Undo dan pembatalan tidak menghapus audit |
| P-08 | Total konsisten | Ringkasan memakai seluruh data aktif, bukan halaman tabel |
| P-09 | Laporan berguna | CSV mempertahankan status dan aman terhadap formula spreadsheet |
| P-10 | Koneksi gagal ditangani | Error terlihat, isian bertahan, submit ganda diblokir |
| P-11 | Bisa dipakai di HP | Navigasi mobile, tabel scroll, dialog dan reduced motion |
| P-12 | Dapat dilanjutkan tim | UI memakai API versi tim dan mendukung hosting satu domain HTTPS |

## Halaman

Masuk/daftar, Ringkasan, Transaksi, Cocokkan kas, Impor CSV, Riwayat, Laporan, dan Pengaturan. Layout, logo, warna, foto, font, dan animasi tetap memakai tampilan prototipe Temu Kas milik Evan.

## Batas saat ini

- Input CSV manual; belum terhubung langsung ke bank, QRIS, atau marketplace.
- Bukan saldo bank langsung, akuntansi lengkap, atau perhitungan laba.
- Tidak memakai AI untuk memutuskan pasangan.
- Belum ada reset password email, MFA, undangan staf, atau hapus akun mandiri.
- Penyegaran memakai SSE workspace milik tim, dengan pemeriksaan state saat reconnect dan polling fallback ketika tab terlihat.
- Akun contoh sintetis; belum ada klaim validasi lapangan atau penghematan terukur.
- Penggunaan finansial nyata memerlukan persetujuan pemilik data, kebijakan privasi, backup/restore dan penilaian keamanan.

## Keberhasilan yang perlu diukur

Bandingkan waktu menyelesaikan pencocokan, jumlah koreksi pasangan, dan selisih tanpa penjelasan sebelum/sesudah memakai aplikasi pada dataset yang sama. Tetapkan target setelah baseline diukur, jangan mengarang angka manfaat.

## Referensi

[FRD](FRD.md), [integrasi](INTEGRATION.md), dan [hasil tes](VERIFICATION.md) menjelaskan implementasi aktif. [PRD awal](reference/PRD-prototype.md) dan [FRD awal](reference/FRD-prototype.md) disimpan utuh. Bagian Grader/Facility dalam arsip bukan fitur aktif repositori ini.
