# Happy path 33 baris (Excel penuh)

- `penjualan-happy-33.xlsx` - 33 penjualan, sheet `Penjualan`, header baris 1.
- `mutasi-happy-33.xlsx` - 33 penerimaan, sheet `Mutasi`, header baris 1.

Isi: 29 pasangan dengan referensi, nominal, kanal, dan tanggal yang sama; 4
pasangan sengaja memakai nama pembayar berbeda (SITI, DINA, RINA, DEWI) sehingga
skor jatuh di zona 0,60-0,95 dan diputuskan AI.

Hasil impor yang diharapkan: 29 pasangan aturan, 4 pasangan AI, 0 temuan,
sisa tagihan Rp0.

AI hanya berjalan bila `GEMINI_API_KEY` aktif di server. Tanpa kunci, keempat
pasangan tetap tersimpan sebagai temuan di Verifikasi.
