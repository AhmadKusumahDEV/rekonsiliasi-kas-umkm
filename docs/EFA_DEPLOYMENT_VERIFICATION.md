# Verifikasi Deployment EFA ke Backend Temukas

Tanggal verifikasi: 28 Oktober 2026.

- Frontend: `https://efa.madgo.my.id/app`
- Backend: `https://temukas.madgo.my.id`
- Akun uji: `pemilik@temukas.test` / `KasDemo!2026`

## Hasil

**Lulus.** Frontend efa sudah memakai backend temukas, bukan lagi fungsi
`/api/...` prototipe. Login, data state, ekspor, dan SSE bekerja dari origin
`efa.madgo.my.id`.

## Bukti statis

| Pemeriksaan | Hasil |
| --- | --- |
| Shell aplikasi | `GET /app` → 200, memuat `assets/app.js?v=team2` |
| Base API | `api-config.js`: produksi memakai `https://temukas.madgo.my.id`, localhost memakai `""` |
| Fetch | `api-client.js`: `credentials: "include"`, `X-CSRF-Token` untuk non-GET, timeout 32 detik |
| EventSource | `app.js`: `new EventSource(apiURL("/api/v1/umkm/events"), { withCredentials: true })` |
| Ekspor | Tautan memakai `apiURL("/api/v1/umkm/export")`, bukan path relatif |
| CSP | `connect-src 'self' https://temukas.madgo.my.id` |
| Password | `PUT /api/v1/auth/password` |
| `/api/v1/*` di efa | 404 dari Vercel; ini benar karena frontend memanggil temukas langsung, bukan lewat rewrite |

## Bukti runtime

1. Login `pemilik@temukas.test` di `https://efa.madgo.my.id/app#login` berhasil.
2. Dashboard menampilkan workspace **Roti Sore (Demo)** dengan **5 penjualan
   Rp650.000** dan **4 penerimaan Rp590.000**, yaitu data dari database
   temukas.
3. Indikator **Data terhubung** muncul.
4. Dari sesi terpisah, satu baris penjualan `SSE-1` Rp1.000 diimpor ke akun yang
   sama. UI efa berubah otomatis menjadi **6 penjualan Rp651.000** tanpa reload.
5. Impor tersebut kemudian dibatalkan. UI kembali otomatis ke **5 penjualan
   Rp650.000**.
6. Tautan **Unduh rekap CSV** mengarah ke
   `https://temukas.madgo.my.id/api/v1/umkm/export`.
7. Console browser tidak menampilkan error atau warning.

## Checklist rencana handoff

| Item | Status |
| --- | --- |
| Base URL produksi ke temukas | Lulus |
| Endpoint `/api/v1` | Lulus |
| Cookie dan `credentials: include` | Lulus |
| CSRF pada request non-GET | Lulus |
| EventSource `withCredentials` | Lulus |
| CSP `connect-src` | Lulus |
| Ekspor CSV absolut | Lulus |
| Domain kustom, bukan preview Vercel | Lulus |
| Login dan data dari database temukas | Lulus |
| Event SSE sampai ke browser | Lulus |

## Catatan minor

1. URL lama seperti `https://efa.madgo.my.id/#reports` sekarang menampilkan
   landing page, bukan aplikasi. Aplikasi berada di `/app#...`. Jika bookmark
   lama masih dipakai, tambahkan redirect dari hash lama ke `/app` yang sesuai.
2. `/health/live` masih menampilkan `version: "dev"` karena build arg
   `VERSION` belum diisi. Tidak memengaruhi fungsi.
3. Data demo temukas adalah 5 penjualan Rp650.000 dan 4 penerimaan Rp590.000,
   berbeda dari data prototipe lama (4 penjualan Rp750.000). Ini sesuai
   `docs/DEMO.md`, bukan bug.
4. Domain preview `*.vercel.app` akan gagal karena cookie `SameSite=Strict`.
   Gunakan `efa.madgo.my.id`.
5. Ada artefak pengujian di database live:
   - akun `qa-20261028-1@temukas.test` dengan workspace `QA Workspace`;
   - batch impor `c6afe4bc1c940bb6f297804081bdab2e` berisi satu baris
     `SSE-1` yang sudah dibatalkan dan tetap terlihat di Riwayat.

   Keduanya tidak memengaruhi angka dashboard, tetapi sebaiknya dibersihkan
   lewat SQL oleh pemilik database jika demo harus benar-benar bersih.
6. SSE masih memakai hub memori satu proses. Untuk beberapa instance API,
   diperlukan bus bersama atau sticky routing.

## Kesimpulan

Tidak ada perubahan backend tambahan yang diperlukan untuk integrasi ini.
Frontend efa sudah mengikuti kontrak di
`docs/HANDOFF_VERCEL_FRONTEND.md`, dan pengujian end-to-end berhasil.
