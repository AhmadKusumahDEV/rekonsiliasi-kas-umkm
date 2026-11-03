# Analytics dan Search Console EFA

GA4 memakai measurement ID `G-8YV3VKNXHK` pada landing dan tiga halaman artikel.
Build menyisipkan initializer eksternal agar CSP tetap menolak inline script.
Tag hanya aktif pada `efa.madgo.my.id`, bukan localhost/preview atau aplikasi akun `/app`.
Tidak mengirim email, input formulir, isi transaksi, query URL, atau fragment URL.
Google Signals dan personalisasi iklan dimatikan. Pengaturan Enhanced Measurement di dashboard GA tetap perlu dikelola pemilik property.

File yang diberikan berisi `google-site-verification: google191985305be39b7b.html`.
Karena itu URL verifikasi yang dipakai adalah:

https://efa.madgo.my.id/google191985305be39b7b.html

Nama `google4ddef4208f435bda0.html` pada chat tidak sesuai lampiran dan tidak dibuat-buat.
Source verifikasi disimpan di root repo. Build mempublikasikan payload sebagai `.txt`,
lalu rewrite internal mempertahankan URL `.html` dengan HTTP 200 tanpa redirect cleanUrls.
Clean URLs global dimatikan karena Vercel tetap meredirect URL HTML sebelum rewrite. Alamat /app dan /artikel/<slug> tetap tanpa ekstensi melalui rewrite statis eksplisit.
Kode verifikasi publik ini memang harus dapat dibaca Google; bukan password/API secret.

## Yang dilakukan pemilik akun Google

1. Di Search Console, buka property URL-prefix https://efa.madgo.my.id/.
2. Pilih upload HTML dengan file yang sesuai lampiran, lalu tekan Verify setelah URL di atas tersedia.
3. Submit sitemap: https://efa.madgo.my.id/sitemap.xml.
4. Gunakan URL Inspection > Request indexing untuk landing dan artikel bila diperlukan.
5. Di GA4, pilih property dengan stream G-8YV3VKNXHK, buka Reports > Realtime.
6. Buka website dari browser tanpa pemblokir analytics; lihat kunjungan masuk. Laporan agregat dapat memerlukan waktu pemrosesan.

`robots.txt` dan sitemap hanya memuat/mengarahkan halaman publik. `/app` tidak dimasukkan sitemap.
Verifikasi kepemilikan dan pemasangan tag bukan jaminan waktu indexing, ranking, atau persetujuan Search Console.
Tidak mengklaim dashboard GA menerima data sebelum pemilik memeriksanya. Pemblokir iklan dan jaringan dapat menahan tag.

Rollback: revert commit integrasi Analytics/Search Console; jangan menghapus file verifikasi tanpa koordinasi pemilik property karena verifikasi bisa dicabut.
