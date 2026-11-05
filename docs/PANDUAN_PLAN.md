# Rencana Panduan Pengguna EFA

Dokumen ini merangkum hasil riset tiga sub agent tentang metode penyusunan
panduan produk dan rencana penerapannya. Status: final, siap dieksekusi.
Tanggal: 3 November 2026, diperbarui 5 November 2026.

## Keputusan final: dua format, satu sumber

Panduan dibuat dalam dua cara untuk konten yang sama:

1. **Tutorial halaman** - teks langkah demi langkah untuk setiap halaman yang
   dipakai pengguna, dengan screenshot bernomor.
2. **Video pendek** - 3-4 video 45-90 detik, ditempel di akhir artikel yang
   relevan, bukan halaman terpisah.

Pengguna memilih sendiri: membaca langkah atau menonton video di akhir. Tidak
ada konten yang ditulis dua kali; video adalah ringkasan dari artikel yang sama.

## Landing page dan panduan

Landing page sudah punya bagian "Cara kerja" empat langkah. Bagian itu tetap
singkat dan berfungsi sebagai penjelasan produk, bukan tutorial.

| Tempat | Isi | Tujuan |
| --- | --- | --- |
| Landing page | 4 langkah ringkas, contoh selisih, CTA daftar | Meyakinkan dan menjelaskan alur |
| Panduan | Langkah rinci per halaman, error umum, screenshot, video | Menuntun saat memakai aplikasi |
| Dalam aplikasi | Bantuan kontekstual di layar rawan | Menjawab saat pengguna macet |

Aturannya:

- Landing tidak perlu menjelaskan setiap field atau error.
- Panduan tidak perlu mengulang cerita pemasaran.
- Landing menautkan "Panduan lengkap"; panduan menautkan kembali ke aplikasi.
- Video utama cukup satu teaser di landing, sisanya di akhir artikel panduan.


## Peta halaman dan video

| Halaman | Tutorial teks | Video di akhir |
| --- | --- | --- |
| Impor | Ya, dua slot dan template | Ya, 60-90 detik |
| Verifikasi | Ya, alasan dan fee hint | Ya, 45-60 detik |
| Ringkasan dan catat harian | Ya | Ya, 45-60 detik |
| Transaksi dan laporan | Ya | Tidak wajib |
| Pengaturan, reset password, hapus data | Ya | Tidak wajib |

Halaman lain tetap punya tutorial teks. Video hanya untuk tiga alur yang paling
sering gagal, supaya perawatan tidak membengkak.


## Ringkasan keputusan

Gunakan empat lapis, bukan satu metode tunggal:

1. Halaman depan berbasis tugas utama (top-task entry), bukan daftar fitur.
2. Artikel how-to per tugas, dengan Diátaxis sebagai aturan internal penulisan.
3. Bantuan kontekstual di dalam aplikasi pada empat layar rawan: Impor CSV,
   Cocokkan kas, Perlu diperiksa, dan Ekspor.
4. Referensi dan penjelasan dipisah: format CSV, status transaksi, glosarium,
   cara kerja pencocokan, dan mengapa muncul selisih.

FAQ dibatasi 8-10 pertanyaan faktual. Guided tour panjang tidak dipakai;
paling banyak satu tur opsional dua langkah.

Alasannya: pengguna UMKM datang dengan tujuan (“impor CSV bulan ini”, “kenapa
ada selisih”), bukan dengan istilah dokumentasi. Benchmark 464 produk SaaS
mencatat tur 1-2 langkah selesai 73%, turun ke 38% pada 3-5 langkah dan 8% pada
9+ langkah; tur yang dibuka sendiri selesai 69% versus 23% untuk tur otomatis.
NN/g juga menemukan tutorial awal sering tidak meningkatkan performa tugas,
sedangkan bantuan yang muncul saat pengguna mengerjakan tugas lebih mudah
diterapkan.

## 1. Konteks dan batasan

