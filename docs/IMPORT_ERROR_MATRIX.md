# Peta error impor

Dokumen ini menjawab satu pertanyaan: kondisi apa yang membuat impor gagal,
pesan apa yang muncul, dan apakah pesan itu datang dari browser atau dari server.

Alurnya dua langkah. Pertama "Periksa file": browser mengirim isi file apa adanya
ke `POST /api/v1/umkm/imports/files`, server membaca CSV atau XLSX lalu
mengembalikan CSV kanonik untuk pratinjau. Kedua "Simpan": browser mengirim CSV
kanonik itu ke `POST /api/v1/umkm/imports`. Kesalahan tipe dan isi hampir selalu
muncul di langkah pertama.

Penanda sumber: **uji** berarti saya menguji langsung ke `efa.madgo.my.id` atau
API produksi, **kode** berarti terbaca dari sumber tetapi tidak saya picu manual.

## 1. Ditolak di browser, request tidak dikirim

| Pemicu | Pesan | Sumber |
| --- | --- | --- |
| Tombol Periksa ditekan tanpa file | Pilih minimal satu file CSV atau XLSX. | kode |
| File 0 byte | File kosong. Pilih CSV atau XLSX yang berisi data. | kode |
| Ekstensi bukan `.csv` atau `.xlsx` | Gunakan CSV atau XLSX. File XLS lama, PDF, dan gambar belum didukung. | uji |
| File lebih dari 10 MB | File melebihi 10 MB. Pecah workbook atau kurangi isinya, lalu unggah kembali. | uji |
| Simpan ditekan sebelum pratinjau selesai | Periksa semua file yang dipilih sebelum menyimpan. | kode |
| Hasil konversi lebih dari 500 baris atau 1 MB | Hasil konversi melebihi batas server. Kurangi baris atau pilih sheet yang lebih kecil. | kode |

Batas 10 MB dicek di browser, jadi pengguna normal tidak akan pernah melihat
halaman 413 dari nginx. Halaman itu hanya muncul bila request dikirim langsung
ke API tanpa lewat aplikasi.

## 2. Ditolak server saat Periksa file

Semua pesan di bawah tampil di panel merah pada halaman impor dan diakhiri
kalimat baku: "Periksa format, header, dan baris yang disebutkan; simpan ulang
file jika rusak atau dilindungi sandi."

| Pemicu | Status | Pesan server | Sumber |
| --- | --- | --- | --- |
| Isi file bukan tabel | 400 | Header tidak ditemukan dalam 30 baris pertama. Gunakan kolom tanggal, nominal, dan nama atau keterangan. | uji |
| PDF, walau ekstensinya CSV | 400 | PDF belum didukung. Ekspor sebagai CSV atau XLSX. | uji |
| File `.xls` lama | 400 | Format .xls lama belum didukung. Simpan sebagai .xlsx atau CSV. | uji |
| Gambar atau teks dengan ekstensi CSV/XLSX | 400 | Header tidak ditemukan dalam 30 baris pertama. | uji |
| File kosong | 400 | File kosong. | uji |
| Dua kolom bernama sama | 400 | Header duplikat untuk kolom amount. Sisakan satu kolom. | uji |
| Kolom tanggal, nominal, atau nama tidak ada | 400 | Header tidak ditemukan dalam 30 baris pertama. | uji |
| Nominal bukan angka | 400 | Baris 2: nominal tidak valid. | uji |
| Nominal nol atau negatif | 400 | Baris 2: nominal penjualan harus lebih dari nol. | uji |
| Tanggal tidak dikenali | 400 | Baris 2: tanggal "kemarin" tidak dikenali | uji |
| Lebih dari 500 baris | 400 | Satu impor maksimal 500 baris. | uji |
| CSV lebih dari 1 MB | 400 | CSV maksimal 1 MB dan 500 baris. | uji |
| Zip atau XLSX rusak | 400 | File Excel tidak bisa dibaca. Periksa apakah file terenkripsi atau rusak. | uji |
| Kutip tidak seimbang | 400 | CSV tidak valid. Periksa tanda kutip dan jumlah kolom. | uji |
| Workbook dua sheet tanpa memilih sheet | 422 | Pilih sheet yang ingin diimpor. Disertai daftar sheet, misalnya Mutasi dan Rekap. | uji |
| Nama sheet tidak ada | 400 | Sheet tidak ditemukan: Hantu | uji |
| Workbook lebih dari 20 sheet | 400 | Workbook maksimal 20 sheet. | kode |
| Sheet hanya berisi header | 400 | Sheet kosong. | kode |
| File yang hanya berisi header | 200 | Tidak ada error. Pratinjau menulis 0 baris siap dan tombol simpan tidak muncul. | uji |
| Tanpa sesi | 401 | Sesi berakhir. Masuk kembali untuk melanjutkan. | uji |
| Body lebih dari 12 MB | 413 | Ukuran permintaan terlalu besar. | kode |

