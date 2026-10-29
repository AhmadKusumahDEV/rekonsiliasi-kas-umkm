# Pemeriksaan PWA EFA

Diambil dari https://efa.madgo.my.id/ pada 29 Oktober 2026.

Viewport desktop 1440×1000 dan HP 375×812. Screenshot memperlihatkan
petunjuk pemasangan dan halaman offline setelah worker berhasil aktif.
Folder ini hanya dokumentasi; tidak ikut aset hosting.

- Manifest EFA berhasil dibaca browser.
- Chrome dengan profil uji biasa melaporkan tanpa installability error.
- Dialog petunjuk dapat ditutup dengan Escape.
- Navigasi baru saat offline membuka fallback; saat online, Coba lagi
  kembali ke formulir daftar tanpa mengirim formulir.
- Cache diperiksa: tidak ada respons API atau HTML aplikasi.
- Tidak ditemukan page error atau overflow horizontal pada layar offline.

Browser uji menahan prompt native agar petunjuk bisa difoto. Kelayakan
pemasangan diperiksa terpisah melalui protokol Chrome. Screenshot dialog
bukan klaim bahwa pemasangan OS atau perangkat HP fisik sudah diuji.

Panduan: [pwa/README.md](../../pwa/README.md).