- Sekitar 10 layar, tim 2-3 orang, tanpa tim support khusus.
- Frontend statis di Vercel: hasil tools/build-static.mjs, CSP ketat
  (script-src self, connect-src self dan API). Tidak ada package.json, jadi
  menambah dependensi ke build aplikasi menaikkan risiko.
- PWA: halaman /app tidak di-cache, hanya halaman offline. Konten bantuan
  karena itu butuh halaman statis yang bisa di-precache, bukan hanya drawer.
- Pengguna: pemilik UMKM, bukan akuntan, dan banyak yang membuka dari ponsel.
- Aset yang sudah ada dan bisa dipakai ulang: docs/PANDUAN_PENGGUNA_EFA.md
  (catatan sudut pandang pengguna), docs/UX_FINDINGS_ACTION_PLAN.md, dan
  docs/ui-guide (capture Playwright, manifest, PDF 42 screenshot).

## 2. Metode yang dipilih

| Metode | Peran | Keputusan | Alasan singkat |
| --- | --- | --- | --- |
| Task-based help center | Tulang punggung struktur | Dipakai | Judul artikel berupa kata kerja dan hasil, bukan nama modul |
| Top-task entry | Halaman depan bantuan | Dipakai | Lima pintu masuk paling sering plus pencarian |
| Diátaxis | Template internal penulisan | Dipakai terbatas | Mencegah artikel campur instruksi, referensi, dan penjelasan; labelnya tidak ditampilkan ke pengguna |
| Knowledge base dan search | Pencarian | Dipakai mulai 8-10 artikel | Pengguna mencari, bukan menelusuri menu; sinonim wajib |
| FAQ | Pelengkap | Dibatasi 8-10 | Hanya pertanyaan faktual, bukan prosedur |
| Guided tour | Onboarding | Dibatasi | Maksimal dua langkah, opsional, bisa dilewati dan diulang |
| Checklist setup | Ringkasan | Dipakai | Tiga langkah yang statusnya dihitung dari data nyata |
| Empty state | Semua daftar dan hasil filter | Dipakai | Membedakan “belum ada data” dari “masih memuat” |
| Help drawer | Empat layar rawan | Dipakai | Konten menengah, pengguna tetap melihat layar |
| Video tutorial | Onboarding | Tidak dipakai | Mahal dirawat; GIF pendek hanya bila benar-benar perlu |

Pola yang ditiru: QuickBooks untuk struktur artikel (prasyarat, langkah,
troubleshooting, tautan terkait), Xero untuk titik masuk bantuan di dalam
produk, Google Workspace untuk pencarian sebagai pintu utama, dan Mailchimp
untuk satu halaman onboarding yang mengarah ke tugas berikutnya.

## 3. Arsitektur informasi

```text
Bantuan
├── Cari bantuan (kotak pencarian di atas)
├── Mulai cepat (tutorial 10 menit)
├── Tugas utama
│   ├── Impor CSV
│   ├── Tinjau transaksi
│   ├── Jalankan kecocokan otomatis
│   ├── Selesaikan yang perlu diperiksa
│   ├── Buat laporan
│   ├── Ekspor data
│   └── Atur akun dan tim
├── Referensi
│   ├── Format CSV yang diterima
│   ├── Status transaksi
│   └── Istilah dan singkatan
├── Penjelasan
│   ├── Cara kerja pencocokan otomatis
│   └── Mengapa muncul selisih
├── FAQ
└── Hubungi bantuan
```

Aturan navigasi:

- Maksimal dua tingkat yang terlihat: kategori lalu artikel.
- Urutkan mengikuti alur kerja (impor, tinjau, cocokkan, bereskan selisih,
  laporan, ekspor, pengaturan), bukan abjad atau nama modul.
- Satu halaman kanonik per tugas. Artikel lain menautkan, bukan menyalin langkah.
- Referensi dan penjelasan tidak dicampur ke dalam langkah tugas.
- Pencarian memakai sinonim pengguna: impor/unggah/upload/CSV/Excel,
  cocok/cocokkan/rekonsiliasi, selisih/pengecualian/belum cocok,
  laporan/rekap, ekspor/unduh.

