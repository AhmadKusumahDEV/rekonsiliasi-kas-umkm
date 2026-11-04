# Riset Matching Engine: Tools dan Arsitektur

Pertanyaan yang dijawab: apakah ada tools siap pakai untuk mencocokkan
penjualan dan penerimaan, dan apakah alur "cocokkan dulu, lalu kirim ke AI"
sudah benar.

Jawaban singkat: tools ada, tetapi tidak ada yang cocok disematkan langsung ke
backend Go kecil. Yang paling masuk akal adalah engine tipis sendiri dengan
teknik yang sudah terbukti, ditambah AI hanya sebagai penilai akhir untuk
kandidat ambigu.

## Tools yang ada

| Tool | Kekuatan | Kenapa tidak dipakai langsung |
| --- | --- | --- |
| Splink | Fellegi-Sunter probabilistik, blocking SQL, ada contoh transaksi bank | Runtime Python terpisah; cocok sebagai laboratorium batch, bukan jalur online |
| Dedupe | Active learning, deduplikasi | Dirancang untuk entitas, bukan pasangan transaksi 1:1 |
| Zingg | ML entity resolution | Butuh Spark, lisensi AGPL, berat |
| Python RecordLinkage | Toolkit lengkap, BSD | Batch di memori, bukan service Go |
| Senzing | Entity resolution komersial, SDK Go | Harga puluhan ribu dolar per tahun, model identitas bukan ledger |
| Tamr | MDM enterprise | SaaS besar, tidak khusus rekonsiliasi transaksi |
| Modern Treasury | Aturan rekonsiliasi 1:1, 1:N, N:1 | Platform treasury, bukan library |
| QuickBooks rules | Pola aturan bank yang terbukti | Produk tertutup, tetapi polanya bisa ditiru |
| Ramp | Pencocokan struk ke kartu | Hanya untuk ekosistem Ramp |
| sahilm/fuzzy | Fuzzy search Go | Hanya untuk pencarian nama, bukan nominal/tanggal |

Kesimpulan: **bangun engine tipis di Go + PostgreSQL**, tiru pola QuickBooks
untuk aturan, tiru Splink untuk skor probabilistik, dan pakai Gemini hanya
untuk kandidat yang masih ambigu.

## Apakah alur kita benar

Ya, dengan satu penegasan: AI tidak menerima seluruh file. Alurnya:

```text
normalisasi -> blocking kandidat -> skor aturan ->
  skor tinggi: simpan otomatis
  skor menengah: AI memilih di antara kandidat
  skor rendah: tidak dipasangkan
```

AI adalah lapisan ketiga, bukan mesin utama. Jika kandidat benar tidak masuk
daftar, AI tidak akan pernah bisa memperbaikinya. Karena itu blocking adalah
bagian yang paling menentukan kualitas.

## Yang perlu diubah di kode sekarang

1. **Blocking.** Ganti perbandingan semua pasangan dengan bucket: nominal,
   jendela tanggal, token referensi, dan kemiripan nama. Ini menghapus batas
   40.000 kombinasi dan membuat data besar tetap cepat.
2. **Assignment eksklusif.** Pilih pasangan dengan maximum-weight bipartite
   matching, bukan greedy. Dua penjualan Rp100.000 tidak boleh berebut satu
   penerimaan.
3. **Aturan pengguna.** Tambah bank rules sederhana: kondisi rekening, arah,
   deskripsi mengandung teks, nominal; aksi mengisi kanal, kategori, atau tag.
4. **AI listwise.** Kirim 5-10 kandidat dalam satu panggilan, minta satu
   keputusan dengan `candidate_id`, `evidence`, dan opsi `ABSTAIN`.
5. **Bukti, bukan angka.** Tampilkan "nominal sama, referensi INV-102, beda 1
   hari", bukan hanya skor 0,93.

## Teknik yang dipakai

| Lapisan | Teknik |
| --- | --- |
| Recall | Blocking nominal, tanggal, referensi; `pg_trgm` untuk nama |
| Skor | Bobot transparan, lalu regresi logistik setelah ada data berlabel |
| Assignment | Maximum-weight bipartite matching |
| AI | Gemini structured output untuk kandidat 0,60-0,95 |
| Audit | Simpan sinyal, alasan, versi aturan, dan koreksi manusia |

## Yang tidak perlu

- Spark, Dedupe, atau Zingg untuk 10 layar.
- Senzing atau Tamr.
- Embedding sejak hari pertama; tambahkan hanya jika recall kandidat terbukti
  kurang.
- Auto-post berdasarkan skor AI saja.

## Metrik

- Pair completeness: berapa persen pasangan benar masuk kandidat.
- Precision auto-match dan false auto-match rate.
- Review rate dan override rate per band.
- Waktu penyelesaian per 100 transaksi.

## Urutan implementasi

| Langkah | Isi | Estimasi |
| --- | --- | --- |
| 1 | Blocking dan uji performa | 1-2 hari |
| 2 | Bipartite matching dan aturan konflik | 1-2 hari |
| 3 | Bank rules sederhana | 1-2 hari |
| 4 | AI listwise pada zona 0,60-0,95 | 2 hari |
| 5 | Gold set dan evaluasi precision | 1-2 hari |

Total 6-10 hari, dan langkah 1-3 sudah memberi manfaat tanpa AI.

