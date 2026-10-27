# Login dan daftar — EFA navy

Screenshot LIVE dari https://efa.madgo.my.id/, 27 Oktober 2026.
Form daftar tidak dikirim dan data keuangan tidak diubah saat pengujian.

| Halaman | Desktop 1440 px | HP 375 px |
| --- | --- | --- |
| Masuk | [PNG](login-1440.png) | [PNG](login-375.png) |
| Daftar | [PNG](register-1440.png) | [PNG](register-375.png) |

Pengujian juga mencakup tablet 768 px. Keenam kombinasi halaman/ukuran
layar tidak mengalami overflow atau error JavaScript. Login akun demo
di live berhasil membuka dashboard. Tombol lihat/sembunyikan sandi,
navigasi login–daftar, validasi input wajib, dan reduced motion diuji
pada preview lokal dengan source UI yang sama. Registrasi akun baru
tidak diuji di production.

Deployment: `dpl_TPGFiyq3TVRzctuk19KdGHLnSF3M`.
Source Go prototype yang dideploy tetap berbeda dari backend integrasi
tim di repository ini; perubahan ini menyamakan UI kedua source.

Desain: UI Design mode Build (colors, login-pages, form-controls,
responsive-design, images, typography, custom-fonts, buttons, interactivity,
dark-mode, surfaces); UI Animation (decision-framework, component-patterns).
Tidak ada library animasi tambahan. Entrance memakai Web Animations API,
feedback tombol memakai CSS, dan reduced motion menonaktifkan perpindahan.
