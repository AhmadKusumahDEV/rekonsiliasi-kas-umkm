# Rekonsiliasi Kas UMKM

Monorepo untuk aplikasi rekonsiliasi kas UMKM.

- `backend/` — API Go dengan Echo, PostgreSQL/pgx, sqlc, dan Viper.
- `frontend/` — ruang kerja frontend yang belum diisi.

Backend saat ini berupa scaffold yang sudah bisa dijalankan. Panduan mulai,
struktur folder, dan aturan arsitektur ada di
[`backend/README.md`](backend/README.md).

CI backend berjalan lewat GitHub Actions pada perubahan di `backend/**`. Alur
branch, pull request, dan branch protection ada di
[`CONTRIBUTING.md`](CONTRIBUTING.md).

Dokumen produk dan teknis:

- [`docs/PRD.md`](docs/PRD.md)
- [`docs/FRD.md`](docs/FRD.md)
- [`backend/docs/PBL_QUERY_INVENTORY.md`](backend/docs/PBL_QUERY_INVENTORY.md)