## 4. Daftar halaman

Total 16 halaman. Sepuluh di antaranya prioritas 1 untuk rilis panduan pertama.

| ID | Judul | Jenis | Prioritas |
| --- | --- | --- | --- |
| quick-start | Mulai cepat: dari impor pertama sampai laporan | Tutorial | 1 |
| import-csv | Impor penjualan dan penerimaan dari CSV | How-to | 1 |
| import-errors | Memperbaiki error impor | Troubleshooting | 1 |
| review-transactions | Meninjau dan mencari transaksi | How-to | 1 |
| match-auto | Menjalankan dan membaca hasil kecocokan | How-to | 1 |
| resolve-exceptions | Menyelesaikan transaksi yang perlu diperiksa | How-to | 1 |
| partial-payment | Memasangkan pembayaran sebagian | How-to | 2 |
| reports | Membuat dan membaca laporan | How-to | 1 |
| export | Mengekspor data | How-to | 2 |
| account | Mengatur akun, kata sandi, dan data workspace | How-to | 2 |
| csv-format | Format CSV yang diterima | Referensi | 1 |
| transaction-status | Arti status transaksi | Referensi | 2 |
| glossary | Istilah dan singkatan | Referensi | 1 |
| how-matching-works | Cara kerja pencocokan otomatis | Penjelasan | 2 |
| why-differences | Mengapa muncul selisih dan sisa tagihan | Penjelasan | 1 |
| faq | Pertanyaan yang sering diajukan | FAQ | 2 |

## 5. Template artikel dan contoh

Setiap artikel memakai urutan yang sama:

```text
Judul: kata kerja + hasil
Satu kalimat: kapan dipakai dan hasil akhirnya
Sebelum mulai: 2-3 syarat (data, akses, periode)
Langkah 1-7: satu tindakan per langkah, label UI ditulis persis
Jika tidak berhasil: tiga error umum dan solusinya
Selanjutnya: dua tautan terkait
Apakah halaman ini membantu? (Ya/Tidak)
```

Contoh ringkas untuk impor CSV:

```text
# Impor penjualan dan penerimaan dari CSV

Gunakan halaman ini saat ingin memasukkan transaksi bulan ini dari file CSV.
Hasil akhir: transaksi muncul di daftar dan siap dicocokkan.

## Sebelum mulai
- File berisi tujuh kolom: id, tanggal, referensi, nama, nominal, kanal, catatan.
- Maksimal 500 baris dan 1 MB per file.
- Format tanggal YYYY-MM-DD, nominal tanpa titik atau Rp.

## Langkah
1. Buka menu Impor.
2. Pilih Penjualan atau Penerimaan.
3. Tekan Unduh template bila belum punya contoh format.
4. Pilih file CSV, lalu tunggu pratinjau muncul.
5. Periksa jumlah baris baru, duplikat yang dilewati, dan error.
6. Tekan Simpan impor hanya setelah angka sesuai.

## Jika tidak berhasil
- “Baris 12: nominal berisi Rp1.250.000” → isi angka saja: 1250000.
- “Baris 8: tanggal tidak valid” → gunakan format 2026-10-31.
- File ditolak karena lebih dari 500 baris → pecah menjadi beberapa file.
```

## 6. Bantuan di dalam aplikasi

Empat titik bantuan, masing-masing membuka drawer berisi tiga langkah pertama
dan tautan ke artikel penuh:

| Layar | Pemicu | Isi singkat |
| --- | --- | --- |
| Impor | Tombol “Bantuan format CSV” | Tujuh kolom, contoh baris, batas ukuran, tautan template |
| Cocokkan kas | Ikon “?” di judul | Apa arti pasangan, kapan dianggap yakin, apa yang terjadi setelah disimpan |
| Perlu diperiksa | Ikon “?” per kartu | Mengapa masuk sini, perbedaan nominal/tanggal/referensi, cara memilih |
| Ekspor | Ikon “?” di tombol | Isi file, periode, dan bahwa ekspor tidak mengubah data |

