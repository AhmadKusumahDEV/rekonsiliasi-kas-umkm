## Ringkasan

Jelaskan masalah yang diselesaikan dan perilaku setelah perubahan ini.

## Perubahan

-

## Cara verifikasi

- [ ] `cd backend && go test ./...`
- [ ] `cd backend && go vet ./...`
- [ ] `cd backend && make lint`
- [ ] `cd backend && go build ./...`
- [ ] Docker build diuji bila Dockerfile berubah

## Database dan sqlc

- [ ] Tidak ada perubahan database, atau migration up/down sudah lengkap
- [ ] Query ditambahkan di `scripts/postgres/queries`
- [ ] `make gen.pg.models` dijalankan dan file generated ikut di-commit

## Risiko dan rollback

Jelaskan risiko utama dan langkah rollback bila perubahan perlu dibatalkan.

## Checklist

- [ ] Tidak ada secret atau credential di kode/config/fixture
- [ ] Tidak memakai `vendor` atau `-mod=vendor`
- [ ] Business logic dan transaksi tetap di `internal/service`
- [ ] Handler tetap tipis dan tidak mengakses sqlc langsung
- [ ] Dokumentasi diperbarui bila struktur atau perilaku berubah
