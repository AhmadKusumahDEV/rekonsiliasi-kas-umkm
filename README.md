# Easy Financial Audit (EFA)

Nama tampilan baru Temu Kas. [Palet, logo dan favicon](docs/BRAND.md)
mengikuti referensi terbaru; folder, data dan endpoint tetap kompatibel.

Temu Kas membantu pemilik UMKM mencocokkan penjualan dengan uang masuk, termasuk pembayaran sebagian, pencairan gabungan, dan biaya layanan. Tampilan memakai prototipe Temu Kas milik Evan; backend memakai struktur tim: Go, Echo, PostgreSQL, pgx, sqlc, dan Viper.

**Live:** https://efa.madgo.my.id/ menyajikan landing pengenalan produk dari
`landing/`; aplikasi ada di https://efa.madgo.my.id/app. Keduanya statis di
Vercel. Browser aplikasi memanggil backend Go/Echo tim langsung di
https://temukas.madgo.my.id/api/v1, dengan cookie `rku_session` dan SSE.
Backend serta database prototipe Vercel tidak lagi dipakai. Lihat
[hasil handoff dan cara deploy](docs/VERCEL_TEAM_API.md).

## Mulai lokal

Prasyarat: Go sesuai backend/go.mod, PostgreSQL, dan Node.js 22+. Frontend adalah HTML/CSS/JavaScript biasa; tidak membutuhkan React, Vite, atau instalasi npm.

1. Salin backend/configs/local.yaml.example menjadi backend/configs/local.yaml. Aktifkan database dan isi koneksi lokal. File ini tidak ikut Git.
2. Terapkan migration sesuai [panduan backend](backend/README.md). Pakai database kosong untuk demo.
3. Dari backend, jalankan `go run ./cmd/api`.
4. Dari frontend di terminal kedua, jalankan `node dev.mjs`. Buka http://127.0.0.1:3000 untuk landing atau http://127.0.0.1:3000/app#login untuk aplikasi.

API default http://127.0.0.1:8080. Isi environment API_UPSTREAM jika alamat API berbeda, PORT untuk port frontend. .env.example backend tidak dibaca otomatis.

Setelah konfigurasi database siap, Windows juga bisa memakai run.bat untuk build
dan menjalankan kedua server. Ctrl+C menghentikan keduanya; port yang sudah
dipakai tidak diambil paksa. PORT mengatur UI, SERVER_PORT mengatur API.

## Struktur

| Folder | Isi |
| --- | --- |
| landing | Halaman pengenalan, carousel, cara kerja, referensi dan artikel coming soon |
| tools/build-static.mjs | Menggabungkan aset publik ke dist untuk Vercel; tidak membundel API |
| frontend/assets | UI, foto, font, animasi, dan contoh CSV Temu Kas |
| frontend/dev.mjs | Server statis dan proxy lokal tanpa dependency |
| frontend/tests | Tes isolasi file dan forwarding cookie/CSRF |
| backend/cmd/api | Composition root API |
| backend/cmd/seed-demo | Pembuat akun sintetis secara manual |
| backend/internal/echohttp | Routing, middleware, binding dan respons HTTP |
| backend/internal/service | Otorisasi, aturan kas, transaksi database |
| backend/internal/models | Kontrak input/output |
| backend/scripts/postgres | Migration dan sumber query sqlc |
| backend/pkg | Utilitas dan infrastruktur |
| docs | Produk, kontrak fungsi, demo, hosting dan hasil tes |

Folder placeholder backend tetap dipertahankan sesuai konvensi tim. Tidak ada UI Grader/Facility atau monolit tiga prototipe dalam aplikasi ini.

## Dokumentasi

- [Panduan seluruh UI - PDF](docs/EFA-UI-Guide.pdf) dan [indeks screenshot](docs/ui-guide/SCREENSHOT-INDEX.md)
- [Daftar dokumentasi](docs/README.md)
- [Arsitektur, direktori, alur dan semua endpoint](architecture.md)
- [PRD](docs/PRD.md) dan [FRD](docs/FRD.md)
- [Akun demo](docs/DEMO.md), [alur API](docs/INTEGRATION.md)
- [Hosting](docs/HOSTING.md), [hasil tes](docs/VERIFICATION.md)
- [Atribusi aset](frontend/ASSETS.md)
- [Landing page: edit, run dan build](landing/README.md), [sumber foto dan analisis](landing/SOURCES.md)
- [Backend](backend/README.md), [kontribusi](CONTRIBUTING.md)
- [PRD awal](docs/reference/PRD-prototype.md), [FRD awal](docs/reference/FRD-prototype.md): dokumen lengkap sebelumnya, termasuk dua ide di luar Temu Kas.

## Tes

Dari backend: `go test ./...`, `go vet ./...`, `go build ./...`.

Tes database memerlukan TEST_DATABASE_URL. Set TEST_REQUIRE_DB=1 agar tidak diam-diam skip. **Harness menghapus schema database uji; jangan arahkan ke database pengguna.**

Dari frontend: `node --test tests/*.test.mjs`. GitHub Actions juga menjalankan backend dengan PostgreSQL, race detector, lint, build Linux, dan tes frontend.

Data contoh bersifat sintetis. Ini proyek PBL, bukan klaim layanan keuangan resmi MAXY atau produk yang sudah digunakan UMKM nyata.
