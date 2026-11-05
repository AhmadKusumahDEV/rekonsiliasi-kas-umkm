# Matriks Uji Impor EFA

Dokumen ini adalah daftar kasus yang akan diuji untuk alur impor, dari happy
case sampai edge case. Frontend live sudah memakai dua slot dan halaman
Verifikasi, jadi pengujian mencakup backend dan browser.

Legenda: P0 wajib demo, P1 wajib sebelum merge, P2 setelah demo.
Jenis: U unit, I integrasi DB/HTTP, E E2E browser ke API nyata.

## A. File dan format

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| F01 | CSV koma standar | Terbaca, preview sesuai | P0 | I |
| F02 | CSV titik koma | Delimiter terdeteksi | P0 | U+I |
| F03 | CSV tab | Delimiter terdeteksi | P1 | U |
| F04 | BOM UTF-8 | Header tetap dikenali | P0 | U |
| F05 | CRLF Windows | Baris tidak rusak | P0 | U |
| F06 | Sel berisi koma dalam tanda kutip | Satu sel utuh | P1 | U |
| F07 | Info rekening di atas header | Header ditemukan dalam 30 baris | P0 | U |
| F08 | File hanya header | 200, count 0, tombol simpan nonaktif | P0 | I+E |
| F09 | File kosong | 400 dengan pesan jelas | P1 | I |
| F10 | Lebih dari 500 baris | Ditolak dengan batas | P1 | I |
| F11 | Lebih dari 1 MB | Ditolak sebelum parsing | P1 | I |
| F12 | XLSX diunggah | Ditolak dengan panduan CSV | P0 | E |
| F13 | PDF atau gambar | Ditolak | P0 | E |
| F14 | Encoding Windows-1252 | Ditolak atau dikonversi dengan jelas | P2 | U |
| F15 | Ekstensi palsu, isi CSV | Dibaca dari isi, bukan nama | P1 | I |

## B. Header dan kolom

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| H01 | Header kanonik 7 kolom | Terbaca | P0 | I |
| H02 | Alias Tgl, Ket, Jumlah | Terpetakan otomatis | P0 | U |
| H03 | Kolom tambahan | Diabaikan, tidak error | P0 | U |
| H04 | Kolom wajib hilang | 400 menyebut kolom yang hilang | P0 | I+E |
| H05 | Header duplikat | Ditolak | P1 | U |
| H06 | Header huruf besar dan spasi | Dinormalkan | P0 | U |
| H07 | Header di baris ke-20 | Ditemukan | P0 | U |
| H08 | Header setelah 30 baris | Ditolak dengan pesan | P2 | U |
| H09 | Dua file berbeda header | Tidak saling memengaruhi | P1 | I |

## C. Nilai nominal

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| V01 | Rp1.250.000 | 1250000 | P0 | U |
| V02 | 1.250.000,00 | 1250000 | P0 | U |
| V03 | 1,250.00 | 1250 | P1 | U |
| V04 | 1.234,56 | 1235 | P1 | U |
| V05 | (500.000) | Negatif dan masuk tinjauan | P0 | U |
| V06 | Kurung tidak seimbang | Ditolak | P1 | U |
| V07 | Nol | Ditolak untuk penjualan | P0 | U |
| V08 | Nominal negatif | Ditolak atau refund, bukan penerimaan | P0 | U |
| V09 | Angka ilmiah | Diterima bila aman | P2 | U |
| V10 | Nominal lebih dari 1 triliun | Ditolak | P1 | U |

## D. Tanggal

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| D01 | 2026-11-05 | Diterima | P0 | U |
| D02 | 05/11/2026 | Diterima | P0 | U |
| D03 | 5/11/2026 | Diterima | P1 | U |
| D04 | 31/02/2026 | Ditolak dengan nomor baris | P0 | U |
| D05 | 03/04/2026 ambigu | Ditolak atau dikonfirmasi | P1 | U |
| D06 | Tanggal masa depan | Diizinkan dengan peringatan | P2 | U |
| D07 | Tanggal kosong | Ditolak | P0 | U |

## E. Kanal dan kategori

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| C01 | transfer, TF, TRF | Menjadi transfer | P0 | U |
| C02 | qris, QR | Menjadi qris | P0 | U |
| C03 | tunai, Cash, kasir | Tunai lunas, tidak masuk antrean | P0 | I |
| C04 | marketplace, Shopee | Menjadi marketplace | P0 | U |
| C05 | e-wallet, GoPay, OVO | Menjadi e-wallet | P1 | U |
| C06 | kanal kosong | Disimpulkan dari keterangan | P0 | U |
| C07 | BUNGA TABUNGAN | Non-operasional, tidak masuk antrean | P0 | I |
| C08 | BIAYA ADMIN | Non-operasional | P0 | I |
| C09 | TRANSFER ANTAR REKENING | Non-operasional | P0 | I |
| C10 | Nama pelanggan Bunga Pertiwi | Tetap operasional | P1 | U |

