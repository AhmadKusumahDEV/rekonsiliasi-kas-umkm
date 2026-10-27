# Panduan UI Easy Financial Audit

**Pembaruan login/daftar:** [screenshot navy terbaru](auth-refresh/) diambil
dari domain live setelah deployment desain baru. PDF dan manifest di bawah
tetap menjadi snapshot versi sebelumnya; dokumentasi alur dan endpointnya
tetap relevan. Label sumber dan hash snapshot lama tidak diubah.

[PDF lengkap](../EFA-UI-Guide.pdf) memuat 42 screenshot dalam 8 kategori.
Halaman desktop dan HP dipisahkan. Screenshot panjang dibagi ke beberapa
halaman PDF agar tulisan tetap terbaca; PNG aslinya tetap utuh.

- [Indeks setiap screenshot dan nomor PDF](SCREENSHOT-INDEX.md)
- [Identitas visual EFA](../BRAND.md)
- [Arsitektur dan endpoint](../../architecture.md)

## Sumber gambar

**19 screenshot LIVE** berasal dari https://efa.madgo.my.id/. Pengambilan
menggunakan login demo biasa dan hanya membaca halaman/transaksi.
Form daftar tidak dikirim pada server live.

**23 screenshot LOCAL** memperlihatkan keadaan kosong, sebagian pembayaran,
biaya layanan, pencairan gabungan, koreksi, preview/error CSV, filter dan
pagination. Skenario dijalankan pada localhost:3044 dengan database
PostgreSQL rku_ui_guide yang terpisah. Akun dokumentasi lokal memakai data
sintetis; password dibuat acak di memori dan tidak ditulis ke laporan.

Source UI lokal berasal dari prototipe deployment yang sama. Hash app.js,
design.js, efa.css dan efa-mark.svg telah dibandingkan dengan live:
[provenance.json](provenance.json). Jangan mengubah label LOCAL menjadi LIVE.

Halaman hash tidak dikenal adalah fallback UI, bukan bukti respons HTTP 404.
Dialog konfirmasi bawaan browser dijelaskan lewat teks; gambar dialog
native tidak direkayasa. Animasi direduksi saat capture untuk mendapatkan
frame akhir yang stabil, bukan saat pemakaian normal.

## Isi folder

| File/folder | Kegunaan |
| --- | --- |
| screenshots/ | PNG utuh: kategori lalu desktop/mobile |
| manifest.json | Sumber, waktu, ukuran layar dan penjelasan 42 gambar |
| pdf-index.json | Nomor halaman, jumlah bagian dan SHA-256 PNG |
| SCREENSHOT-INDEX.md | Indeks yang dapat dibaca langsung di GitHub |
| provenance.json | Perbandingan hash aset live dan lokal |
| capture-ui.cjs | Capture browser; semua operasi data dibatasi localhost:3044 |
| build-pdf.py | Membuat PDF, bookmark, nomor halaman dan indeks |
| requirements.txt | Dependency Python pembuat PDF |

## Membangun ulang PDF

Gunakan Python 3.12+ dengan paket di requirements.txt. Dari root repository:

```sh
python -m pip install -r docs/ui-guide/requirements.txt
python docs/ui-guide/build-pdf.py
```

Script memakai font Segoe UI Windows atau DejaVu Sans Linux, dan Poppins dari
aset proyek. Untuk font alternatif, isi FONT_BODY dan FONT_BOLD dengan path
TTF yang tersedia. Node/Playwright tidak diperlukan untuk membangun ulang
dari PNG yang sudah ada.

Hasil utama selalu docs/EFA-UI-Guide.pdf. File sementara/render PDF disimpan
di .cache, bukan dimasukkan Git.

## Mengambil ulang screenshot

Perlu Node, Playwright dan Chrome. PLAYWRIGHT_MODULE dapat menunjuk instalasi
Playwright yang sudah ada, sehingga dependency browser automation tidak
ditambahkan ke frontend produksi.

1. Siapkan database lokal disposable khusus dokumentasi.
2. Jalankan source prototipe demo dengan DATABASE_URL yang eksplisit menuju
   database lokal itu, HOST=127.0.0.1, PORT=3044, dan SEED_DEMO=false.
3. Pastikan halaman lokal memakai endpoint /api milik prototipe, bukan
   /api/v1 milik backend integrasi. Penjelasan kedua versi ada di architecture.md.
4. Jalankan node docs/ui-guide/capture-ui.cjs dari root repository.
5. Bangun PDF lagi dan periksa hasil render.

Script membuat workspace sintetis baru **hanya pada localhost** untuk
menyusun contoh. Jangan menggantinya agar menulis ke produksi.
Login live memakai akun demo yang sudah dipublikasikan pada panduan demo,
bukan credential privat. Jika password demo berubah, sesuaikan input login
script tanpa menaruh password privat di Git.

## Batas pembuktian

Screenshot mendokumentasikan tampilan dan alur pada waktu capture. Ini
bukan bukti uji beban, audit keamanan menyeluruh, integrasi bank, manfaat
bisnis terukur, atau bahwa source backend integrasi sudah menjadi deployment
Vercel yang sedang tayang. Tidak ada transaksi akun demo utama diubah.
