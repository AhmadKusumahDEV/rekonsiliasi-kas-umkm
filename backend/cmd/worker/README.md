# cmd/worker

Entrypoint worker ditambahkan saat ada consumer atau job nyata. Jangan membuat
worker yang hanya idle tanpa fungsi. Saat worker pertama dibuat, gunakan pola
yang sama dengan `cmd/api`: load config, buat pool database, susun service, lalu
jalankan consumer dengan graceful shutdown.

Worker dan API dapat berbagi package `pkg/database` serta service yang sama,
tetapi proses deployment-nya terpisah. Consumer harus idempotent karena
message broker dapat mengirim ulang pesan.
