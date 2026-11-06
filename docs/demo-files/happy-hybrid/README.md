# Happy path 33 baris (CSV dan Excel)

- `penjualan-happy-33.csv` - 33 penjualan, dipisah koma.
- `mutasi-happy-33.xlsx` - 33 penerimaan, sheet `Mutasi`.

Dataset sama dengan `happy-excel/`, hanya format penjualannya CSV. Cocok untuk
menunjukkan bahwa penjualan dan mutasi bank boleh berbeda format.

Hasil impor yang diharapkan: 29 pasangan aturan, 4 pasangan AI, 0 temuan,
sisa tagihan Rp0.
Syarat agar AI aktif dan batas ukuran file sama dengan `happy-excel/README.md`.
