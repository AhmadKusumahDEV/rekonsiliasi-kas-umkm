# Preview landing EFA

Screenshot dari https://efa.madgo.my.id/ pada 29 Oktober 2026.
Hero terbaru memakai komposisi penuh, bentuk biru berlapis, copy di kiri,
dan ilustrasi tiga orang transparan di kanan. Ilustrasi orang dibuat baru;
bukan testimoni pengguna. Section bawah mempertahankan gaya EFA sebelumnya.
Hero utama dan screenshot seluruh halaman memakai slide pertama,
setelah semua section terlihat dan animasi selesai.

| Awalan file | Viewport |
| --- | --- |
| desktop | 1440×1000 px |
| tablet | 768×1024 px |
| mobile | 375×812 px |
| small | 320×568 px |

`*-hero.png` menampilkan bagian atas. `*-full.png` memperlihatkan semua
konten. `desktop-slide-2.png` dan `desktop-slide-3.png` menunjukkan foto
dan judul carousel lainnya. Pada layar pendek, hero dapat digulir; konten
dan tombol tidak dipotong oleh tinggi yang dikunci.

Pemeriksaan browser: tiga slide, autoplay, pause, panah/Home/End,
reduced motion, menu HP dan Escape, tiga contoh nominal, FAQ, CTA
login/daftar, serta pengalihan tautan aplikasi lama. Tidak ditemukan
overflow horizontal atau page error pada empat ukuran di atas.
Tanpa JavaScript, konten utama dan tautan pendaftaran tetap terbaca.

Pemeriksaan aplikasi live memakai akun demo yang sudah ada, tanpa
membuat/mengubah transaksi: login, 21 kombinasi halaman/ukuran,
cookie, handshake SSE dan unduhan CSV berhasil. API tetap milik
server tim; landing tidak mengirim atau membaca data transaksi.

Panduan mengubah halaman: `../../landing/README.md`.
Asal foto dan referensi: `../../landing/SOURCES.md`.
