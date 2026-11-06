# Kolom paling minimal

Dua file ini memakai kolom sesedikit mungkin yang masih diterima aplikasi:

- `penjualan-minim.csv` — `tanggal,nama,nominal` (33 baris, total Rp6.022.500)
- `mutasi-minim.csv` — `tanggal,keterangan,nominal` (33 baris, total Rp6.022.500)

Hasil uji di `temukas.madgo.my.id`: keduanya terbaca 33 baris tanpa peringatan,
tetapi menghasilkan 0 pasangan otomatis dan 66 temuan. Penyebabnya bukan format,
melainkan kolom `referensi` yang tidak ada. Pencocokan otomatis mensyaratkan
referensi terisi, jadi semua baris menunggu keputusan manual di Verifikasi.

Kolom `kanal` juga tidak ada, sehingga server mengisinya dengan `lainnya`.
Kalau ingin file seminimal mungkin yang masih cocok otomatis, tambahkan satu
kolom `referensi` berisi nilai sama di penjualan dan mutasi. Empat kolom itu
batas praktisnya.

Berguna untuk menguji bahwa aplikasi tidak rewel soal kolom, dan untuk
menunjukkan ke pengguna bahwa referensi adalah kunci pencocokan otomatis.
