# Panduan EFA

Situs bantuan terpisah memakai Astro Starlight. Frontend aplikasi tetap HTML/CSS/JavaScript; API dan alur keuangan tidak dipindahkan ke Astro.

## Menjalankan

Dari root repository:

```powershell
npm ci --prefix docs-site --no-fund --no-audit
node tools/build-static.mjs
node tools/serve-guide-preview.mjs
```

Buka `http://127.0.0.1:3060/panduan/`. Server ini melayani hasil build dengan CSP deployment. Ia tidak meneruskan request API. Untuk menyunting sambil melihat perubahan: `npm --prefix docs-site run dev` lalu buka alamat `/panduan/` yang ditampilkan Astro. Hanya hasil **build root** yang dipakai untuk deploy.

## Tempat mengubah

- `src/content/docs/`: 16 artikel, satu sumber untuk isi panduan.
- `src/components/`: header, pencarian, footer, dan video.
- `src/styles/efa.css`: tampilan panduan.
- `public/images/`: screenshot bernomor, dihasilkan pipeline.
- `public/examples/`: CSV sintetis yang dipakai video.
- `public/video/`: MP4, poster, dan caption VTT.
- `video-manifest.json`: durasi, teks caption/transkrip, dan ukuran video.
- `../docs/ui-guide/capture-panduan.cjs`: ambil screenshot/rekaman dari fixture localhost.
- `../tools/encode-guide-videos.py`: konversi rekaman WebM menjadi MP4 dan VTT.
- `../tools/build-panduan.mjs`: build, indeks pencarian, penyesuaian CSP, dan versi konten.

Ubah `lastUpdated` di artikel saat langkah berubah. Build menghitung versi dari seluruh output panduan, termasuk gambar/video. Versi ini juga masuk fingerprint service worker induk. `dist/`, `.astro/`, `node_modules/`, dan rekaman mentah `.cache/` tidak masuk Git.

## CSP dan pencarian

Pagefind diuji dan memerlukan izin kompilasi WebAssembly di luar `script-src 'self'`. Rilis ini memakai fallback indeks JSON dan JavaScript lokal; pencarian tidak membutuhkan server, WASM, API aplikasi, Gemini, CDN, `unsafe-inline`, atau `unsafe-eval`. Pagefind tetap dibuild sebagai artefak alternatif; runtime-nya tidak dimuat. Keputusan ini dibatasi pada pencarian, bukan penggantian Starlight.

Bootstrap tema/sidebar Starlight dipublikasikan sebagai file dengan hash konten. Style statis juga diekstrak ke CSS lokal. CSP aplikasi tidak dilonggarkan. Selalu uji hasil `tools/build-static.mjs`, bukan hanya server dev Astro.

## Screenshot dan video

Jalankan server aplikasi lokal (`$env:PORT='3058'; node frontend/dev.mjs`). Sediakan Playwright dan FFmpeg di lingkungan pengujian, lalu:

```powershell
# PLAYWRIGHT_MODULE boleh menunjuk instalasi Playwright yang sudah ada.
node docs/ui-guide/capture-panduan.cjs
python tools/encode-guide-videos.py --ffmpeg "C:\lokasi\ffmpeg.exe"
node tools/build-static.mjs
```

`--screenshots-only` atau `--videos-only` membatasi pekerjaan capture. Fixture menolak domain selain localhost/127.0.0.1. Tidak ada akun live yang dipakai; tidak ada email reset atau penghapusan data sungguhan. MP4 tanpa suara, 720p, di bawah 5 MB, dengan caption Indonesia, transkrip, kontrol native, dan pilihan kecepatan. Video memakai `preload="none"` tanpa autoplay.

## Pemeriksaan

```powershell
node --test frontend/tests/*.test.mjs
node tools/serve-guide-preview.mjs
node frontend/tests/browser/panduan.cjs
```

Browser test menerima `GUIDE_URL` untuk menguji preview. Jangan menjalankan fixture transaksi pada domain produksi. Tutorial catat harian menjelaskan jalur CSV yang tersedia; tidak mengarang formulir input manual di Ringkasan.