Aturan permukaan bantuan:

- Tooltip hanya untuk tambahan opsional, bukan syarat menyelesaikan tugas.
- Drawer untuk penjelasan menengah dan panjang yang tidak memblokir.
- Modal hanya untuk keputusan berisiko: simpan pasangan manual, pembayaran
  sebagian, dan kosongkan data workspace.
- Empty state menjelaskan status, sebab, dan tindakan berikutnya; jangan
  menampilkan “tidak ada data” saat masih memuat.

Checklist di Ringkasan memakai tiga langkah yang dihitung dari state, bukan
hardcode: impor penjualan, impor penerimaan, konfirmasi satu pasangan.
Checklist bisa ditutup dan dibuka lagi.

Konten bantuan disimpan sebagai manifest JavaScript yang ikut bundel aplikasi,
bukan diambil dari domain lain, supaya tidak melanggar CSP connect-src:

```js
// frontend/assets/help-content.js
window.EFA_HELP = {
  schemaVersion: 1,
  version: "2026-11-03.1",
  locale: "id-ID",
  entries: [
    {
      id: "glossary.sisa-tagihan",
      kind: "glossary",
      title: "Sisa tagihan",
      short: "Penjualan yang belum punya pasangan penerimaan.",
      body: "Bisa berarti belum dibayar, dibayar sebagian, atau menunggu pencairan. Ini bukan kerugian.",
      keywords: ["belum cocok", "selisih", "uang masuk"],
      appliesTo: ["overview", "reports"],
      updatedAt: "2026-11-03"
    }
  ]
};
```

Validasi saat build: id unik, field wajib lengkap, appliesTo hanya route yang
dikenal. Untuk aksesibilitas, gunakan elemen dialog native untuk konfirmasi dan
drawer yang memblokir, tombol ikon dengan nama aksesibel, aria-expanded dan
aria-controls untuk panel, serta hormati prefers-reduced-motion.

## 7. Bahasa dan glosarium

| Istilah internal | Label untuk pengguna | Penjelasan singkat |
| --- | --- | --- |
| Kecocokan | Sudah cocok | Dua transaksi dianggap satu pasangan karena nominal, tanggal, dan arah dana sesuai |
| Pengecualian | Perlu diperiksa | Belum bisa dicocokkan otomatis; ada beberapa transaksi mirip |
| Sisa tagihan | Penjualan belum tercocokkan | Bagian penjualan tanpa pasangan penerimaan; bukan kerugian |
| Alokasi | Pasangkan pembayaran | Membagi satu penerimaan ke satu atau beberapa penjualan |
| Kandidat ambigu | Transaksi yang mirip | Lebih dari satu kemungkinan; pilih yang nominal dan referensinya paling sesuai |
| Fee | Biaya layanan | Potongan dari uang yang masuk, bukan penjualan tambahan |
| Rekonsiliasi | Cocokkan kas | Membandingkan catatan penjualan dengan uang yang benar-benar masuk |

Gaya penulisan: kalimat pendek, kata sehari-hari, satu jalur yang disarankan,
label tombol ditulis persis, dan troubleshooting memakai teks error yang muncul
di layar. Istilah akuntansi dijelaskan sekali di glosarium, bukan diselipkan
sebagai asumsi.

## 8. Alur kerja konten di repo

Struktur yang diusulkan:

```text
docs-site/
├── astro.config.mjs         (base: /panduan)
├── package.json
└── src/content/docs/
    ├── index.mdx            (top-task entry)
    ├── mulai-cepat.md
    ├── tugas/
    │   ├── impor-csv.md
    │   ├── tinjau-transaksi.md
    │   ├── cocokkan.md
    │   ├── selesaikan-selisih.md
    │   ├── laporan.md
    │   ├── ekspor.md
    │   └── akun.md
    ├── referensi/
    │   ├── format-csv.md
    │   ├── status-transaksi.md
    │   └── istilah.md
    └── penjelasan/
        ├── cara-kerja-pencocokan.md
        └── mengapa-ada-selisih.md
```

