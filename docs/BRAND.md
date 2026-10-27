# Easy Financial Audit — identitas visual

Easy Financial Audit (EFA) adalah nama tampilan baru proyek yang sebelumnya
bernama Temu Kas. Folder repository, kind umkm, ID workspace, email akun,
database dan endpoint tetap memakai identitas teknis yang sama.

## Palet

| Warna | Hex | Pemakaian |
| --- | --- | --- |
| Biru | #5AA2E0 | Logo, penanda navigasi, detail identitas |
| Cyan | #54B0D9 | Detail logo dan aksen informasi |
| Amber | #F59E0B | Tombol utama dan penanda perlu perhatian |
| Coral | #EB7A7A | Penanda peringatan dan turunan error |
| Charcoal | #3D414A | Teks, panel saldo, menu HP |
| Putih | #FFFFFF | Permukaan utama |

Warna teks dan latar status memakai turunan terang/gelap palet agar dapat
dibaca. Tombol amber memakai tulisan charcoal; tulisan putih pada biru
terang tidak dipakai sebagai pasangan teks tombol.

## Sumber visual

Logo digambar ulang sebagai SVG berdasarkan referensi yang diberikan
pemilik proyek: kaca pembesar, grafik, panah naik dan kotak data.
SVG tetap tajam pada ukuran berbeda; PNG disediakan untuk aplikasi yang
memerlukan raster. Ini bukan klaim bahwa foto raster asli berubah menjadi
detail baru melalui pembesaran.

| Aset | Format/ukuran |
| --- | --- |
| efa-mark.svg | Simbol vektor, dipakai di header/login |
| efa-logo.svg | Logo lengkap dan wordmark vektor |
| efa-logo-hd.png | PNG transparan 1920 x 1400 |
| favicon.svg | Ikon browser vektor |
| favicon-32.png | Fallback browser 32 x 32 |
| favicon.ico | 16, 32, 48, 64 dalam satu file |
| apple-touch-icon.png | Ikon iOS 180 x 180 |
| icon-192.png, icon-512.png | Ikon raster 192 dan 512 |
| favicon.png | Simbol raster 512, untuk kompatibilitas |

Semua aset ada di frontend/assets. Referensi palet dan logo berasal dari
lampiran pengguna. Font dan foto usaha yang sudah ada dipertahankan beserta
atribusi/lisensinya.

## Tempat mengedit

frontend/assets/efa.css menjadi lapisan tema EFA setelah style.css,
brand.css, dan motion.css. Komposisi layout dan interaksi tidak diganti.
design.js menyusun logo header, index.html memuat favicon serta metadata.

Cache URL aset memakai versi efa2. Saat mengganti aset berikutnya,
perbarui versi agar cache browser memvalidasi file baru.

## Live dan source tim

- Source integrasi: frontend/assets pada repository aktif.
- Source deployment Vercel yang sedang tayang:
  ../archives/three-go-prototypes/internal/platform/web.
- Keduanya memakai aset dan efa.css yang sama untuk UMKM.
- Domain demo: https://efa.madgo.my.id/.

Perubahan ini tidak memigrasikan backend demo Vercel menjadi backend tim.

## Halaman masuk dan daftar

`frontend/assets/auth.css` memuat komposisi navy khusus `.auth-umkm`, setelah
`efa.css`. Warna navy #102B43 dan #153B58 merupakan turunan gelap biru EFA;
cyan #91D4ED dipakai untuk teks tautan agar terbaca di latar gelap. Tombol
utama tetap amber #F59E0B. Warna dashboard tidak ikut berubah.

Di HP, foto dekoratif dan alur tiga langkah disembunyikan agar form cepat
dijangkau. Di desktop, keduanya mendampingi form. Animasi memakai transform
dan opacity, dibatalkan saat pengguna memakai keyboard, serta mengikuti
pengaturan reduced motion. Tidak ada animasi dekoratif yang berulang.

Snapshot terbaru login/daftar: [auth-refresh](ui-guide/auth-refresh/).
