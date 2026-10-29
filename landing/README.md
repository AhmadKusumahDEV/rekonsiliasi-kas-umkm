# Landing page EFA

Halaman pengenalan produk disimpan terpisah dari aplikasi pencatatan kas.
Tidak ada salinan backend atau database di folder ini.

```text
landing/
├── index.html                 isi, tautan, struktur halaman
├── assets/
│   ├── landing.css            layout, warna, responsivitas, transisi
│   ├── editorial.css          komposisi section di bawah hero
│   ├── hero.css               hero penuh, bentuk biru berlapis dan cutout orang
│   ├── landing.js             carousel, menu HP, contoh pencocokan
│   └── images/                cutout PNG hero dan foto pendukung
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
- Hero pertama memakai `efa-team-hero.png`, ilustrasi tiga orang fiktif
  dengan alpha transparan, bukan testimoni pelanggan. Bentuk dekoratif
  dibuat dengan CSS agar tajam di berbagai ukuran. Sumber/prompt ada di SOURCES.
- Slide kedua memakai `efa-payments-hero.png` (pemilik usaha memeriksa
  pembayaran), slide ketiga `efa-records-hero.png` (pemeriksaan catatan).
  Ketiga ilustrasi memakai alpha transparan dan aturan layout yang sama,
  tanpa bingkai foto. Foto stok tetap tersedia untuk section pendukung.
- Palet ada di `:root` dalam `landing.css`. Logo dan font mengikuti EFA.
  Stack font sistem dipakai bersama dari `frontend/assets/typography.css`.
  Hero penuh dan breakpoint-nya berada di `hero.css`; ilustrasi lembar
  catatan, thumbnail artikel dan section lain berada di `editorial.css`.
  Keduanya tidak mengubah UI `/app`.
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

Tinggi judul dan deskripsi diukur terhadap ketiga variasi teks pada lebar
layar saat ini. Dengan begitu CTA, gambar dan bagian bawah hero tidak
bergeser ketika slide diganti. Pengukuran diulang saat resize; tanpa
JavaScript, slide pertama tetap memakai layout HTML biasa. Tombol panah
memakai chevron sederhana dengan area sentuh 48 px.

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
