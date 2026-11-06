# File uji error impor

Semua file di folder ini sengaja salah. Gunanya untuk memastikan pesan error
yang muncul di aplikasi memang sesuai penyebabnya, bukan untuk demo produk.
Buat ulang dengan `python tools/generate_error_cases.py`.

| File | Yang diuji | Hasil di API | Pesan yang muncul |
| --- | --- | --- | --- |
| `01-teks-biasa.txt` | teks biasa | 400 | Header tidak ditemukan dalam 30 baris pertama. |
| `02-pdf-palsu.csv` | isi PDF | 400 | PDF belum didukung. Ekspor sebagai CSV atau XLSX. |
| `03-xls-lama.xls` | Excel lama | 400 | Format .xls lama belum didukung. Simpan sebagai .xlsx atau CSV. |
| `04-png-palsu.xlsx` | isi gambar | 400 | Header tidak ditemukan dalam 30 baris pertama. |
| `05-kosong.csv` | file 0 byte | 400 | File kosong. |
| `06-hanya-header.csv` | tanpa baris data | 200 lalu 400 saat simpan | 0 baris siap, lalu CSV belum memiliki baris transaksi. |
| `07-header-duplikat.csv` | dua kolom nominal | 400 | Header duplikat untuk kolom amount. Sisakan satu kolom. |
| `08-tanpa-kolom.csv` | kolom acak | 400 | Header tidak ditemukan dalam 30 baris pertama. |
| `09-nominal-rusak.csv` | nominal teks | 400 | Baris 2: nominal tidak valid. |
| `10-tanggal-rusak.csv` | tanggal teks | 400 | Baris 2: tanggal "kemarin" tidak dikenali |
| `11-baris-501.csv` | 501 baris | 400 | Satu impor maksimal 500 baris. |
| `12-besar-1mb.csv` | lebih dari 1 MB | 400 | CSV maksimal 1 MB dan 500 baris. |
| `13-zip-palsu.xlsx` | zip biasa | 400 | File Excel tidak bisa dibaca. Periksa apakah file terenkripsi atau rusak. |
| `14-kutip-rusak.csv` | kutip tidak seimbang | 400 | CSV tidak valid. Periksa tanda kutip dan jumlah kolom. |
| `15-nominal-negatif.csv` | nominal negatif | 400 | Baris 2: nominal penjualan harus lebih dari nol. |
| `16-nominal-nol.csv` | nominal nol | 400 | Baris 2: nominal penjualan harus lebih dari nol. |
| `17-mutasi-tanpa-kredit.csv` | mutasi tanpa kolom kredit | 400 | Header tidak ditemukan dalam 30 baris pertama. |

`12-besar-1mb.csv` tidak di-commit karena ukurannya; skrip generator akan
membuatnya lagi. Rincian lengkap ada di `docs/IMPORT_ERROR_MATRIX.md`.
