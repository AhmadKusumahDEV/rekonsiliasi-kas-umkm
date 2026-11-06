# File Demo Impor

Dataset: 33 penjualan dan 32 mutasi bank. File dibuat oleh
`tools/generate_demo_imports.py`.

## File penjualan

| File | Format | Kegunaan |
| --- | --- | --- |
| `penjualan-33.csv` | CSV koma | Demo utama |
| `penjualan-33-semicolon.csv` | CSV titik koma | Uji delimiter |
| `penjualan-33.xlsx` | XLSX | Demo Excel |
| `penjualan-33-header8.xlsx` | XLSX, header baris 8 | Uji metadata di atas tabel |
| `penjualan-header-duplikat.csv` | CSV rusak | Harus ditolak |

## File mutasi

| File | Format | Kegunaan |
| --- | --- | --- |
| `mutasi-32.csv` | CSV koma | Demo utama |
| `mutasi-32-semicolon.csv` | CSV titik koma | Uji delimiter |
| `mutasi-32.xlsx` | XLSX | Demo Excel |
| `mutasi-32-multisheet.xlsx` | XLSX dua sheet | Uji pemilih sheet |
| `mutasi-total-row.csv` | CSV dengan baris TOTAL | Baris total harus dilewati |

## Kombinasi demo

| Skenario | Penjualan | Mutasi | Yang ditunjukkan |
| --- | --- | --- | --- |
| A | `penjualan-33.csv` | `mutasi-32.csv` | Alur CSV standar |
| B | `penjualan-33.xlsx` | `mutasi-32.xlsx` | Alur Excel penuh |
| C | `penjualan-33.csv` | `mutasi-32.xlsx` | Campuran CSV dan Excel |
| D | `penjualan-33.xlsx` | `mutasi-32.csv` | Campuran Excel dan CSV |
| E | `penjualan-33-header8.xlsx` | `mutasi-32-multisheet.xlsx` | Header baris 8 dan pemilih sheet |
| F | `penjualan-header-duplikat.csv` | `mutasi-32.csv` | Penolakan header duplikat |
| G | `penjualan-33.csv` | `mutasi-total-row.csv` | Baris total dilewati |

## Isi dataset

- 20 pasangan exact, siap dicocokkan otomatis.
- 4 pasangan AI: referensi, nominal, kanal, dan tanggal sama, nama berbeda.
- 3 pasangan selisih fee QRIS Rp6.500.
- 2 penjualan dan 2 penerimaan nominal sama tanpa referensi.
- 2 penjualan tunai.
- 1 penjualan belum dibayar.
- 1 pencairan marketplace dengan fee Rp10.000.
- 2 penerimaan non-operasional: bunga dan transfer antar rekening.

## Happy path murni

Folder `happy/` berisi 24 penjualan dan 24 penerimaan tanpa kasus fee,
ambigu, tunai, atau non-operasional.

| File | Isi |
| --- | --- |
| `happy/penjualan-happy-24.csv` | 24 penjualan |
| `happy/mutasi-happy-24.csv` | 24 penerimaan |

Hasil impor yang diharapkan: 20 pasangan aturan, 4 pasangan AI, 0 temuan.

## Happy path 33 baris

Dua folder ini memakai dataset bersih 33 penjualan dan 33 penerimaan: 29
pasangan cocok lewat aturan, 4 pasangan masuk zona ambigu dan diselesaikan AI.

| Folder | Penjualan | Mutasi |
| --- | --- | --- |
| `happy-excel/` | XLSX | XLSX |
| `happy-hybrid/` | CSV | XLSX |

Hasil impor yang diharapkan: 29 pasangan aturan, 4 pasangan AI, 0 temuan.
Angka 4 pasangan AI hanya tercapai bila server memasang `AI_ENABLED=true`,
`GEMINI_API_KEY`, dan `AI_MIN_SCORE` tidak lebih tinggi dari 0,90; rinciannya ada
di `happy-excel/README.md`.

## Edge case

Folder `edge-cases/` berisi enam skenario kecil. Semua memakai kolom resmi
`tanggal,referensi,nama,nominal,kanal` dan `Tanggal,Keterangan,Debit,Kredit,Saldo`;
yang dibuat sulit adalah isinya. Rincian hasil per skenario ada di
`edge-cases/README.md`.


## Cara memakai

Untuk demo, pilih satu file penjualan dan satu file mutasi dari kombinasi di
atas. Untuk menguji Excel, gunakan kombinasi B, C, D, atau E.
Untuk demo cepat tanpa temuan, gunakan `happy-excel/`. Untuk menunjukkan
penanganan selisih, gunakan `edge-cases/`.

