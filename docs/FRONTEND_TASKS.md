# Tugas Frontend: Yang Harus Dibaca dan Dibangun

Status pengerjaan frontend dan bukti verifikasi tersedia di [implementasi handoff](FRONTEND_HANDOFF_IMPLEMENTATION.md).

Dokumen ini adalah pintu masuk untuk frontend. Backend sudah tersedia;
beberapa layar belum ada.

## Urutan baca

1. `FRONTEND_VERIFICATION_CONTRACT.md` - kontrak API, field, dan alur layar.
2. `JENIS_IMPOR.md` - dua slot impor, deteksi jenis file, dan format bank.
3. `FORMAT_IMPOR_V2.md` - template penjualan dan bentuk mutasi bank.
4. `KEPUTUSAN_IMPOR.md` - keputusan scope: dua slot, tanpa deteksi otomatis.
5. `DEMO_READINESS_SCOPE.md` - kriteria demo dan data uji.
6. `PASSWORD_RESET.md` - route `#forgot` dan `#reset`.
7. `CLEAR_WORKSPACE_DATA.md` - danger zone di Pengaturan.
8. `MVP_ALIGNMENT_PROPOSAL.md` - batas MVP; AI bukan bagian MVP.

Dokumen riset yang tidak wajib dibaca frontend: `AI_IMPORT_AUTOMATION.md`,
`MATCHING_ENGINE_RESEARCH.md`, `CONTAINER_MATCHING.md`,
`AUDIT_IMPORT_AI_2026-11-04.md`, `PREDIKSI_UPLOAD.md`.

## Yang harus dibangun

### 1. Layar impor dua slot

- Slot Penjualan dan slot Mutasi bank dalam satu halaman.
- Setiap slot memanggil `POST /api/v1/umkm/imports/flexible`.
- Tampilkan `count`, `total`, `skipped`, `warnings`, dan preview.
- Commit memakai `response.csv`, bukan CSV mentah.
- Commit penjualan lebih dulu, lalu penerimaan.
- Tampilkan `imported`, `auto_matched`, `ai_matched`.
- Tombol unduh template penjualan.

Layar ini belum ada. Modal impor sekarang hanya satu file dan satu jenis.

### 2. Halaman Verifikasi

- Rute `#verifikasi`, menggantikan `#reconcile` dan `#exceptions`.
- Sumber data `GET /api/v1/umkm/exceptions` dan `/suggestions`.
- Gabungkan dua item exception yang saling menunjuk menjadi satu kartu.
- Tampilkan `reason`, `remaining`, `fee_hint`, dan kategori.
- Aksi setujui memakai `POST /api/v1/umkm/allocations`.
- Untuk `fee_hint`, isi `amount` dan `fee` otomatis, pengguna tetap setuju.
- Penjualan tunai dan penerimaan non-operasional jangan diberi tombol alokasi.
- Sediakan filter: siap disetujui, ambigu, tanpa kandidat, sisa sebagian.

### 3. Lupa dan reset kata sandi

- Route `#forgot` dan `#reset?token=...`.
- Token diambil dari `location.hash`, bukan query string.
- Jangan simpan token di `localStorage`.
- Setelah reset, hapus token dari URL dan arahkan ke `#login`.

### 4. Kosongkan data workspace

- Danger zone di Pengaturan.
- Input password dan frasa `HAPUS`.
- Jelaskan bahwa akun, workspace, dan audit tetap ada.
- Setelah sukses, muat ulang state dan arahkan ke Ringkasan.

### 5. Checklist verifikasi dua fitur di atas

Backend reset password dan kosongkan workspace sudah aktif. Frontend harus
memeriksa dan mengimplementasikan bila belum ada.

- [ ] `#forgot` dan `#reset` bisa dibuka dan tidak 404.
- [ ] Form email mengirim request dan menampilkan pesan generik.
- [ ] Token diambil dari hash, lalu dibersihkan dari URL.
- [ ] Password baru bisa dipakai login.
- [ ] Danger zone ada di Pengaturan.
- [ ] Tombol kosongkan meminta password dan frasa `HAPUS`.
- [ ] Setelah kosongkan, transaksi dan impor hilang, akun tetap bisa login.
- [ ] Audit “Kosongkan data workspace” muncul.
- [ ] Tombol dinonaktifkan saat request gagal atau sesi habis.



### 6. Penyesuaian navigasi dan PWA

- Ganti label Cocokkan kas menjadi Verifikasi.
- Alias `#reconcile` dan `#exceptions` diarahkan ke `#verifikasi`.
- Perbarui shortcut PWA.
- Naikkan versi aset di `index.html` setelah perubahan JS/CSS.

## Yang tidak perlu dibangun sekarang

- Panggilan langsung ke Gemini dari frontend.
- Unggah XLSX atau PDF.
- OCR struk.
- Bank rules buatan pengguna.
- Blocking atau bipartite matching.
- Multi-mata uang.

## Tampilan status AI

Respons impor memuat ai_status. Gunakan pemetaan berikut:

| ai_status | Copy yang ditampilkan |
| --- | --- |
| ok | “N pasangan dicek AI dan disimpan otomatis” |
| no_candidates | “Tidak ada pasangan ambigu untuk AI” |
| disabled | “Pemeriksaan AI tidak aktif; pencocokan memakai aturan” |
| skipped | “Pemeriksaan AI dilewati; pencocokan memakai aturan” |
| failed | “Pemeriksaan AI gagal. Hasil aturan tetap tersimpan; tinjau temuan manual” |

Aturan tampilan:

- Jangan menulis “AI mencocokkan semua transaksi”.
- Pisahkan auto_matched (aturan) dan ai_matched (AI).
- Di Verifikasi, sediakan bagian “Sudah beres otomatis” dengan label
  “Cocok otomatis” atau “Dicek AI”, alasan, dan tombol batalkan.
- Jangan memanggil Gemini dari frontend.
- Jangan menampilkan skor sebagai persentase keyakinan.


## Ringkasan API

| Endpoint | Kegunaan |
| --- | --- |
| `POST /umkm/imports/flexible` | Pratinjau file longgar, balikan CSV kanonik |
| `POST /umkm/imports` | Simpan CSV kanonik |
| `GET /umkm/state` | Entri, import, alokasi, audit |
| `GET /umkm/exceptions` | Antrean verifikasi per entri |
| `GET /umkm/suggestions` | Pasangan yang bisa disetujui |
| `POST /umkm/allocations` | Setujui atau alokasi manual |
| `POST /auth/password/forgot` | Kirim tautan reset |
| `POST /auth/password/reset` | Simpan kata sandi baru |
| `DELETE /umkm/data` | Kosongkan data workspace |

## Kriteria demo frontend

- Dua slot bisa diisi, salah satu boleh kosong.
- Warnings tampil sebelum simpan.
- Hasil impor menyebut jumlah baris, otomatis, dan AI.
- Verifikasi hanya menampilkan yang perlu keputusan.
- Setujui pasangan berhasil tanpa pindah halaman.
- Tunai dan non-operasional tidak muncul di antrean.
- Semua angka memakai `remaining` dari backend.

