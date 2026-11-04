# Format Impor v2: Penjualan Sederhana, Penerimaan Mengikuti Bank

Dokumen ini menetapkan format masukan yang dipakai pengguna, sementara format
kanonik internal tetap tujuh kolom. Tujuannya: pengguna tidak perlu menghafal
struktur teknis, tetapi sistem tetap menerima data yang konsisten.

## Prinsip

1. Audit membutuhkan dua sisi: penjualan dan penerimaan. Penjualan tunai
   dianggap lunas, jadi hanya sisi non-tunai yang perlu dicocokkan.
2. Sisi penjualan dibuat sesederhana mungkin karena pengguna yang mengetik.
3. Sisi penerimaan mengikuti bentuk mutasi bank atau laporan marketplace,
   karena itu sumber yang paling mungkin dimiliki pengguna.
4. Semua masukan dipetakan ke format kanonik oleh adapter, bukan oleh pengguna.

## Template penjualan sederhana

Cukup lima kolom, semuanya dalam bahasa Indonesia:

```csv
tanggal,referensi,nama,nominal,kanal
2026-11-03,INV-101,Budi,150000,tunai
2026-11-03,INV-102,Rani,100000,qris
2026-11-03,,Penjualan harian,850000,tunai
```

| Kolom | Wajib | Aturan |
| --- | --- | --- |
| `tanggal` | Ya | `YYYY-MM-DD`; format lain dinormalkan |
| `referensi` | Tidak | Boleh kosong; dipakai untuk memperkuat pencocokan |
| `nama` | Ya | Nama pelanggan atau “Penjualan harian” |
| `nominal` | Ya | Angka Rupiah, boleh `Rp1.250.000` |
| `kanal` | Ya | tunai, qris, transfer, marketplace, e-wallet |

Kolom `id` tidak perlu diisi pengguna. Sistem membuat ID deterministik dari
tanggal, referensi, nominal, dan hash file, sehingga unggahan ulang tidak
menggandakan data.

Baris agregat harian diperbolehkan: satu baris per tanggal dan kanal. Ini
paling cepat untuk warung yang tidak mencatat per pesanan.

## Penerimaan dari mutasi bank

Bentuk yang lazim:

```csv
Tanggal,Keterangan,Debit,Kredit,Saldo
2026-11-03,TRF DARI BUDI,0,150000,2500000
2026-11-03,BIAYA ADMIN,6500,0,2493500
```

Aturan pemetaan:

| Kolom bank | Menjadi | Catatan |
| --- | --- | --- |
| Tanggal transaksi | `tanggal` | Format `dd/mm/yyyy` dinormalkan |
| Keterangan | `referensi` dan `nama` | Referensi dicari dari pola INV/TRX |
| Kredit | `nominal` penerimaan | Hanya baris kredit yang menjadi penerimaan |
| Debit | biaya atau pengeluaran | Bukan penerimaan; biaya admin dilampirkan ke pasangan |
| Saldo | diabaikan | Bukan transaksi |

Baris pembuka seperti nama rekening, nomor rekening, dan periode dianggap
metadata dan dilewati. Header tabel dicari dalam 30 baris pertama.

## Penerimaan dari marketplace

Laporan pencairan sering memuat dua sisi sekaligus:

```csv
No Pesanan,Tanggal,Total Penjualan,Biaya,Dana Diterima
240803ABC,2026-11-01,250000,15000,235000
```

- `Total Penjualan` menjadi sisi penjualan.
- `Dana Diterima` menjadi sisi penerimaan.
- `Biaya` menjadi fee yang dilampirkan ke pasangan.
- Satu file marketplace boleh menghasilkan dua sisi, sehingga pengguna tidak
  perlu mengunggah dua file.

## Tiga lapisan uang di marketplace

Laporan marketplace tidak bisa diperlakukan sebagai satu jenis file. Ada tiga
lapisan yang harus dipisahkan:

| Lapisan | Isi | Contoh |
| --- | --- | --- |
| Pesanan | Nilai barang, diskon, ongkir | Laporan pesanan |
| Settlement | Fee, refund, dana yang dilepas ke saldo | Laporan penghasilan |
| Payout | Transfer saldo ke rekening bank | Laporan rekening penjual |

Konsekuensinya:

- Shopee dan Tokopedia umumnya butuh dua file untuk sisi lengkap: laporan
  pesanan/penghasilan untuk penjualan, lalu riwayat saldo atau mutasi bank
  untuk penerimaan.