## F. Pencocokan dan biaya

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| M01 | Referensi, nominal, kanal, tanggal sama | Otomatis, skor 1,00 | P0 | I |
| M02 | Tanggal beda 1 hari | Otomatis bila unik | P0 | I |
| M03 | Nama penerima beda | Masuk AI, bukan aturan | P0 | I |
| M04 | Fee QRIS 6.500 | Fee hint bank, tetap konfirmasi | P0 | I |
| M05 | Fee marketplace 10.000 | Fee hint marketplace | P0 | I |
| M06 | Dua penjualan nominal sama | Tidak otomatis, masuk tinjauan | P0 | I |
| M07 | Dua penerimaan nominal sama | Tidak otomatis | P0 | I |
| M08 | Referensi kosong | Masuk tinjauan | P0 | I |
| M09 | Referensi salah ketik | Tidak otomatis | P1 | I |
| M10 | Tanggal beda lebih dari 7 hari | Tidak otomatis | P1 | I |
| M11 | Penjualan tanpa penerimaan | Sisa tagihan, bukan selisih | P0 | I+E |
| M12 | Impor file sama dua kali | Idempoten, tidak menggandakan | P0 | I |
| M13 | Impor bulan berbeda | Tidak salah pasang | P1 | I |
| M14 | Kandidat skor tinggi dan rendah | Skor tinggi memblokir yang lemah | P0 | I |
| M15 | Pasangan dari impor lama | Tidak boleh diubah impor baru | P1 | I |

## G. AI

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| A01 | Dua kandidat unik skor 0,90 | AI memanggil satu batch, 2 verdict | P0 | I |
| A02 | ai_status ok | Tampil “dicek AI” | P0 | I+E |
| A03 | Tidak ada kandidat | ai_status no_candidates | P0 | I |
| A04 | AI dimatikan | ai_status disabled, impor tetap sukses | P0 | I |
| A05 | AI timeout | ai_status failed, hasil aturan tetap ada | P0 | I |
| A06 | AI 401 atau 429 | failed, tidak menggagalkan impor | P1 | I |
| A07 | Respons JSON rusak | failed, fallback aturan | P1 | I |
| A08 | AI mengembalikan ID asing | Ditolak, tidak ada alokasi | P0 | I |
| A09 | Verdict duplikat konflik | Ditolak atau fail-closed | P1 | U |
| A10 | Verdict match=false | Tidak ada alokasi | P0 | I |
| A11 | Alasan berisi NUL atau newline | Dibersihkan sebelum disimpan | P1 | U |
| A12 | Alasan lebih dari 120 karakter | Dipotong per rune | P1 | U |
| A13 | AI menyentuh impor lama | Diblokir oleh scope import | P0 | I |
| A14 | ai_status failed di UI | Copy “tinjau manual”, bukan error | P0 | E |

## H. Workspace, auth, dan keamanan

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| S01 | Kosongkan workspace | Transaksi hilang, akun tetap | P0 | I+E |
| S02 | Kosongkan tanpa sesi | 401 | P0 | I |
| S03 | Kosongkan tanpa CSRF | 403 | P0 | I |
| S04 | Kosongkan sebagai non-owner | 403 | P1 | I |
| S05 | Akses workspace lain | Ditolak | P0 | I |
| S06 | Impor dengan formula =SUM | Diekspor sebagai teks aman | P1 | I |
| S07 | File sangat besar | Ditolak sebelum memori habis | P1 | I |
| S08 | Rate limit forgot/reset | 429 | P2 | I |

## I. Frontend E2E

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| E01 | Impor dua slot terisi | Sukses, ringkasan tampil | P0 | E |
| E02 | Hanya penjualan | Sukses, status belum lengkap | P0 | E |
| E03 | Hanya mutasi | Sukses, penjualan menunggu | P0 | E |
| E04 | Warnings file | Tampil sebelum simpan | P0 | E |
| E05 | Ringkasan auto dan AI | Angka terpisah dan benar | P0 | E |
| E06 | ai_status failed | Copy fallback, bukan error | P0 | E |
| E07 | Verifikasi pasangan | Setujui tanpa pindah halaman | P0 | E |
| E08 | Fee hint | Amount dan fee terisi, tetap konfirmasi | P0 | E |
| E09 | Tunai dan non-operasional | Tidak muncul di antrean | P0 | E |
| E10 | Mobile 375 px | Dua slot dan Verifikasi terbaca | P0 | E |
| E11 | PWA/offline | Tidak mengklaim data kosong saat offline | P1 | E |

## J. Batas dan performa

| ID | Kasus | Harapan | Prioritas | Jenis |
| --- | --- | --- | --- | --- |
| P01 | Tepat 500 baris | Diterima | P1 | I |
| P02 | 501 baris | Ditolak | P1 | I |
| P03 | 90 baris | Cepat dan benar | P0 | I |
| P04 | 40.000 kombinasi | Guard bekerja, tidak menggantung | P1 | I |
| P05 | Impor bersamaan | Lock bekerja, tidak ganda | P1 | I |

## Urutan eksekusi

1. Unit: nilai, tanggal, kanal, header, AI config, sanitasi.
2. Integrasi: file valid, duplikat, cash/non-op, auto-match, clear data.
3. AI: fake reviewer happy/fail, lalu Gemini live satu kali.
4. E2E frontend: dua slot, Verifikasi, fee hint, status AI.
5. Batas dan performa.

