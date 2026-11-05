# Rencana Upload Excel

Tujuan: menerima `.xlsx` tanpa mengubah alur pencocokan, AI, Verifikasi, atau
laporan. Excel dikonversi ke CSV kanonik yang sama seperti sekarang.

## Keputusan arsitektur

- Tambah endpoint baru, jangan ganti endpoint CSV yang stabil.
- `POST /api/v1/umkm/imports/files` menerima multipart: `file`, `kind`,
  `sheet` opsional.
- Hasilnya sama dengan endpoint fleksibel: CSV kanonik, preview, warnings.
- Commit tetap memakai `POST /api/v1/umkm/imports` dengan CSV kanonik.
- Tidak ada perubahan skema database.

```http
POST /api/v1/umkm/imports/files
Content-Type: multipart/form-data

file=@mutasi.xlsx
kind=receipt
sheet=Sheet1 (opsional)
```

```json
{
  "csv": "id,tanggal,referensi,nama,nominal,kanal,catatan\n...",
  "count": 13,
  "total": 1661000,
  "preview": [],
  "warnings": [],
  "source": {"format":"xlsx","sheet":"Mutasi","header_row":8,"sheets":["Mutasi","Rekap"]}
}
```

Kalau ada beberapa sheet berisi data dan `sheet` belum dipilih, kembalikan
`422 sheet_required` beserta daftar sheet. Frontend menampilkan pemilih sheet
lalu memanggil ulang.

## Langkah backend

1. Tambah dependensi `github.com/xuri/excelize/v2`.
2. Buat `internal/imports/xlsx.go` untuk membuka workbook, membaca sheet, dan
   mengambil nilai sel.
3. Batasi file: 10 MB terkompresi, 500 baris, 1 MB CSV hasil konversi,
   maksimal 20 sheet, dan batas dekompresi untuk mencegah zip bomb.
4. Tolak file terenkripsi, `.xls` lama, file rusak, dan PDF.
5. Abaikan makro; baca nilai sel, bukan menjalankan formula.
6. Deteksi header dalam 30 baris pertama; dukung header dua baris dan merged
   cell dengan mengambil nilai kiri-atas.
7. Gunakan normalisasi yang sudah ada: nominal, tanggal, kanal, ID, dan
   referensi.
8. Baca flag `date1904` untuk tanggal serial.
9. Tolak formula tanpa nilai cache dan sel error `#N/A`, `#REF!`, dan sejenisnya.
10. Kembalikan CSV kanonik beserta `source`, `warnings`, dan error per baris.
11. Simpan file sementara di luar webroot, hapus setelah selesai.
12. Tambah feature flag `IMPORT_XLSX_ENABLED`.

## Langkah frontend

1. Ubah input menjadi `.csv,.xlsx` dengan MIME yang sesuai.
2. Kirim `FormData`, bukan membaca file sebagai teks.
3. Kalau respons `422 sheet_required`, tampilkan pemilih sheet dan kirim ulang.
4. Tampilkan warnings dan preview seperti sekarang.
5. Commit tetap memakai `response.csv`.
6. Tampilkan progres untuk file besar.
7. Jangan mengubah halaman Verifikasi.

## Happy case yang harus lulus

| ID | Kasus | Harapan |
| --- | --- | --- |
| XH01 | XLSX penjualan 20 baris | Terbaca, preview benar |
| XH02 | XLSX mutasi bank 21 baris | Header dan kredit terbaca |
| XH03 | CSV dan XLSX dicampur | Keduanya bisa diimpor |
| XH04 | Header di baris ke-8 | Terdeteksi |
| XH05 | Dua sheet, satu dipilih | Hanya sheet terpilih dibaca |
| XH06 | Commit setelah konversi | Pencocokan dan AI berjalan sama |
| XH07 | 90 baris data | Selesai tanpa timeout |

## Edge case yang harus lulus

| ID | Kasus | Harapan |
| --- | --- | --- |
| XE01 | `.xls` lama | Ditolak dengan panduan simpan ulang |
| XE02 | File terenkripsi | Ditolak |
| XE03 | File rusak | Ditolak |
| XE04 | Zip bomb | Ditolak sebelum dekompresi penuh |
| XE05 | Lebih dari 10 MB | Ditolak |
| XE06 | Lebih dari 500 baris | Ditolak |
| XE07 | Workbook multi-sheet | Minta pilih sheet |
| XE08 | Sheet tersembunyi | Dilewati kecuali dipilih |
| XE09 | Sheet kosong | Dilewati |
| XE10 | Pivot table | Ditolak atau minta konfirmasi |
| XE11 | Merged header | Digabung dengan benar |
| XE12 | Header duplikat | Ditolak |
| XE13 | Formula tanpa cache | Baris ditolak dengan alasan |
| XE14 | Sel `#N/A` | Baris ditolak |
| XE15 | Tanggal 1904 | Dikonversi benar |
| XE16 | Nominal `Rp1.250.000` | Menjadi 1250000 |
| XE17 | Nominal `(500.000)` | Negatif, masuk tinjauan |
| XE18 | ID `00123` sebagai angka | Ditolak atau minta ubah ke teks |
| XE19 | Baris total di bawah tabel | Tidak diimpor |
| XE20 | Dua tabel satu sheet | Minta pilih tabel |
| XE21 | File bermakro `.xlsm` | Makro diabaikan, nilai dibaca |
| XE22 | Sheet dilindungi | Dibaca read-only |
| XE23 | External link | Tidak diambil, nilai cache dipakai |
| XE24 | Formula injection | Diekspor sebagai teks aman |

## Batasan implementasi v1

- Deteksi header memakai satu baris; header dua baris dan merged cell belum
  digabung otomatis.
- Tanggal serial mengikuti nilai terformat dari Excelize; belum ada penanganan
  date1904 eksplisit.
- Pivot table dan dua tabel dalam satu sheet belum dipilih otomatis.
- Formula tanpa nilai cache dan sel error bergantung pada nilai yang dibaca
  Excelize; baris yang tidak valid akan ditolak oleh validasi.
- File terenkripsi dapat muncul sebagai pesan .xls lama karena signature OLE.

Batasan ini harus ditutup atau diuji sebelum mengklaim dukungan penuh.


## Urutan pengerjaan

| Fase | Isi | Estimasi |
| --- | --- | --- |
| 1 | Endpoint multipart dan pembaca XLSX dasar | 2 hari |
| 2 | Header, merged cell, sheet picker | 2 hari |
| 3 | Nilai, tanggal 1904, formula, error sel | 2 hari |
| 4 | Keamanan dan batas file | 1 hari |
| 5 | Frontend upload dan sheet picker | 2 hari |
| 6 | Golden file, integrasi, E2E | 2 hari |

Total sekitar 9-11 hari kerja. CSV lama tetap jalan selama pengembangan.

## Kriteria selesai

- XLSX penjualan dan mutasi bisa diunggah tanpa konversi manual.
- Hasil konversi masuk ke alur commit, auto-match, dan AI yang sama.
- File berbahaya atau tidak didukung ditolak dengan pesan jelas.
- Tidak ada regresi pada CSV.
- Ada golden file XLSX di repo test.

