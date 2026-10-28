# Landing page EFA

Halaman pengenalan produk disimpan terpisah dari aplikasi pencatatan kas.
Tidak ada salinan backend atau database di folder ini.

```text
landing/
├── index.html                 isi, tautan, struktur halaman
├── assets/
│   ├── landing.css            layout, warna, responsivitas, transisi
│   ├── editorial.css          bingkai foto, shapes, komposisi visual dan motion
│   ├── landing.js             carousel, menu HP, contoh pencocokan
│   └── images/                foto hero ukuran 1600 dan 4000 px
├── README.md                  panduan ini
└── SOURCES.md                 sumber foto dan analisis kebutuhan
```

Logo, favicon, font, dan foto dokumen dipakai bersama dari `frontend/assets/`.
Folder `docs/landing-preview/` berisi screenshot hasil pemeriksaan, bukan kode web.

## Alamat halaman

| Alamat | Isi |
| --- | --- |
| `/` | Landing page: hero, tentang, cara kerja, analisis, artikel, FAQ |
| `/app#login` | Masuk ke aplikasi |
| `/app#register` | Daftar pemilik dan usaha |
| `/app#overview` | Ringkasan setelah sesi terverifikasi |

Tautan lama seperti `/#login`, `/#register`, dan `/#reports` diarahkan ke
`/app` dengan hash yang sama. Anchor landing seperti `/#tentang` tetap
berada di halaman pengenalan. Setelah masuk atau keluar, aplikasi tetap
di `/app`; tidak menampilkan landing sebagai dashboard.

## Menjalankan lokal

Dari root repository:

```powershell
node frontend/dev.mjs
```

Buka http://127.0.0.1:3000 untuk landing, atau
http://127.0.0.1:3000/app#login untuk aplikasi. Landing tidak memerlukan API.
Login dan data aplikasi memerlukan backend Go lokal; ikuti README root.
Jika port dipakai: `$env:PORT='3058'; node frontend/dev.mjs`.

## Mengubah tampilan

- Ubah isi section dan CTA di `index.html`.
- Tiga judul dan deskripsi hero ada pada array `slides` di `landing.js`.
  Samakan slide pertama dengan HTML agar tetap terbaca tanpa JavaScript.
- Foto hero pertama dan kedua ada di `assets/images/`. Pertahankan versi
  1600 px untuk layar biasa dan 4000 px untuk layar beresolusi tinggi.
  Foto ketiga memakai `frontend/assets/efa-ledger-4k.jpg`.
- Palet ada di `:root` dalam `landing.css`. Logo dan font mengikuti EFA.
  Komposisi foto melengkung, ilustrasi lembar catatan, thumbnail artikel dan
  elemen dekoratif berada di `editorial.css`; file ini tidak mengubah UI `/app`.
- Contoh pencocokan berada pada objek `examples`. Nilainya ilustrasi,
  terpisah sepenuhnya dari transaksi pengguna dan tidak memanggil API.
- Artikel belum diterbitkan. Jangan menambahkan tautan atau tanggal seolah
  artikelnya sudah tersedia.

Carousel berganti setiap 6,5 detik, bisa dijeda, dipilih langsung, digeser
di HP, atau dikendalikan dengan panah/Home/End. Autoplay berhenti saat
hover, interaksi keyboard, tab tersembunyi, atau hero meninggalkan layar.
Pengaturan reduced motion memakai mode manual. Scroll reveal hanya
diaktifkan jika JavaScript dan IntersectionObserver tersedia; tanpa JS,
isi dan tautan daftar tetap terlihat.

## Build dan pemeriksaan

```powershell
node tools/build-static.mjs
node --test frontend/tests/*.test.mjs
```

Build menghasilkan `dist/index.html` untuk landing, `dist/app.html` untuk
aplikasi, serta aset publik. Vercel `cleanUrls` membuat `/app` membuka
`app.html`. Tidak ada fungsi API, konfigurasi privat, dokumentasi atau tes
di output web. Folder `dist` adalah hasil build dan tidak masuk Git.

Periksa carousel, menu HP, contoh nominal, CTA dan overflow pada lebar
320, 375, 768 dan 1440 px. Pada production, login tetap memakai API tim
dan aturan cookie/CORS yang dijelaskan di `docs/VERCEL_TEAM_API.md`.

Pilihan desain mengikuti panduan UI Design untuk marketing page,
header, gambar, warna, responsive layout, dan UI Animation untuk
transisi serta gesture. Tema tetap memakai identitas EFA yang sudah ada.
