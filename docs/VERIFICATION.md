# Verifikasi integrasi Temu Kas

Pengujian dilakukan pada 27 Oktober 2026. Basis tim sudah mencakup main 6cc2d6e (merge SSE). Hasil ini berlaku pada konfigurasi lokal yang diuji, bukan jaminan bebas bug untuk semua hosting.

## Lingkungan

- Go 1.25.3, Node.js, Chrome headless dengan Playwright.
- PostgreSQL 12.4 lokal disposable, terpisah dari database pengguna.
- Database integration test dan browser demo terpisah; tidak mengakses database domain tim.
- UI melalui proxy lokal 127.0.0.1:3055 ke Go 127.0.0.1:18080.
- CI tim menggunakan PostgreSQL 16 dan menjalankan race detector.

## Hasil

| Pemeriksaan | Hasil lokal |
| --- | --- |
| go test -count=1 ./... dengan TEST_DATABASE_URL dan TEST_REQUIRE_DB=1 | Lulus, termasuk service, handler, database dan SSE |
| go vet ./... | Lulus |
| go build ./... | Lulus |
| Cross-build API linux/amd64, CGO_ENABLED=0 | Lulus |
| node --test tests/*.test.mjs | Lulus: aset, isolasi path, proxy PUT/CSRF, forwarding cookie |
| Registrasi dari UI sampai dashboard | Lulus |
| Preview dan commit kedua template CSV | Lulus |
| File identik tidak membuat entri kedua | Lulus |
| Pembayaran sebagian, fee marketplace, penerimaan gabungan | Lulus |
| Cancel impor dengan alokasi aktif | Ditolak 409 sesuai aturan |
| Undo pasangan lalu cancel impor | Lulus |
| Ekspor CSV | Lulus |
| Akses tanpa sesi | Ditolak 401 |
| Ganti password lalu login kembali | Lulus |
| SSE perubahan transaksi terlihat di tab kedua | Lulus, melalui proxy yang sama |
| Tujuh halaman pada 375/768/1440 px | Tidak ada overflow halaman |
| Runtime JavaScript selama alur browser | Tidak ada pageerror |
| Cookie HTTPS proxy, CORS header CSRF, limiter 429 | Lulus regression test handler |

Screenshot dan keluaran browser lokal disimpan di test-results yang diabaikan Git. API regression test yang ada pada tim tetap menjadi tes aturan saldo, konkurensi dan isolasi workspace; tes baru tidak menggantikan suite tersebut.

## Menjalankan ulang

Backend:

```sh
TEST_DATABASE_URL='postgres://.../database_uji?sslmode=disable' TEST_REQUIRE_DB=1 go test -count=1 ./...
go vet ./...
go build ./...
```

**Database uji akan di-reset oleh harness. Jangan gunakan database produksi, staging, atau demo yang perlu dipertahankan.**

Frontend: dari frontend jalankan node --test tests/*.test.mjs. Untuk pemeriksaan manual, ikuti [DEMO](DEMO.md), lalu tes navigasi melalui ukuran 375/768/1440 px dan prefers-reduced-motion.

## Batas verifikasi

Belum melakukan deployment ke server teman, tes restore backup hosting, uji beban multi-instance, atau mengukur waktu kerja UMKM nyata. UI asli dipertahankan, tetapi implementasi produksi tetap harus diverifikasi setelah pemasangan proxy dan TLS. Dockerfile tidak diubah dan build Docker tidak dijalankan. Status lint/race Linux ditentukan oleh GitHub Actions pada PR.
