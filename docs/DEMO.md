# Akun demo

Akun berikut sudah dibuat dan diuji di **database lokal pengujian**. Belum otomatis tersedia pada domain tim sebelum seed dijalankan di database demo domain itu.

| Email / username | Password | Usaha | Peran |
| --- | --- | --- | --- |
| pemilik@temukas.test | KasDemo!2026 | Roti Sore (Demo) | Pemilik |
| pemilik.dua@temukas.test | UsahaDemo!2026 | Kedai Pagi (Demo) | Pemilik |

Password sengaja publik untuk data sintetis. Jangan gunakan akun ini untuk data finansial nyata.

## Buat akun

Siapkan database demo terpisah, jalankan migration, dan isi konfigurasi backend. Dari backend:

PowerShell:

```powershell
$env:ALLOW_DEMO_SEED='true'
go run ./cmd/seed-demo
Remove-Item Env:ALLOW_DEMO_SEED
```

Linux: `ALLOW_DEMO_SEED=true go run ./cmd/seed-demo`.

Seed memakai service API dan template ../frontend/assets. Akun yang sudah ada dilewati tanpa reset password atau overwrite. Jika seed terputus setelah akun dibuat, impor template melalui UI untuk melengkapinya.

## Skenario demo

1. Login akun pertama: lima penjualan Rp650.000, empat penerimaan Rp590.000.
2. Cocokkan kas: periksa dan simpan usulan unik INV-102/Rani Rp100.000.
3. INV-101 dibayar Rp100.000 dari tagihan Rp150.000, sisa Rp50.000.
4. INV-103: penerimaan Rp190.000 + fee Rp10.000 dengan alasan.
5. Penerimaan Kirana Rp200.000 dibagi ke INV-104 Rp80.000 dan INV-105 Rp120.000.
6. Unduh laporan, undo keputusan, lihat jejak di Riwayat.
7. Cancel impor beralokasi ditolak. Undo pasangan dahulu sebelum membatalkan batch.
8. Login akun kedua di browser berbeda: keputusan akun pertama tidak ikut berubah.

Template di frontend/assets juga tersedia dari dialog impor. Semua transaksi contoh sintetis.
