# Skenario edge case

Semua file memakai kolom resmi aplikasi. Yang sulit adalah isinya, bukan
formatnya, sehingga cocok untuk menunjukkan bagaimana Verifikasi bekerja.

| Folder | Penjualan | Penerimaan | Alokasi otomatis | Temuan |
| --- | --- | --- | --- | --- |
| `01-selisih-fee` | 3 | 3 | 0 | 6 |
| `02-bayar-sebagian` | 2 | 3 | 0 | 5 |
| `03-tunai-dan-belum-bayar` | 4 | 0 | 0 | 2 |
| `04-nominal-kembar` | 2 | 2 | 0 | 4 |
| `05-non-operasional` | 1 | 3 | 1 | 0 |
| `06-format-berantakan` | 3 | 3 | 2 | 2 |

Angka di atas adalah hasil uji nyata di `temukas.madgo.my.id`, bukan perkiraan.

## 01-selisih-fee

QRIS Rp150.000 cair Rp143.500, marketplace Rp200.000 cair Rp190.000, QRIS
Rp90.000 cair Rp83.500, plus biaya admin Rp2.500. Mesin pencocokan menuntut
nominal identik untuk pasangan otomatis, jadi ketiga pasangan masuk Verifikasi.
Tiga temuan membawa `fee_hint` berisi selisih dan sumbernya, sehingga pengguna
tinggal menekan konfirmasi.

## 02-bayar-sebagian

INV-311 senilai Rp500.000 dibayar Rp300.000 lalu Rp200.000. INV-312 dibayar
penuh tetapi keterangan mutasi tidak memuat referensi. Keduanya menunggu
keputusan manual karena nominal tiap baris tidak sama dengan nilai penjualan.

## 03-tunai-dan-belum-bayar

Dua penjualan tunai langsung lunas tanpa mutasi bank. Dua penjualan transfer dan
QRIS belum dibayar, jadi muncul sebagai dua temuan.

## 04-nominal-kembar

Dua penjualan Rp95.000 dan dua penerimaan Rp95.000 tanpa referensi. Tidak ada
yang dicocokkan otomatis karena tanpa referensi tidak ada bukti pembeda;
keempat baris menunggu Verifikasi.

## 05-non-operasional

Satu penjualan cocok otomatis. Bunga tabungan, transfer antar rekening sendiri,
dan biaya admin diabaikan sebagai penerimaan non-operasional.

## 06-format-berantakan

Kolom resmi, tetapi header di baris 8, tanggal `dd/mm/yyyy`, nominal `Rp 1.250.000`,
baris TOTAL di akhir, dan dua sheet. Hasilnya 3 penjualan dan 3 penerimaan
terbaca, 2 pasangan cocok otomatis, dan pasangan QRIS Rp875.000 yang cair
Rp868.500 menunggu Verifikasi.
