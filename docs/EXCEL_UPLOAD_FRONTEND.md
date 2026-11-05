# Handoff Frontend: Upload Excel

Status: backend sudah tersedia (5 November 2026). Frontend tinggal mengganti
input menjadi multipart dan menangani pemilih sheet.

Backend akan menambah endpoint multipart. Halaman Verifikasi dan alur
pencocokan tidak berubah.

## Endpoint

```http
POST /api/v1/umkm/imports/files
Content-Type: multipart/form-data

file=@penjualan.xlsx
kind=sale
sheet=Sheet1 (opsional)
```

Respons sama seperti endpoint fleksibel:

```json
{
  "csv": "id,tanggal,referensi,nama,nominal,kanal,catatan\n...",
  "count": 20,
  "total": 2500000,
  "preview": [],
  "skipped": 0,
  "warnings": [],
  "source": {"format":"xlsx","sheet":"Penjualan","header_row":8,"sheets":["Penjualan","Rekap"]}
}
```

Commit tetap:

```http
POST /api/v1/umkm/imports
{"kind":"sale","csv":"<response.csv>"}
```

## Perubahan UI
Field source berisi format (csv/xlsx), sheet terpilih, header_row, dan
daftar sheets. Jika ada lebih dari satu sheet berisi data, backend membalas 422
dengan details.sheets; kirim ulang request yang sama dengan field sheet.


1. Input menerima `.csv` dan `.xlsx`.
2. Gunakan `FormData`, bukan `FileReader.readAsText`.
3. Tampilkan nama file, ukuran, dan progres unggah.
4. Kalau respons `422`, tampilkan modal pemilih sheet dari
   `details.sheets`, lalu kirim ulang dengan `sheet`.
5. Tampilkan `warnings` sebelum tombol simpan.
6. Setelah sukses, commit memakai `response.csv`.
7. Jangan mengubah halaman Verifikasi.

## Error yang perlu ditangani
Catatan batas:

- File maksimal 10 MB; body transport sekarang 12 MB untuk memberi ruang multipart.
- Hasil konversi CSV tetap dibatasi 1 MB dan 500 baris.
- Workbook maksimal 20 sheet.
- Endpoint memerlukan cookie sesi dan X-CSRF-Token.
- 415 dan 429 belum dihasilkan endpoint ini; format tidak didukung biasanya 400.
- 503 berarti sesi atau database belum siap, bukan layanan file terpisah.


| Status | Arti | Tindakan UI |
| --- | --- | --- |
| 400 | File/header/nilai tidak valid | Tampilkan pesan dan baris |
| 413 | File terlalu besar | Minta pecah file |
| 415 | Format tidak didukung | Arahkan ke CSV/XLSX |
| 422 | Perlu pilih sheet | Buka pemilih sheet |
| 429 | Terlalu banyak unggahan | Minta coba lagi |
| 503 | Layanan file tidak siap | Tawarkan coba lagi |

## Checklist E2E

- XLSX penjualan 20 baris berhasil.
- XLSX mutasi 21 baris berhasil.
- CSV lama tetap berhasil.
- Multi-sheet memunculkan pemilih.
- File terenkripsi, `.xls`, PDF, dan file rusak ditolak.
- File lebih dari 10 MB ditolak sebelum upload selesai.
- Ringkasan `imported`, `auto_matched`, `ai_matched`, `ai_status` tampil.
- Mobile 375 px tetap bisa memilih file dan sheet.