- TikTok Shop bisa satu file karena ekspor settlement memuat order, statement,
  dan payment sekaligus.
- Lazada bergantung pada kelengkapan Account Statement; bila hanya Transaction
  Details, payout belum ada.
- Jangan mencampur settlement dan mutasi bank sebagai dua penerimaan. Pilih
  salah satu sebagai sumber kas, yang lain menjadi bukti pendukung.


## Format mutasi bank yang ditemukan

Riset terhadap sepuluh bank menunjukkan formatnya tidak seragam, dan PDF
justru menjadi bentuk yang paling luas. CSV dan XLSX lebih sering tersedia di
kanal bisnis atau bank tertentu.

| Bank | Format umum | Kolom khas | Ciri yang perlu diwaspadai |
| --- | --- | --- | --- |
| BCA | PDF e-Statement; CSV/HTML di KlikBCA Bisnis | Tanggal, Keterangan, Cabang, Jumlah, flag, Saldo | Tanggal tanpa tahun; flag DB/CR; deskripsi bisa mengandung koma |
| Mandiri | PDF Livin; Excel dilaporkan tersedia | Nominal bertanda, Saldo; varian lama Debit/Kredit | Header bilingual dua baris; tanggal dan jam terpisah |
| BRI | PDF dan CSV BRImo | Tanggal, Uraian, Teller, Debet, Kredit, Saldo | Rekening koran punya tiga kolom nominal |
| BNI | PDF wondr; CSV dilaporkan ada | Tanggal, Rincian, Nominal, Saldo | Nominal bertanda plus/minus; angka tanpa desimal |
| BTN | PDF e-Statement | Nominal tunggal dengan penanda D | Baris saldo pemindahan berulang tiap halaman |
| Permata | PDF | Tgl Trx, Tgl Valuta, Uraian, Debet, Kredit, Saldo | Ada baris SALDO AWAL sebelum transaksi |
| CIMB Niaga | PDF; CSV di OCTO Biz | Mutasi Debet, Mutasi Kredit, Saldo | Dua varian header; kolom No Cek/BG |
| SeaBank | PDF | Keluar, Masuk, Saldo Akhir | Keterangan ada di baris terpisah di atas transaksi |
| Jenius | PDF | Tiga kolom tanggal, Debet, Kredit, Saldo | Multi-akun dan multi-valuta |

Implikasi untuk implementasi:

- Prioritas pertama tetap CSV dan XLSX. PDF ditunda sampai ada pratinjau wajib
  dan file nyata untuk diuji.
- Header dicari dalam 30 baris pertama, karena identitas rekening dan periode
  biasanya berada di atas tabel.
- Kolom yang mengandung saldo tidak pernah menjadi nominal.
- Arah dana ditentukan berurutan: kolom debit/kredit, lalu flag DB/CR, lalu
  tanda plus/minus, terakhir selisih saldo.
- Hanya baris masuk yang menjadi penerimaan. Biaya admin adalah pengeluaran,
  bunga adalah penerimaan non-operasional, keduanya tidak dicocokkan ke
  penjualan.
- Jangan menulis adapter final tanpa salinan file anonim dari tiap bank;
  dokumentasi saja tidak cukup.


## Ringkasan pemetaan

| Sumber | Sisi penjualan | Sisi penerimaan |
| --- | --- | --- |
| Template penjualan | Semua baris | Tidak ada |
| Mutasi bank | Tidak ada | Baris kredit |
| Laporan marketplace | Total penjualan | Dana diterima |
| Rekap harian | Satu baris agregat | Tunai langsung lunas |

## Validasi yang tetap berlaku

- Nominal harus lebih dari nol dan maksimal satu triliun.
- Tanggal harus valid; tanggal mustahil ditolak.
- ID internal unik; unggahan ulang idempoten.
- Baris dengan nama atau kanal kosong ditolak dengan nomor baris.
- Saldo bank tidak pernah dianggap nominal transaksi.

## Yang perlu diverifikasi dengan file nyata

1. Ekspor BCA, Mandiri, BRI, BNI: posisi header dan nama kolom.
2. Format tanggal dan delimiter tiap bank.
3. Laporan pencairan Shopee, Tokopedia, TikTok Shop, Lazada.
4. Apakah biaya dan refund muncul sebagai kolom atau baris terpisah.

Tanpa file nyata, adapter hanya boleh dianggap selesai setelah diuji dengan
salinan anonim dari masing-masing sumber.

