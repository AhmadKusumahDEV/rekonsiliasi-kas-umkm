# dist

Folder ini adalah lokasi hasil build backend. Isi folder `dist/` diabaikan Git,
tetapi README ini tetap di-commit agar lokasi artifact jelas.

## Hasil build lokal

```text
make build
  -> dist/<os>-<arch>/rekonsiliasi-kas-umkm-api[.exe]

make build-linux
  -> dist/linux-amd64/rekonsiliasi-kas-umkm-api

make build-linux-arm64
  -> dist/linux-arm64/rekonsiliasi-kas-umkm-api
```

Build Linux memakai `CGO_ENABLED=0`, sehingga binary dapat dipindahkan ke VPS
Linux tanpa mengandalkan library C lokal.

## Menjalankan di VPS

Dari root `backend`:

```bash
chmod +x dist/linux-amd64/rekonsiliasi-kas-umkm-api
./dist/linux-amd64/rekonsiliasi-kas-umkm-api
```

Jika ingin masuk ke folder binary, arahkan config secara eksplisit:

```bash
cd dist/linux-amd64
APP_CONFIG_FILE=../../configs/local.yaml ./rekonsiliasi-kas-umkm-api
```

## Artifact GitHub Actions

Workflow `Backend CI` mengunggah artifact bernama
`rekonsiliasi-kas-umkm-api-linux-amd64`. Download dengan GitHub CLI:

```bash
gh run download --name rekonsiliasi-kas-umkm-api-linux-amd64 -D dist/linux-amd64
cd dist/linux-amd64
chmod +x rekonsiliasi-kas-umkm-api
APP_CONFIG_FILE=../../configs/local.yaml ./rekonsiliasi-kas-umkm-api
```

Artifact hanya menyimpan binary selama 14 hari. Untuk deployment permanen,
bangun ulang di VPS atau gunakan release asset pada tag Git.
