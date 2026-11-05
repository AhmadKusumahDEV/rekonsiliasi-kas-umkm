# Audit Frontend Live — 5 November 2026

Target: https://efa.madgo.my.id/app

## Kesimpulan

Frontend live sudah mengikuti dokumen terbaru. Dua slot impor, Verifikasi,
fee hint, reset password, dan kosongkan workspace sudah ada. Masalahnya ada
pada dua bug UX dan pada source repo yang tertinggal dari live.

## Status fitur

| Fitur | Live |
| --- | --- |
| Dua slot impor | Ada |
| Endpoint `/umkm/imports/flexible` | Ada |
| Template penjualan dan contoh mutasi | Ada |
| `auto_matched` dan `ai_matched` | Ada di UI |
| Halaman `#verifikasi` | Ada, alias rute lama |
| `fee_hint` | Ada di kartu |
| Kategori cash dan non-operational | Ada |
| `#forgot` dan `#reset` | Ada |
| Danger zone kosongkan data | Ada |
| PWA shortcut Verifikasi | Ada |

## Bug yang ditemukan

**1. Membuka route reset membuat sesi terlihat berakhir.**

Di bundle live, `app.js` baris 960 melakukan:

```js
if (["forgot","reset"].includes(normalizeRoute())) { state.me = null; auth(); }
```

Akibatnya state pengguna dikosongkan di klien. Saat kembali ke `#overview`,
aplikasi menampilkan login meskipun cookie sesi masih ada. Perbaikannya:
jangan mengosongkan `state.me` untuk route reset, atau panggil ulang
`/auth/me` saat meninggalkan route tersebut.

**2. Kartu `fee_hint` terpotong di layar sempit.**

Teks “Saran biaya Rp10.000” terbungkus sangat sempit pada kartu marketplace.
Perlebar kartu, kurangi padding, atau pindahkan label ke baris penuh.

**3. Error 401 `/auth/me` sebelum login.**

Ini muncul saat pemuatan awal sebelum sesi ada. Tidak berbahaya, tetapi bisa
dihilangkan dengan tidak memanggil `/auth/me` bila cookie belum ada.

## Source repo tertinggal

| File | Repo | Live |
| --- | --- | --- |
| `app.js` | versi lama, satu modal impor | versi baru, dua slot |
| `import-flow.js` | tidak ada | ada |
| `verification.js` | tidak ada | ada |
| `validation.js` | versi lama | menambah filter settled/cash |
| `template-penjualan.csv` | tidak ada | ada |
| `contoh-mutasi-bank.csv` | tidak ada | ada |
| manifest PWA | shortcut lama | shortcut Verifikasi |

Deploy ulang dari repo sekarang akan menghapus fitur yang sudah live.

## Tindakan

1. Minta frontend mem-backport bundle live ke source repo.
2. Perbaiki `state.me` pada route reset.
3. Perbaiki layout `fee_hint`.
4. Samakan versi aset, misalnya `handoff2`, termasuk import internal di `app.js`.
5. Merge branch backend `feat/ai-matching-verification` agar `ai_matched`
   benar-benar dikirim API.
6. Setelah source repo sama dengan live, baru deploy ulang.

