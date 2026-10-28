# EFA di Vercel, API di server tim

Sejak 28 Oktober 2026, https://efa.madgo.my.id/ menyajikan landing dari
`landing/`, dan `/app` menyajikan aplikasi dari `frontend/`. Browser
aplikasi memanggil https://temukas.madgo.my.id/api/v1 langsung.
Vercel tidak menjalankan Go atau menyimpan transaksi EFA lagi. PostgreSQL
tetap dikelola backend tim; tidak ada migration atau pemindahan data lama.

## Alur request

1. Vercel mengirim HTML, CSS, JavaScript, foto dan font.
2. `assets/api-config.js` menjadi satu sumber alamat API. Pada localhost,
   alamat kosong memakai proxy `dev.mjs`; pada production, alamatnya backend tim.
3. `api-client.js` mengirim `credentials: include`, membaca error `message`,
   dan membawa CSRF dari memori pada request non-GET setelah login.
4. Cookie `rku_session` dipasang oleh backend untuk host `temukas.madgo.my.id`.
   Browser mengirimkannya langsung ke host tersebut. Cookie tidak dibaca JS.
5. `EventSource` memakai URL backend absolut dan `withCredentials: true`.
   Setiap koneksi/reconnect mengambil snapshot REST; event state membawa versi.
6. Tautan ekspor menuju backend secara langsung, tanpa atribut download
   lintas-origin. Backend mengirim attachment `umkm-rekap.csv`.

Origin EFA dan backend sama-sama HTTPS di bawah `madgo.my.id`, sehingga
cookie SameSite=Strict dapat digunakan. Domain preview Vercel bukan situs
yang sama dan tidak diizinkan oleh CORS production.

## Konfigurasi hosting

`vercel.json` menjalankan `node tools/build-static.mjs` dan memilih output
`dist`, tanpa install atau rewrite API. `cleanUrls` membuat `/app` membuka
`app.html`. Tautan hash aplikasi lama di root diarahkan ke `/app`.
CSP mengizinkan `connect-src` hanya ke origin sendiri dan backend tim.
Build hanya menyalin HTML dan aset frontend/landing; backend, server dev,
tes, dokumentasi, `.env`, dan konfigurasi privat tidak menjadi aset web.

Deployment handoff sebelum landing: `dpl_Akkq1WdUiHEjhek6n15i5ChseRKB`.
Konfigurasi routing landing dijelaskan pada `landing/README.md`.
Deployment landing: `dpl_6WZ8pXVMXfjDdb6AZRyWvwPjtkeJ`, domain tetap
https://efa.madgo.my.id/. Login, 21 tampilan aplikasi, handshake SSE dan
ekspor CSV diperiksa ulang setelah `/app` diterbitkan; semuanya berhasil.

Deploy dari **root repository ini**, bukan folder prototipe dalam archives:

```powershell
vercel deploy --prod --yes --scope hash-a09b
```

Gunakan proyek Vercel `temukas` yang sudah terhubung ke domain EFA. Tidak
perlu mengubah DNS atau memasukkan credential PostgreSQL ke frontend.
Jangan menjalankan deployment baru dari archives karena akan mengaktifkan
kembali monolit dan database prototipe.

## Hasil pengujian

**Live, tanpa mengubah transaksi:**

- Config dan preflight CORS berhasil; allow-origin tepat EFA dan credentials true.
- 401 awal pada auth/me membuka form login, bukan layar backend terputus.
- Akun demo `pemilik@temukas.test` berhasil masuk dengan akun database tim.
- Cookie terbukti HttpOnly, Secure, SameSite=Strict.
- Tujuh halaman pada lebar 375/768/1440 px: 21 pemeriksaan tanpa overflow
  atau page error JavaScript.
- Handshake SSE berhasil. Event versi tidak harus dikirim saat koneksi
  pertama dibuka; tanpa mutasi, tidak ada event perubahan yang perlu muncul.
- Ekspor berhasil mengunduh `umkm-rekap.csv` dari domain backend.
- Tidak ditemukan request ke API prototipe atau `/api/...` milik EFA.

**Database lokal disposable `rku_handoff_test`, melalui browser:**

- Register workspace sintetis, preview dan commit CSV penjualan/penerimaan.
- Alokasi Rp190.000 + biaya Rp10.000 untuk penjualan Rp200.000.
- Event state SSE benar-benar diterima setelah alokasi.
- Undo alokasi dan pembatalan kedua impor berhasil.
- Mutasi tanpa CSRF ditolak 403.
- Ekspor CSV, perubahan password melalui PUT, login dengan password baru,
  dan logout berhasil. Dialog password dibersihkan saat kembali ke login.

Seluruh tes Go dengan `TEST_REQUIRE_DB=1` pada `rku_integration_test`,
`go vet`, `go build`, serta lima tes frontend lulus. Pengujian mutasi
produksi tidak dijalankan; tidak ada akun atau data transaksi tim yang
dibuat/diubah oleh skenario uji ini.

## Akun dan rollback

Akun prototipe lama tidak otomatis tersedia di database tim. Gunakan akun
tim atau daftar lewat UI. Password dan CSRF tidak disimpan di localStorage.

Deployment prototipe sebelum handoff tetap tersedia sebagai rollback:
`dpl_GxzYHVENzfadiD1K9y3YYxUS2o8D`. Mengembalikannya juga mengaktifkan
database/sesi prototipe yang berbeda. Pertimbangkan dampak ini sebelum
rollback; jangan menyatakan data kedua database otomatis tersinkron.

Backup repo sebelum pull tersedia privat di folder Backups pada laptop.
Referensi instruksi: [handoff tim](HANDOFF_VERCEL_FRONTEND.md).
