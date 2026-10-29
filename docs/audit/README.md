# Hasil pemeriksaan audit EFA

Pemeriksaan ulang 29 Oktober 2026 atas [laporan tim](../EFA_BUG_REPORT.md).
Laporan asli dipertahankan. Tabel berikut membedakan kesalahan yang bisa
diulang, perbaikan kenyamanan, dan batasan yang masih ada.

## Kesimpulan

Laporan tersebut berisi masalah nyata. Namun, tidak semua poin merupakan
bug yang merusak data. Delapan skenario browser pembanding gagal pada
`app.js` sebelum perubahan (`c3a4b67`) dan lulus setelah perubahan.
Duplikasi impor yang mengubah audit dibuktikan melalui integration test
HTTP dengan PostgreSQL lokal. Batas bcrypt 72 **byte** memang diperlukan;
penjelasan UI sebelumnya yang keliru.

Fixture browser memakai data buatan dan seluruh API-nya diintersep.
Tes backend memakai database disposable `rku_integration_test`, bukan
database tim. Pemeriksaan live hanya login akun demo yang sudah tersedia,
membaca halaman, membuka SSE, dan mengunduh rekap. Tidak ada impor,
alokasi, penghapusan, atau perubahan kata sandi di database produksi.

## Sembilan temuan utama

| No. laporan | Hasil validasi | Perubahan |
| --- | --- | --- |
| 1. Halaman basi setelah simpan | Terulang saat request refresh lama masih berjalan ketika alokasi disimpan; bukan setiap penyimpanan | Penyimpanan menunggu refresh yang berjalan, lalu meminta snapshot baru; event versi yang masuk selama refresh dicatat |
| 2. Hash login/register dengan sesi aktif | Terulang di browser | Dinormalisasi ke Ringkasan |
| 3. Total pratinjau menghitung duplikat | Terulang: Rp200.000 sebelumnya, Rp50.000 setelah baris identik dilewati | Preview server membaca workspace; UI juga memfilter agar tetap kompatibel dengan server lama |
| 4. Impor identik menaikkan versi/audit | Dikonfirmasi dari kode lama dan diuji lewat HTTP/database | Impor identik mengembalikan hasil tanpa membuat batch, audit, version bump, atau publish SSE baru |
| 5. Klik luar dialog | Terulang; ini pilihan interaksi, bukan dialog benar-benar macet | Klik backdrop mengikuti aturan tutup yang sama: konfirmasi jika kotor, ditolak selama request menulis |
| 6. Fokus ke Tutup | Terulang; perbaikan aksesibilitas | Fokus awal ke field pertama atau judul dialog |
| 7. Label Menyimpan untuk semua submit | Terulang pada preview | Label dibedakan: Memproses, Mendaftarkan, Memeriksa, Menyimpan, Mengganti |
| 8. Karakter vs byte password | Ketidakcocokan nyata; ekspektasi agar 75 byte diterima perlu dikoreksi | Minimal 12 Unicode code point, maksimal 72 byte UTF-8; UI dan backend sama, tanpa mengganti bcrypt |
| 9. Ekspor sesi habis ke JSON | Terulang di browser | Fetch dengan cookie, validasi CSV, unduh blob; 401 kembali ke login di `/app` |

## Temuan tambahan

Urutan mengikuti bagian “Temuan kode yang belum divalidasi di browser”.

| Temuan | Keputusan dan bukti |
| --- | --- |
| ID alokasi hilang memilih transaksi pertama | Bug terulang. ID eksplisit yang hilang sekarang ditolak; hanya form tanpa pilihan awal boleh memakai opsi pertama. Unit dan browser test tersedia |
| Suggestions gagal membuang state baru | Bug terulang. State diterapkan sebelum suggestions; fixture 503 membuktikan transaksi baru tetap terlihat |
| State dan suggestions berbeda snapshot | Risiko desain nyata. UI menolak kandidat yang tidak cocok dengan entri/saldo snapshot; fixture ID hilang lulus. Dua endpoint tetap belum menjadi satu snapshot atomik; server tetap memeriksa saldo di dalam transaksi |
| SSE saat refresh berlangsung | Cabang lama membuang event. Versi tertinggi sekarang ditampung untuk refresh berikutnya; skenario refresh bersamaan dengan simpan mengujinya |
| CSRF tab lain | Recovery sebelumnya tidak ada. 403 kini meminta ulang identitas/CSRF; pergantian akun mengakhiri UI sesi. POST keuangan tidak diputar ulang otomatis, pengguna harus memeriksa lalu mencoba lagi. Unit test membuktikan satu POST saja |
| Login sukses, state gagal | Recovery ditambahkan. Browser fixture sesi aktif + state 503 menampilkan “Sudah masuk, catatan belum terhubung”, bukan form login; tombol coba ulang memulihkan dashboard |
| Draft hilang saat sesi habis | Batasan tetap ada. Isian sensitif tidak disimpan di localStorage dan harus diisi ulang setelah login. Pesan menjelaskannya; belum ada pemulihan draft lintas sesi |
| 500 baris hanya di server | Bukan celah validasi: server sudah menolak. UI kini menjelaskan batas 500 baris/1 MiB. Parser CSV kedua tidak ditambahkan di browser |
| Batas alokasi tidak mengikuti pilihan | Diperbaiki. Ganti penjualan/penerimaan menghitung ulang max dan sisa; handler UI memeriksa amount + fee dan server tetap otoritatif |
| Kuota sebelum dedupe / termasuk batal | Perhitungan sebelum dedupe memang keliru dan dipindah setelah klasifikasi baris baru. Entri batal tetap dihitung karena masih menggunakan penyimpanan dan audit; batas ini tidak diubah menjadi kuota entri aktif |
| CSV non-UTF-8 rusak diam-diam | Diperbaiki memakai TextDecoder fatal. Browser fixture byte invalid membuktikan request preview tidak dikirim |
| Semua duplikat membuat batch kosong | Diperbaiki dan diuji HTTP: file checksum berbeda tetapi seluruh isi identik tetap tidak membuat batch/audit/version baru |
| Pagination klien / audit 120 terbaru | Batasan produk tetap ada. Belum ada pagination server atau penelusuran seluruh audit historis; tidak diklaim selesai |
| Fokus gelap kurang kontras | Warna fokus di auth/account/menu diubah ke cyan terang dengan outline 3 px. Ini bukan sertifikasi aksesibilitas seluruh aplikasi |
| Toggle password tanpa status | Ditambahkan aria-pressed, aria-controls, dan label sesuai kondisi |
| Preview tidak diumumkan | Region berlabel dengan aria-live dan pemindahan fokus setelah respons |
| Error toast hilang 4,5 detik | Error umum kini bertahan sampai ditutup; pesan sukses tetap singkat. Error form tetap melekat pada form |
| Scroll tabel tak bisa difokuskan | Container tabel diberi tabindex, role region, dan label |
| Target kecil pada 900 px | Tombol small minimal 44 px hingga breakpoint 1100 px |
| Tujuan setelah login hilang | Hash halaman yang dikenal disimpan selama pemulihan sesi dan dipakai setelah login |
| Pesan jaringan bahasa Inggris | TypeError jaringan diterjemahkan menjadi arahan memeriksa data sebelum mengulang; unit test lulus |
| Menu HP layar pendek | Tidak terbukti terpotong pada 375×320. Browser test dapat menggulir ke Tutup menu, menutupnya, dan tidak menemukan overflow horizontal. Bukan bukti semua perangkat fisik |

