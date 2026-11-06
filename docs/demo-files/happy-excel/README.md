# Happy path 33 baris (Excel penuh)

- `penjualan-happy-33.xlsx` - 33 penjualan, sheet `Penjualan`, header baris 1.
- `mutasi-happy-33.xlsx` - 33 penerimaan, sheet `Mutasi`, header baris 1.

Isi: 29 pasangan dengan referensi, nominal, kanal, dan tanggal yang sama; 4
pasangan sengaja memakai nama pembayar berbeda (SITI, DINA, RINA, DEWI) sehingga
skornya tepat 0,90: referensi 0,35 + nominal 0,25 + kanal 0,15 + tanggal 0,15,
tanpa poin nama.

Hasil impor yang diharapkan: 29 pasangan aturan, 4 pasangan AI, 0 temuan,
sisa tagihan Rp0.

Agar AI ikut berjalan, server perlu `AI_ENABLED=true`, `GEMINI_API_KEY` terisi,
`GEMINI_MODEL=gemini-2.5-flash`, `GEMINI_TIMEOUT=30s`, dan `AI_MIN_SCORE=0.90`.
Keempat pasangan berhenti di skor 0,90 sehingga hanya terkirim bila
`AI_MIN_SCORE` tidak lebih tinggi dari 0,90. Tanpa AI, keempatnya tetap
tersimpan dan muncul sebagai temuan di Verifikasi.

Batas server: 500 baris data per impor, 10 MB untuk unggahan XLSX, 1 MB untuk CSV.
