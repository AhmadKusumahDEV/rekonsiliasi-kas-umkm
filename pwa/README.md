# PWA EFA

EFA bisa dipasang dari https://efa.madgo.my.id/ dan dibuka melalui ikon di
layar utama atau daftar aplikasi komputer. Aplikasi terpasang membuka
`/app`, dengan login serta data dari backend tim yang sama.

## Memasang

Tombol **Pasang EFA** tersedia di footer landing, halaman masuk/daftar, dan
Pengaturan akun. Jika browser mendukung prompt pemasangan, tombol membuka
prompt tersebut. Jika belum tersedia, tombol menampilkan petunjuk.

1. **Android, Chrome:** buka EFA, tekan Pasang EFA dan ikuti konfirmasi.
   Alternatifnya, buka menu ⋮ → Instal aplikasi/Tambahkan ke layar utama.
2. **iPhone, Safari:** buka EFA → Bagikan → Tambahkan ke Layar Utama.
   Pilih Buka sebagai App jika tersedia, lalu Tambah.
3. **Komputer, Chrome/Edge:** gunakan tombol Pasang EFA atau ikon pemasangan
   di bilah alamat. Ikuti konfirmasi browser.

Nama menu dapat berbeda menurut versi browser. Dalam mode incognito,
pemasangan biasanya tidak tersedia. Gunakan browser biasa. Safari pada
iPhone memakai menu Bagikan; tidak menampilkan prompt Chromium.

## Saat koneksi terputus

- Tab yang sudah terbuka menampilkan pemberitahuan koneksi. Tidak ada
  reload otomatis yang menghapus isian.
- Navigasi baru ke beranda atau aplikasi membuka halaman offline yang
  ringan setelah service worker berhasil aktif pada kunjungan sebelumnya.
- Sambungkan internet, lalu tekan **Coba lagi** untuk membuka halaman tadi.
- Login, data terbaru, impor dan perubahan transaksi memerlukan internet.
  Tidak ada antrean transaksi offline atau sinkronisasi mutasi otomatis.
- Data yang sudah tersimpan tetap berada di server. Isian yang belum
  dikirim tidak disimpan sebagai draf PWA; simpan pekerjaan sebelum
  menutup atau memuat ulang halaman.

## Struktur file

```text
pwa/
├── manifest.webmanifest       nama, start URL, scope, ikon, shortcuts
├── service-worker.js          source worker: cache publik dan offline fallback
├── register.js                registrasi, install, status koneksi/pembaruan
├── pwa.css                    gaya kontrol PWA dan halaman offline
├── offline.html               halaman ketika navigasi gagal terhubung
├── offline.js                 tombol coba lagi dan pesan koneksi kembali
├── icons/
│   ├── icon.svg               logo EFA pada latar navy
│   ├── maskable.svg           source dengan jarak aman untuk ikon launcher
│   ├── icon-192.png
│   ├── icon-512.png
│   ├── maskable-512.png
│   └── apple-touch-icon.png
└── README.md                  panduan ini; tidak dipublikasikan
```

Source PWA tetap di satu folder. Build menaruh worker di `dist/sw.js`
agar scope-nya mencakup beranda dan `/app`; ini hasil build, bukan source
worker kedua yang perlu diedit. File publik lainnya ada di `dist/pwa/`.
Dokumentasi dan tes tidak masuk output hosting.

Ikon diturunkan dari `frontend/assets/efa-mark.svg`. Versi maskable
memberi ruang aman di sekeliling simbol agar potongan lingkaran/squircle
launcher tidak memotong logo. PNG sudah disimpan dalam Git.

## Cache dan pembaruan

Worker hanya menangani GET aset publik pada origin EFA, dengan batas
60 entri. Halaman offline, CSS dan ikonnya disiapkan saat instalasi worker;
foto/font/script publik yang dikunjungi bisa memakai cache saat jaringan
gagal. Request normal mendahulukan jaringan.

Respons API, sesi, CSRF, data transaksi, SSE, ekspor CSV, request lintas
origin dan request selain GET tidak ditangani worker. HTML landing dan
aplikasi juga tidak disimpan sebagai cache halaman; navigasi yang gagal
memakai halaman offline publik. Tidak ada credential di manifest.

Build membuat ID cache dari isi semua aset publik dan source worker.
Perubahan aset menghasilkan worker baru. Worker baru menunggu tab versi
lama selesai; notifikasi meminta pengguna menyimpan pekerjaan, menutup
semua tab/jendela EFA, lalu membuka kembali. Tidak ada paksa reload atau
`skipWaiting()`. Cache EFA versi lama dibersihkan saat worker baru aktif;
cache produk lain tidak dihapus.

Untuk rollback UI, pertahankan jalur worker dan aset PWA pada deployment.
Menghapus link manifest atau kembali ke deployment sebelum PWA tidak
otomatis menghapus worker yang sudah terpasang di browser pengguna.
Jangan memakai penghapusan file `/sw.js` sebagai cara menonaktifkan PWA.

Cookie login tetap milik backend tim. Instalasi PWA tidak mengubah
otorisasi, CORS, database, atau aturan transaksi.

## Menjalankan dan memeriksa

Dari root repo:

```powershell
node frontend/dev.mjs
```

Buka http://127.0.0.1:3000. Loopback mendukung service worker untuk
pengembangan; untuk pemasangan di HP gunakan domain HTTPS live.
Alamat HTTP berupa IP LAN komputer bukan pengganti HTTPS.

```powershell
node tools/build-static.mjs
node --test frontend/tests/*.test.mjs
```

Di Chrome DevTools → Application:

1. Manifest: nama EFA, start URL `/app`, display standalone dan ikon tampil.
2. Service Workers: worker `/sw.js` aktif dan scope `/`.
3. Cache Storage: hanya aset publik EFA; tidak ada `/api/v1` atau HTML `/app`.
4. Network: pilih Offline, navigasikan ke `/app?uji-offline=1`, lalu lihat
   halaman offline. Mengganti hash saja tidak membuat navigasi jaringan.
5. Kembali Online, tekan Coba lagi; halaman aplikasi terbuka kembali.

Jangan melakukan transaksi nyata ketika menjalankan pemeriksaan jaringan.
Tes otomatis kebijakan cache ada di `frontend/tests/pwa.test.mjs`.

Ikon hanya perlu dirender ulang saat logo berubah. Tool opsional
`tools/build-pwa-icons.mjs` memakai Playwright dan Chrome yang sudah
terpasang; bukan bagian build/deploy rutin. Jika Playwright tersedia pada
environment Node: `node tools/build-pwa-icons.mjs`. Environment
`PLAYWRIGHT_MODULE` bisa menunjuk instalasi Playwright yang sudah ada.
Tidak perlu memasang dependency ini untuk menjalankan EFA.

## Rujukan

- [MDN: Making PWAs installable](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Making_PWAs_installable)
- [MDN: Trigger installation](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/How_to/Trigger_install_prompt)
- [web.dev: Service worker lifecycle](https://web.dev/articles/service-worker-lifecycle)

Pemeriksaan Chrome menguji manifest dan kelayakan pemasangan, halaman
offline, koneksi kembali serta cache di desktop dan viewport HP.
Pemasangan OS pada perangkat iPhone/Android fisik tetap mengikuti browser
perangkat; tidak diklaim sudah diuji pada perangkat fisik.