## Source dan deployment

- Ketidaksamaan `main` dan live benar: merge PR fitur sebelumnya berakhir
  pada branch fitur lain. Branch pekerjaan ini membawa landing dan PWA
  sekaligus perbaikannya langsung ke `main` melalui satu PR.
- Pada pengecekan 29 Oktober, `GET /assets/app.js` dengan User-Agent kosong
  maupun `Mozilla/5.0` sama-sama **200 JavaScript**. Klaim 404 khusus
  non-browser tidak terulang; tidak ditambahkan workaround tanpa bukti.
- Artefak QA di database bersama tidak dihapus. Data tersebut bukan bukti
  kerusakan aplikasi; pembersihan perlu koordinasi pemilik workspace.
- Frontend baru sudah dipublikasikan ke `https://efa.madgo.my.id/`, deployment
  `dpl_73PRse74cQVLfQgNKkFxSfMFxAtp`. Vercel hanya menampung frontend statis.
- **Perubahan Go di PR belum dipasang pada server tim.** Jalankan pipeline
  deployment backend tim setelah review/merge. Tidak ada migration, query,
  file generated sqlc, Dockerfile, atau konfigurasi credential yang diubah.
  Field `skipped` pada preview bersifat tambahan; UI kompatibel dengan
  respons server lama maupun baru.

## Bukti dan menjalankan ulang

- [Sebelum](browser-before.json): delapan regresi browser pada source lama.
- [Sesudah](browser-after.json): 13 skenario browser fixture lulus.
- [Landing live](landing-live.json): 320/375/768/1440 px, tiga slide,
  autoplay/pause/keyboard, reduced motion, tanpa JavaScript, tanpa overflow.
- [Aplikasi live](app-live-readonly.json): login, cookie HttpOnly/Secure,
  21 kombinasi halaman/ukuran, SSE dan ekspor; tanpa perubahan transaksi.
- [PWA live](pwa-live.json): manifest, tidak ada installability error Chrome,
  fallback offline, reconnect, tidak ada cache data privat.
- [Screenshot](../landing-preview/README.md): diambil dari domain live.

Dari root repository:

```powershell
node --test frontend/tests/*.test.mjs
$env:PORT='3058'; node frontend/dev.mjs
```

Di terminal lain, gunakan Playwright yang sudah tersedia pada mesin uji
(tidak diperlukan pada build hosting):

```powershell
# Jika modul tidak ada di node_modules lokal, isi lokasi modul yang sudah terpasang.
$env:PLAYWRIGHT_MODULE='C:\lokasi\node_modules\playwright'
node frontend/tests/browser/audit.cjs
# Opsional pembanding source lama; delapan failure diharapkan.
$env:AUDIT_BASELINE='1'; node frontend/tests/browser/audit.cjs
Remove-Item Env:AUDIT_BASELINE
```

Skrip hanya menerima localhost/127.0.0.1, memakai API fixture dan data buatan.
Hasilnya ditulis ke `docs/audit/browser-*.json`.

Backend, dari `backend/`, dengan database **khusus pengujian yang boleh di-reset**:

```powershell
$env:TEST_DATABASE_URL='postgres://postgres@127.0.0.1:55439/rku_integration_test?sslmode=disable'
$env:TEST_REQUIRE_DB='1'
go test ./...
go vet ./...
go build ./...
```

Ketiganya lulus pada pemeriksaan lokal. Suite frontend lulus 14 tes.
Kondisi ini merupakan hasil pengujian pada skenario di atas, bukan jaminan
tidak akan ada bug atau sertifikasi siap menampung beban produksi tertentu.
