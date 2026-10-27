# Hosting pada domain tim

Backend Go tetap berjalan seperti sebelumnya. Sajikan frontend statis pada domain yang sama, dan proxy /api/v1/* serta /health/* ke API. Node hanya diperlukan untuk development lokal.

PR ini **belum mengubah server temukas.madgo.my.id**. Pemilik server memasang perubahan setelah review.

## Deployment

1. Backup database dan konfigurasi, catat commit yang sedang berjalan.
2. Build backend dari backend dengan make build-linux atau gunakan artifact CI yang sudah ada. Pertahankan environment dan service user tim.
3. Salin frontend ke /srv/temukas/frontend. Jangan sajikan root repository, backend, .git, configs atau .env.
4. Adaptasi frontend/Caddyfile.example: domain, path, dan upstream harus sesuai server. Jika memakai Nginx, terapkan routing yang sama pada konfigurasi yang ada.
5. Validasi konfigurasi lalu reload proxy secara graceful. Jangan menyalakan proxy kedua pada port yang sudah dipakai.
6. Uji /health/ready, halaman /, login, impor, alokasi dan laporan melalui HTTPS domain yang sama.

Tidak ada migration baru untuk integrasi UI. Untuk database baru tetap terapkan migration tim.

## Proxy dan cookie

Bind backend ke 127.0.0.1:8080 bila proxy berada di server yang sama. Hanya proxy terpercaya boleh mengaksesnya. Proxy harus menimpa X-Forwarded-Proto dan X-Forwarded-For dari client; Echo memakainya untuk Secure cookie dan IP limiter. Jangan percaya header client dari backend yang langsung terbuka ke internet.

Di HTTPS, cookie rku_session harus Secure, HttpOnly, SameSite=Strict. Perubahan password harus menolak sesi lama. Satu origin menghindari third-party cookie; jika sengaja memakai origin terpisah, sesuaikan CORS dan kebijakan cookie dengan pengujian tersendiri.

## Data dan cache

Aset contoh memakai no-cache agar versi baru divalidasi. Auth/state tidak boleh dicache publik. Credential hanya di server.

TEST_DATABASE_URL harus database disposable karena harness menghapus schema. Seed akun publik hanya untuk database demo terpisah. Backup terjadwal, restore teruji, pembatasan PostgreSQL dan kebijakan log menjadi tanggung jawab pengelola hosting.

## Rollback dan batasan

Kembalikan frontend dan binary ke commit sebelumnya lalu restart service/reload proxy. Tidak ada rollback schema yang diperlukan untuk PR ini; jangan hapus transaksi atau reimport otomatis.

SSE branch lain perlu tes streaming/shutdown saat digabung. Limiter per proses belum menjadi limiter bersama beberapa replica. Pengukuran performa dan verifikasi domain sebenarnya tetap diperlukan setelah deployment.
