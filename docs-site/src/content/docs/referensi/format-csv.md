---
title: "Format CSV yang diterima EFA"
description: "Kolom penjualan sederhana, mutasi bank, dan batas masukan yang perlu diperiksa."
lastUpdated: 2026-11-05
---

## Penjualan

[Unduh template penjualan](/assets/template-penjualan.csv). Gunakan CSV UTF-8, bukan XLSX, PDF, gambar, atau dokumen yang hanya diganti nama ekstensi.

| Kolom | Isi | Wajib |
| --- | --- | --- |
| tanggal | Tanggal transaksi, disarankan YYYY-MM-DD | Ya |
| referensi | Nomor pesanan, misalnya INV-101 | Tidak |
| nama | Nama pelanggan atau Penjualan harian | Ya |
| nominal | Nilai penjualan dalam Rupiah | Ya |
| kanal | tunai, qris, transfer, marketplace, atau e-wallet | Ya |

```csv
tanggal,referensi,nama,nominal,kanal
2026-11-03,INV-101,Budi,150000,tunai
2026-11-03,INV-102,Rani,200000,qris
```

Pakai angka tanpa pemisah ribuan untuk contoh yang mudah diperiksa. Format bernilai koma harus diapit tanda petik sesuai aturan CSV. Referensi boleh kosong; jangan mengarang nomor transaksi hanya agar sistem memasangkan.

## Mutasi bank

[Unduh contoh mutasi](/assets/contoh-mutasi-bank.csv). Isi slot **Mutasi bank**, bukan Penjualan.

```csv
Tanggal,Keterangan,Debit,Kredit,Saldo
2026-11-03,INV-102 Rani,0,190000,2500000
2026-11-03,BIAYA ADMIN,10000,0,2490000
```

**Kredit** mewakili uang masuk. **Debit** bukan penerimaan. **Saldo** tidak dijumlahkan sebagai transaksi. Baca peringatan pratinjau karena tidak semua variasi laporan bank atau marketplace dapat diterima dengan struktur yang sama.

## Batas dan duplikat

Lihat batas file dan baris pada aplikasi; nilainya berasal dari server. Unggahan identik dapat dilewati sebagai duplikat. File dengan data yang diubah tidak otomatis dianggap sama: jangan mengunggah total harian dan rincian pesanan yang sama sebagai dua sumber penjualan.

Untuk satu pencairan marketplace, pilih satu sumber penerimaan agar settlement dan mutasi bank tidak dihitung dua kali. Jika formatnya belum dikenali, ubah ke template yang didukung dan periksa pratinjau sebelum menyimpan.

[Ikuti langkah impor](/panduan/tugas/impor/).
