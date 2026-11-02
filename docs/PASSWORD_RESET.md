# Handoff Reset Kata Sandi

Fitur ini menambahkan reset kata sandi melalui email Resend. Endpoint backend
sudah tersedia; halaman frontend perlu menambahkan dua route.

## Kontrak backend

### Minta tautan

```http
POST /api/v1/auth/password/forgot
Content-Type: application/json

{ "email": "owner@example.com" }
```

Response selalu `202 {"ok":true}` untuk email valid, baik akun ditemukan maupun
tidak. Jangan tampilkan pesan yang membedakan kedua kondisi. Backend mengirim
email hanya jika akun ditemukan.

### Reset

```http
POST /api/v1/auth/password/reset
Content-Type: application/json

{
  "token": "token-dari-fragment-email",
  "password": "kata-sandi-baru-minimal-12-karakter"
}
```

Sukses `200 {"ok":true}`. Token salah, kedaluwarsa, atau sudah dipakai:
`400`. Setelah sukses, semua sesi lama dicabut; frontend harus kembali ke
halaman login.

## Route frontend yang disarankan

Tautan email berbentuk:

```text
https://efa.madgo.my.id/app#reset?token=<token>
```

Frontend perlu menambahkan:

1. `#forgot`: form email, kirim ke `/auth/password/forgot`, lalu tampilkan pesan
   generik “Jika email terdaftar, tautan reset akan dikirim.”
2. `#reset?token=...`: ambil token dari fragment URL, tampilkan form password
   baru dan konfirmasi password, kirim ke `/auth/password/reset`.
3. Setelah reset sukses, hapus token dari URL, arahkan ke `#login`, dan minta
   pengguna masuk dengan password baru.

Jangan menaruh token di `localStorage` atau mengirimkannya ke analytics/log.
Token tidak perlu disimpan setelah form dikirim.

## Environment

Backend membutuhkan:

```text
RESEND_API_KEY=...
RESEND_FROM_EMAIL=noreply@madgo.my.id
APP_PUBLIC_URL=https://efa.madgo.my.id
PASSWORD_RESET_TTL=15m
```

Jangan commit nilai `RESEND_API_KEY`. Di VPS, tambahkan ke environment container
API (file `otel.env` yang dipakai Compose) lalu rebuild container API.

## Keamanan dan batas

- Token 32 byte acak, disimpan sebagai SHA-256; token asli hanya ada di email.
- Token sekali pakai, default berlaku 15 menit, dan token lama untuk user yang
  sama dibatalkan saat permintaan baru.
- Rate limit auth berlaku untuk request forgot/reset.
- Database menyimpan `user_id`, bukan email di tabel token.
- Respons forgot dibuat sama untuk akun ada/tidak ada.
- Belum ada pengiriman email asynchronous/queue; request menunggu provider
  selesai. Untuk volume besar, pindahkan ke worker tanpa mengubah kontrak API.

## Pengujian yang sudah dilakukan

- `TestPasswordResetFlow` integration dengan PostgreSQL: register, forgot,
  ambil token dari URL, reset, sesi lama dicabut, password lama ditolak, token
  tidak bisa dipakai ulang, dan email untuk akun tidak dikenal tidak dikirim.
- Email adapter diuji secara manual melalui Resend ke alamat pengguna; provider
  menerima request dengan status 200.