## 3. Gagal saat Simpan

| Pemicu | Status | Pesan | Sumber |
| --- | --- | --- | --- |
| CSV kanonik kosong atau tidak lengkap | 400 | CSV kosong atau header tidak valid. | uji |
| Pratinjau berisi header saja | 400 | CSV belum memiliki baris transaksi. | uji |
| Jenis bukan penjualan atau penerimaan | 400 | Data permintaan tidak valid: kind tidak valid. | uji |
| Header CSRF hilang atau salah | 403 | Sesi formulir berubah. Muat ulang halaman sebelum menyimpan. | uji |
| Batch yang sama diimpor ulang | 201 | Bukan error. Balasan berisi duplicate true, imported 0, skipped 24. | uji |
| Batch pernah dibatalkan | 409 | Impor ini sudah dibatalkan. Gunakan ID sumber baru jika ingin mencatat ulang. | kode |
| ID sumber sama dengan data berbeda | 409 | ID sumber sudah ada dengan data berbeda atau telah dibatalkan. Impor tidak disimpan. | kode |
| Melebihi 10.000 transaksi | 400 | Ruang kerja mencapai batas MVP 10.000 transaksi. Ekspor dan hubungi pengelola. | kode |
| AI gagal, timeout, atau kuota habis | 201 | Bukan error. ai_status berisi failed, dan panel menulis Pemeriksaan AI gagal. Hasil aturan tetap tersimpan; tinjau temuan manual. | uji |
| Server error tak terduga | 500 | Terjadi kesalahan pada server. | kode |

## 4. Kalau file dipaksa masuk

Atribut `accept` pada kedua input hanya berisi `.csv` dan `.xlsx`. Nilai itu
petunjuk picker, bukan penjaga. Drag-drop atau mengganti ekstensi tetap lolos
ke validasi berikutnya.

| Paksaan | Yang terjadi |
| --- | --- |
| PDF diganti ekstensi `.csv` | Server melihat magic bytes PDF, pesan PDF belum didukung. |
| XLS lama diganti `.xlsx` | Server melihat header OLE, pesan Format .xls lama belum didukung. |
| PNG diganti `.xlsx` atau `.csv` | Tidak ada magic bytes yang dikenali, server menganggapnya CSV, pesan Header tidak ditemukan. |
| Teks diganti `.csv` | Sama, pesan Header tidak ditemukan dalam 30 baris pertama. |
| CSV asli diganti `.xlsx` | Server tidak melihat awalan zip, lalu membacanya sebagai CSV. Tetap berhasil. |
| XLSX asli diganti `.csv` | Server melihat awalan zip, lalu membacanya sebagai XLSX. Tetap berhasil. |

Jadi tipe file benar-benar dijaga di server lewat isi file, bukan lewat nama.
Yang belum rapi: pengguna yang mengganti ekstensi tetap mendapat pesan yang
menyesatkan, karena isi gambar dilaporkan sebagai header tidak ditemukan.

## 5. Yang perlu diwaspadai saat demo

Impor duplikat terlihat seperti sukses. Balasannya 201 dengan `duplicate: true`,
dan panel hanya menulis 0 baris tersimpan, 24 dilewati, lalu Sudah diproses.
Tidak ada label bahwa file itu sudah pernah diimpor.

File berisi header saja tidak memunculkan error. Pratinjau menulis 0 baris siap
dan tombol simpan hilang, jadi pengguna tidak tahu apakah file salah atau memang
kosong.

Kalimat penutup error 400 terlalu umum. Untuk header duplikat atau 501 baris,
aplikasi tetap menyarankan simpan ulang file jika rusak atau dilindungi sandi,
padahal penyebabnya bukan file rusak.

Nama kolom di pesan duplikat memakai istilah internal, misalnya amount, bukan
Nominal.

AI gagal bukan error impor. Data tetap tersimpan dengan ai_status failed, dan
pasangan yang seharusnya diputuskan AI akan muncul sebagai temuan.

Sesi habis bisa tanpa pesan. Bila 401 datang dari pengecekan latar belakang,
halaman langsung kembali ke login tanpa toast, dan pilihan file hilang.

Dua slot saling mengunci. Jika satu file gagal, file lain yang valid tidak bisa
disimpan sampai slot bermasalah diganti atau halaman dimuat ulang.

## 6. Cara mengulang pengujian

`python tools/generate_error_cases.py` membuat ulang seluruh file di
`docs/demo-files/error-cases/`. `pwsh tools/probe_import_errors.ps1` mendaftarkan
akun sementara, mengunggah semua file itu, lalu mencetak tabel status dan pesan.
Untuk melihat perilakunya di aplikasi, unggah file yang sama lewat
`https://efa.madgo.my.id/app#imports`.
