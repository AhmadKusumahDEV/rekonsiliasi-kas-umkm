# Handoff Kosongkan Data Workspace

Fitur ini mengosongkan ledger UMKM tanpa menghapus akun atau workspace. Ini
dipilih untuk scope PBL agar pengguna dapat memulai data baru tanpa kehilangan
identitas login.

Status: aktif di production di `https://temukas.madgo.my.id` sejak 3 November
2026.

## Kontrak backend

```http
DELETE /api/v1/umkm/data
Content-Type: application/json
X-CSRF-Token: ...
Cookie: rku_session=...
```

```json
{
  "password": "kata-sandi-saat-ini",
  "confirm": "HAPUS"
}
```

Sukses `200 {"ok":true}`. Password harus benar dan frasa konfirmasi harus tepat
`HAPUS`. Endpoint hanya untuk owner.

Contoh pemanggilan:

```sh
curl -i -X DELETE https://temukas.madgo.my.id/api/v1/umkm/data \
  -H 'Content-Type: application/json' \
  -H "X-CSRF-Token: $CSRF" \
  -b cookies.txt \
  -d '{"password":"kata-sandi-saat-ini","confirm":"HAPUS"}'
```

Setelah sukses, backend menaikkan versi workspace dan mengirim event SSE
bernama `state` berisi nomor versi baru. Frontend boleh memakai event itu untuk
memuat ulang state, atau langsung memanggil `GET /api/v1/umkm/state` setelah
respons `200`.

## Yang dihapus

- `umkm_allocations`
- `umkm_entries`
- `umkm_imports`
- state UMKM legacy di `pblg_workspaces.state`

## Yang tetap ada

- akun dan password
- workspace, nama, dan kode workspace
- user lain di workspace
- sesi login
- audit sebelumnya dan audit baru “Kosongkan data workspace”
- konfigurasi aplikasi

Tidak ada restore otomatis. Backup database tetap tanggung jawab operator VPS.

## UI yang perlu dibuat

Di Pengaturan, tambahkan bagian berbahaya dengan alur:

1. Tombol **Kosongkan data workspace**.
2. Dialog peringatan bahwa transaksi, batch impor, dan alokasi akan hilang.
3. Ajakan mengunduh rekap CSV terlebih dahulu.
4. Input password saat ini.
5. Input frasa `HAPUS`.
6. Tombol konfirmasi yang dinonaktifkan sampai kedua input valid.
7. Setelah sukses, tutup dialog, muat ulang state, dan arahkan ke Ringkasan.
8. Tampilkan keadaan kosong yang menjelaskan bahwa workspace masih aktif dan
   pengguna dapat mengimpor data baru.

Jangan memberi label “Hapus workspace” pada tombol ini; workspace tetap ada.

## Penanganan error

- `400`: password salah atau frasa konfirmasi tidak tepat.
- `401`: sesi berakhir; kembali ke login.
- `403`: CSRF/role tidak valid.
- `503`: database belum siap; jangan mengulang otomatis.

Pesan `400` yang mungkin muncul: `Ketik HAPUS untuk mengonfirmasi.` dan
`Kata sandi saat ini belum cocok.` Endpoint ini mengunci baris workspace di dalam
transaksi, jadi tidak mengembalikan `409` saat ada perubahan bersamaan; request
kedua menunggu lock sampai request pertama selesai.

## Pengujian backend

`TestClearWorkspaceDataKeepsAccountAndAudit` memeriksa:

- konfirmasi dan password salah ditolak;
- alokasi, entri, dan batch impor terhapus;
- akun, user, workspace, dan audit tetap ada;
- event audit baru tercatat;
- impor ulang file yang sama setelah clear berhasil, karena checksum lama ikut
  terhapus.

## Verifikasi live 3 November 2026

- `DELETE /api/v1/umkm/data` tanpa cookie → `401`.
- Frasa selain `HAPUS` → `400` dengan pesan `Ketik HAPUS untuk mengonfirmasi.`.
- Password salah → `400` dengan pesan `Kata sandi saat ini belum cocok.`.
- Preflight `OPTIONS` dari `https://efa.madgo.my.id` → `204`, origin diizinkan,
  method `DELETE` dan header `X-CSRF-Token` masuk daftar izin.
- Login akun demo → `200`, `role` = `owner`; `GET /api/v1/umkm/state` → `200`.