Screenshot memakai pipeline yang sudah ada (docs/ui-guide/capture-ui.cjs dan
build-pdf.py). Simpan PNG di docs-site/public/ dengan nama
layar-keadaan-lebar.png, lalu rujuk dari Markdown. Setiap gambar diberi nomor
langkah; screenshot layar penuh tanpa anotasi tidak dipakai.

## 9. Tooling

| Opsi | Kekuatan | Kelemahan | Cocok untuk EFA |
| --- | --- | --- | --- |
| Astro Starlight | Pencarian Pagefind tanpa server, i18n, aksesibilitas baik | Perlu proyek Node terpisah; sebagian skrip inline harus lolos CSP | Ya, pilihan utama |
| VitePress | Cepat, markdown, pencarian lokal bawaan, konfigurasi kecil | i18n terbatas, perlu Node saat build | Ya, alternatif |
| Docusaurus | Paling lengkap, versi dokumen, i18n | Berat, banyak dependensi, berlebihan untuk 16 halaman | Belum |
| MkDocs Material | Pencarian sangat baik, banyak tema | Perlu Python; proyek ini EOL 5 Mei 2027 | Tidak untuk proyek baru |
| Docsify | Tanpa build, markdown langsung | Rendering saat runtime, CSP dan offline merepotkan | Tidak |
| Markdown di repo tanpa situs | Nol biaya | Tidak ada pencarian, sulit dibaca non-teknis | Hanya untuk arsip internal |

Keputusan: Astro Starlight di folder docs-site/ dengan package.json sendiri,
hasil build disalin ke dist/panduan oleh tools/build-static.mjs. Satu domain,
SEO menyatu, pencarian Pagefind berjalan tanpa server, dan frontend utama tetap
statis tanpa dependensi baru. VitePress menjadi alternatif bila Starlight
terbukti terlalu berat.

Satu hal yang wajib diuji lebih dulu: header CSP saat ini memakai
script-src self untuk seluruh path. Starlight menyisipkan skrip inline untuk
inisialisasi tema, jadi build harus menghasilkan hash atau nonce, atau skrip
itu dipindahkan ke file terpisah. Jika CSP tidak bisa dipertahankan tanpa
melonggarkan aturan, pindahkan situs ke proyek Vercel terpisah di subdomain
(misalnya panduan.madgo.my.id) sebagai fallback; konsekuensinya satu CNAME di
Cloudflare dan tautan absolut dari aplikasi.

## 10. Pemeliharaan agar tidak basi

- Satu pemilik dokumentasi dari tim 2-3 orang.
- Definition of done: PR yang mengubah label, tombol, atau alur wajib
  memperbarui artikel terkait dan screenshot dalam PR yang sama.
- Setiap artikel mencantumkan tanggal “Terakhir diperbarui”.
- Review kuartalan, plus review langsung setelah rilis yang mengubah alur inti.
- Screenshot diambil ulang dengan capture-ui.cjs; manifest mencatat tanggal dan
  hash aset sehingga gambar lama tidak diam-diam dipakai.
- Changelog pengguna singkat (“Apa yang baru”) setiap rilis.
- Maksimal 16 halaman; artikel baru harus menggantikan atau menggabungkan yang
  ada agar perawatan tidak membengkak.

## 11. Pengukuran

Ukuran utama bukan pageview:

1. Task success: uji 5 pemilik UMKM dengan empat tugas (impor CSV, temukan
   transaksi belum cocok, selesaikan satu selisih, ekspor laporan).
2. Search success: query tanpa hasil dan pencarian ulang setelah membuka artikel.
3. Helpfulness: tombol “Apakah halaman ini membantu?” di akhir artikel.
4. Contact rate per topik: pertanyaan WhatsApp/email yang masuk, ditandai topik.
5. Funnel produk: impor dimulai versus selesai, pasangan pertama dikonfirmasi,
   laporan diekspor.

Implementasi ringan: Vercel Web Analytics untuk pageview dan empat event kustom
(bantuan dibuka, pencarian, hasil kosong, umpan balik). Endpoint backend khusus
umpan balik baru dibuat bila volume sudah layak; jangan menambah tabel dulu.

