# Scope Kelayakan Demo

Target: demo yang menunjukkan alur impor, pencocokan otomatis, dan antrean
verifikasi tanpa terlihat berantakan. Bukan produksi penuh.

## Definisi layak demo

1. Pengguna mengunggah rekap penjualan dan mutasi bank dalam satu layar.
2. Pasangan yang yakin tersimpan otomatis tanpa klik.
3. Penjualan tunai tidak muncul sebagai selisih.
4. Bunga, biaya admin, dan transfer antar rekening tidak masuk antrean.
5. Setiap item yang tersisa punya alasan yang bisa dibaca.
6. Pengguna bisa menyetujui pasangan dari satu halaman Verifikasi.

## Daftar pekerjaan

| Prioritas | Pekerjaan | Kenapa perlu untuk demo | Estimasi |
| --- | --- | --- | --- |
| P0 | Penjualan tunai otomatis lunas | Menghapus kartu review palsu | 3-4 jam |
| P0 | Kategori non-operasional: bunga, biaya admin, transfer antar rekening | Bukan penjualan, jangan tampil sebagai selisih | 4-5 jam |
| P0 | Tampilkan alasan pada setiap saran dan temuan | Pengguna butuh tahu mengapa | 3-4 jam |
| P0 | Layar impor dua slot dengan endpoint fleksibel | Menunjukkan format longgar | 5-6 jam |
| P0 | Halaman Verifikasi | Menggantikan Cocokkan kas | 6-8 jam |
| P0 | Data demo dan skrip pemuatan | Demo bisa diulang | 2-3 jam |
| P1 | Tampilkan `auto_matched` dan warnings setelah impor | Menjelaskan hasil otomatis | 2-3 jam |
| P1 | Uji alur end-to-end desktop dan mobile | Menghindari demo gagal | 3-4 jam |

Total P0 sekitar 23-30 jam kerja agent, atau 2-3 hari kalender dengan review.
P1 menambah sekitar 5-7 jam.

## Kriteria selesai

- Impor 14 penjualan dan 14 baris mutasi selesai dalam satu alur.
- Minimal tiga pasangan tersimpan otomatis dan tidak ada yang salah.
- Penjualan tunai tidak muncul di antrean.
- Bunga, biaya admin, dan transfer antar rekening tidak muncul di antrean.
- Setiap kartu sisa menampilkan alasan dan nominal.
- Halaman Verifikasi bisa menyetujui pasangan tanpa pindah halaman.
- Test backend dan frontend lulus.

## Yang tidak dikerjakan untuk demo

- Unggah XLSX dan multipart.
- PDF dan screenshot.
- AI live dan API key.
- Bank rules buatan pengguna.
- Blocking dan bipartite matching.
- Peran pengguna baru.
- Multi-mata uang.

## Rencana uji

| Uji | Data | Harapan |
| --- | --- | --- |
| Alur bersih | 14 penjualan, 14 mutasi | 3 otomatis, sisanya beralasan |
| Tunai | Satu penjualan tunai | Tidak masuk antrean |
| Non-operasional | Bunga, biaya, transfer sendiri | Tidak masuk antrean |
| Nominal beda | Fee 6500 | Masuk review, bukan otomatis |
| Nama beda | Budi vs Siti | Masuk review |
| Referensi salah | INV-O11 | Masuk review |
| Dua nominal sama | Dua transaksi Rp95.000 | Tidak dipasangkan otomatis |
| File keluar semua | Mutasi debit saja | Warning, bukan error |

## Urutan pengerjaan

1. Backend: tunai lunas dan kategori non-operasional, plus test.
2. Backend: alasan pada saran dan temuan.
3. Frontend: impor dua slot dan tampilan warnings.
4. Frontend: halaman Verifikasi dan aksi setujui.
5. Data demo, uji end-to-end, dan perbaikan.

## Status implementasi 4 November 2026

| Item | Status |
| --- | --- |
| Tunai lunas otomatis | Selesai, termasuk varian Cash dan kasir |
| Kategori non-operasional | Selesai, frasa spesifik dan bisa ditambah lewat env |
| Alasan pada saran dan temuan | Selesai di API |
| Endpoint impor fleksibel | Selesai |
| `auto_matched` dan warnings | Selesai di API |
| Data demo bersih | Selesai, 11 otomatis dan 3 temuan |
| Data demo edge case | Selesai, 3 otomatis dan 18 temuan |
| Test regresi dan lock database | Selesai |
| Layar impor dua slot | Menunggu frontend |
| Halaman Verifikasi | Menunggu frontend |
| Uji end-to-end browser | Menunggu frontend |

Kontrak frontend ada di [FRONTEND_VERIFICATION_CONTRACT.md](FRONTEND_VERIFICATION_CONTRACT.md).


## Yang perlu diputuskan sebelum mulai

1. Tunai lunas otomatis untuk semua kanal `tunai`, atau bisa dimatikan?
2. Kategori non-operasional memakai daftar kata kunci atau pilihan pengguna?
3. Halaman Verifikasi menggantikan Cocokkan kas, atau hanya menambah tab?

