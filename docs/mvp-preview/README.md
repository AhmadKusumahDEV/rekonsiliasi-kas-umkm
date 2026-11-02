# Bukti integrasi MVP — 2 November 2026

Seluruh screenshot publik di folder ini memakai frontend lokal dengan
API fixture dan data buatan (Ayu / MVP Fixture), bukan data akun live.
Pengujian HTTPS terpisah dilakukan dengan izin baca-saja, tetapi screenshot
dan isi respons dari akun tersebut tidak dipublikasikan dalam repository.

- desktop-exceptions.png: halaman antrean pada 1440×1000.
- desktop-history.png: riwayat dengan tautan entitas pada 1440×1000.
- mobile-exceptions.png: antrean pada 375×1000.
- browser-fixtures.json: 11 skenario lokal sintetis, termasuk audit 130 event.
- https-ui-readonly.json: hasil pemeriksaan layout tanpa data transaksi/akun.
- live-api-readonly.json: status keberhasilan pemeriksaan endpoint, tanpa jumlah atau isi transaksi.

Analisis scope dan keterbatasan ada di [verifikasi MVP](../MVP_FRONTEND_VERIFICATION.md).
