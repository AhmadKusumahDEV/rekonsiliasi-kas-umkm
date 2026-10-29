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
lampiran pengguna. Foto usaha dan aset font terdahulu dipertahankan beserta
atribusi/lisensinya.

## Tipografi

UI memakai stack sistem `Helvetica Neue, Helvetica, Arial, sans-serif`,
didefinisikan di `frontend/assets/typography.css`. Landing, login/daftar,
dashboard dan kontrol menggunakan satu keluarga huruf yang netral.
Tidak ada unduhan font eksternal atau preload Poppins/Inter. Di Windows
umumnya Arial yang tersedia; di perangkat lain browser memilih huruf
pertama yang terpasang. Jarak judul dilonggarkan dari gaya display sebelumnya.

## Tempat mengedit

frontend/assets/efa.css menjadi lapisan tema EFA setelah style.css,
brand.css, dan motion.css. Komposisi layout dan interaksi tidak diganti.
design.js menyusun logo header, index.html memuat favicon serta metadata.

Cache URL aset memakai versi efa5. Saat mengganti aset berikutnya,
perbarui versi agar cache browser memvalidasi file baru.

## Live dan source tim

- Source integrasi: frontend/assets pada repository aktif.
- Source deployment Vercel yang sedang tayang: `frontend/` pada repo ini.
- Source prototipe di `../archives/three-go-prototypes` hanya arsip/rollback.
- Domain demo: https://efa.madgo.my.id/.

Handoff 28 Oktober menyambungkan frontend EFA langsung ke backend tim.
Perubahan branding sendiri tidak memindahkan akun/data lama; yang aktif
sekarang adalah akun dan database milik backend tim.

## Halaman masuk dan daftar

`frontend/assets/auth.css` memuat komposisi khusus `.auth-umkm`, setelah
`efa.css`. Form berada di kiri dan panel foto editorial berada di kanan,
dengan takikan melengkung dan kartu terang yang menumpuk pada bagian bawah.
Di HP, form tampil dahulu, diikuti panel visual. Urutan DOM sama dengan
urutan visual agar keyboard dan pembaca layar tidak berpindah secara janggal.

Latar memakai biru/cyan EFA dengan aksen coral dan amber; panel memakai
turunan gelap #204660, #344A57 dan #52636C. Input berbentuk pil dengan latar
#10212C, teks terang dan focus ring cyan. Tombol utama tetap amber EFA.
Warna dashboard tidak ikut berubah.

`auth-scene.js` menyusun panel visual yang dipakai bersama oleh login dan
daftar. Keduanya memakai foto 4K Leeloo The First yang berbeda (lihat
[sumber dan lisensi](../frontend/ASSETS.md)); bukan testimoni pengguna.
Tidak ditambahkan tombol login sosial atau pemulihan sandi yang belum
memiliki alur backend.

Foto dan kartu memakai animasi masuk transform/opacity. Animasi dibatalkan
saat pengguna memakai keyboard dan mengikuti reduced motion. Tidak ada
animasi dekoratif berulang atau tambahan library animasi.

Caption foto dan catatan PBL tidak ditampilkan pada halaman autentikasi.
Sumber foto tetap terdokumentasi di ASSETS.md. Foto pengganti diambil dari
atas, 4000 x 2670 piksel, sehingga dokumen dan kalkulator lebih merata tajam.
Dashboard memakai foto yang sama dengan login dan memanfaatkan cache browser.
Tidak ada tulisan kredit atau foto referensi pada UI EFA.

Navigasi, tombol, toast, modal, menu HP, dan pergantian halaman memakai
feedback konsisten. Bagian yang mulai di bawah viewport diberi reveal
singkat saat digulir menggunakan IntersectionObserver. Elemen tetap terlihat
tanpa observer, tidak ada state bisnis yang ditunda, dan observer lama
dilepas ketika halaman diganti atau pengguna memakai keyboard/reduced motion.

Snapshot terbaru login/daftar: [auth-refresh](ui-guide/auth-refresh/).
