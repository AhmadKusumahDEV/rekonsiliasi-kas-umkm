# Kesesuaian Implementasi dengan Proposal

Sumber: Solution Brief & Mini Proposal - Rekonsiliasi Kas UMKM (Evan William),
dibaca 4 November 2026.

## Kesimpulan

Implementasi inti tidak salah total. Yang menyimpang adalah penempatan AI:
proposal menyatakan AI, OCR, dan kredensial bank tidak dipakai pada MVP.
Pencocokan MVP sepenuhnya deterministik, dengan manusia memutuskan kasus
biaya, sebagian, gabungan, dan referensi kosong.

## Yang sudah sesuai

| Ketentuan proposal | Status di repo |
| --- | --- |
| Impor dua CSV dengan template | Ada, ditambah impor fleksibel |
| Pratinjau tanpa mengubah saldo | Ada |
| Validasi tanggal, nominal, duplikat | Ada |
| Usulan dari referensi, kanal, tanggal, nominal | Ada |
| Otomatis hanya untuk pasangan unik 1:1 | Ada, dengan ambang 0,95 dan cek unik dua sisi |
| Kasus biaya, sebagian, gabungan ditahan | Ada di antrean verifikasi |
| Alasan usulan dan sisa tagihan | Ada di API |
| Audit dan pembatalan | Ada |
| Ekspor CSV menetralkan formula | Ada |
| Batas 500 baris per file | Ada |
| Tanpa kredensial bank dan OCR | Ada |

## Yang menyimpang

| Penyimpangan | Dampak | Keputusan |
| --- | --- | --- |
| AI dan Gemini direncanakan untuk MVP | Bertentangan dengan proposal dan biaya Rp0 | Keluarkan dari demo, jadikan post-MVP |
| Dokumen AI listwise dan gateway | Membuat fokus tim pecah | Tandai sebagai backlog |
| Stack Go + frontend statis | Proposal menulis Node/TS + React | Pertahankan, catat sebagai deviasi yang disetujui |
| Impor fleksibel dan auto-match berbobot | Melampaui satu template CSV | Pertahankan sebagai peningkatan, tetap dalam batas unik 1:1 |
| Belum ada studi waktu dan 10 bukti uji | Proposal mensyaratkan | Tambahkan sebelum demo |

## Posisi AI yang benar

1. MVP: tidak ada AI. Pencocokan memakai aturan dan skor transparan.
2. Post-MVP: AI hanya untuk kandidat ambigu, dengan kandidat tertutup dan
   keputusan tetap di verifikasi.
3. Kode Gemini boleh tetap ada, tetapi tidak dipanggil dan tidak masuk demo.

## Pemetaan 10 test case proposal

| ID | Status | Catatan |
| --- | --- | --- |
| TC01 unggah dua CSV | Sudah | Demo scenario test |
| TC02 header/tanggal/nominal salah | Sebagian | Perlu test pesan per baris |
| TC03 impor ulang file sama | Sudah | Uji duplikat ada |
| TC04 pasangan unik dua arah | Sudah | Suggestion dan auto-match |
| TC05 dua transaksi sama | Sudah | Demo edge case |
| TC06 biaya, sebagian, gabungan | Sebagian | Fee hint dan alokasi ada; gabungan belum |
| TC07 dua tab serentak | Belum | Perlu test konkurensi |
| TC08 batalkan pasangan | Sudah | Undo allocation |
| TC09 akses usaha lain atau sesi habis | Sebagian | Perlu test eksplisit |
| TC10 timeout dan formula CSV | Sebagian | Formula export sudah; timeout belum |

## Tindakan

1. Keluarkan AI dari demo dan dari kriteria selesai MVP.
2. Tambahkan studi waktu dan bukti 10 test case.
3. Lengkapi test TC02, TC06 gabungan, TC07, TC09, TC10.
4. Catat deviasi stack di dokumen proyek.
5. Baru evaluasi AI setelah pilot dan data koreksi cukup.