## 12. Roadmap

| Fase | Isi | Estimasi |
| --- | --- | --- |
| 0 | Setujui glosarium, audit label yang membingungkan, siapkan kerangka Starlight | 0,5 hari |
| 1 | Sepuluh artikel prioritas 1, halaman depan top-task, build /panduan dan uji CSP | 3-4 hari |
| 2 | Help drawer di empat layar, checklist Ringkasan, perbaikan empty state, help-content.js | 2 hari |
| 3 | Screenshot dan PDF, analytics, umpan balik, changelog | 1-2 hari |
| Rutin | Review kuartalan, update saat rilis | 2-3 jam per bulan |

Total 6-8 hari kerja orang dan bisa berjalan paralel dengan pengembangan fitur.

## 13. Risiko

| Risiko | Dampak | Mitigasi |
| --- | --- | --- |
| Dokumentasi basi setelah UI berubah | Pengguna mengikuti langkah salah | DoD PR, review rilis, tanggal diperbarui |
| Screenshot cepat usang | Kebingungan letak tombol | Capture otomatis, anotasi, hash provenance |
| Terlalu banyak artikel | Beban perawatan melebihi manfaat | Batas 16 halaman, satu kanonik per tugas |
| Bantuan hanya di drawer | Tidak tersedia offline dan tidak bisa dicari | Halaman statis plus drawer dari manifest yang sama |
| CSP memblokir konten bantuan | Drawer kosong | Manifest ikut bundel, bukan fetch lintas domain |
| Skrip inline Starlight diblokir CSP | Halaman panduan gagal memuat tema atau pencarian | Uji lebih dulu; pakai hash atau file eksternal; fallback ke subdomain |
| Istilah teknis lolos | Pengguna berhenti di langkah pertama | Glosarium dan uji copy dengan 5 pengguna |

## 14. Checklist eksekusi

- [ ] Setujui tabel glosarium bagian 7.
- [ ] Audit label yang bertentangan dengan glosarium di frontend.
- [ ] Buat folder docs-site dan kerangka Starlight.
- [ ] Tulis sepuluh artikel prioritas 1.
- [ ] Build /panduan dan pastikan CSP tetap ketat (fallback: subdomain).
- [ ] Tambahkan tautan Bantuan di footer landing, halaman masuk, dan Pengaturan.
- [ ] Implementasikan help drawer dan checklist Ringkasan.
- [ ] Ambil ulang screenshot dengan capture-ui.cjs.
- [ ] Ukur dengan lima pengguna dan catat hasilnya di docs/VERIFICATION.md.

## Rujukan

- Diátaxis: https://diataxis.fr/start-here/
- NN/g, onboarding tutorial versus contextual help: https://www.nngroup.com/articles/onboarding-tutorials/
- NN/g, empty state: https://www.nngroup.com/articles/empty-state-interface-design/
- NN/g, tooltip: https://www.nngroup.com/articles/tooltip-guidelines/
- NN/g, confirmation dialog: https://www.nngroup.com/articles/confirmation-dialog/
- QuickBooks help center: https://quickbooks.intuit.com/learn-support/en-us/quickbooks-online/all-topics/
- Google Workspace Learning Center: https://support.google.com/a/users/
- Zendesk, struktur self-service: https://support.zendesk.com/hc/en-us/articles/4408883389338
- Benchmark onboarding SaaS 2026: https://produktly.com/research/saas-onboarding-benchmarks-2026
- WCAG 2.2, error prevention: https://www.w3.org/WAI/WCAG22/Understanding/error-prevention-legal-financial-data.html
- MDN, PWA caching: https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Caching
- Astro Starlight, pencarian: https://starlight.astro.build/guides/site-search/
- Pagefind: https://pagefind.app/
- VitePress, pencarian: https://vitepress.dev/reference/default-theme-search
- MkDocs Material, status proyek: https://github.com/squidfunk/mkdocs-material

