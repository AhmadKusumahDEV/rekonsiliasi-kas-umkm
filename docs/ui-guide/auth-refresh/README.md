# Login dan daftar — EFA editorial

Screenshot LIVE dari https://efa.madgo.my.id/, 28 Oktober 2026.
Form daftar tidak dikirim dan data keuangan tidak diubah saat pengujian.

| Halaman | Desktop 1440 px | HP 375 px |
| --- | --- | --- |
| Masuk | [PNG](login-1440.png) | [PNG](login-375.png) |
| Daftar | [PNG](register-1440.png) | [PNG](register-375.png) |

Pengujian mencakup lebar 320, 375, 768 dan 1440 px. Login akun demo
di live berhasil membuka dashboard. Tombol lihat/sembunyikan sandi,
navigasi login–daftar, validasi input wajib, dan reduced motion diuji
pada preview lokal dengan source UI yang sama. Registrasi akun baru
tidak diuji di production.

Pemeriksaan live terakhir mencakup 36 kombinasi: login, daftar, dan tujuh
halaman utama pada empat lebar layar. Tidak ditemukan overflow halaman,
page error JavaScript, atau respons HTTP gagal selain 401 awal yang memang
diharapkan sebelum login. Dialog impor/pengaturan dapat dibuka dan ditutup,
menu HP dapat dibuka dan ditutup, serta reduced motion tidak menyisakan
animasi berjalan. Pengujian ini tidak mengirim perubahan transaksi.

Komposisi mengikuti referensi pengguna: form kiri, input pil, panel foto
kanan dengan takikan melengkung, dan kartu terang yang menumpuk. Foto
login dan daftar berbeda, dengan pilihan 2400 x 3600 atau 4000 x 6000
melalui srcset, dari Mikhail Nilov/Pexels. Kredit disimpan di ASSETS.md;
caption foto dan catatan PBL telah dihapus dari halaman login/daftar.
Di HP form tampil lebih dahulu, dilanjutkan panel visual; seluruh isi
tetap bisa digulir. Tidak ada testimoni atau angka keberhasilan rekaan.

Deployment: `dpl_Ca6ZyoiDM7Zc9WYNBHEToTauUjpi`.
Source Go prototype yang dideploy tetap berbeda dari backend integrasi
tim di repository ini; perubahan ini menyamakan UI kedua source.

Desain: UI Design mode Build (colors, login-pages, form-controls,
responsive-design, images, typography, custom-fonts, buttons, interactivity,
dark-mode, surfaces); UI Animation (decision-framework, component-patterns).
Tidak ada library animasi tambahan. Entrance memakai Web Animations API,
feedback tombol memakai CSS, dan reduced motion menonaktifkan perpindahan.
