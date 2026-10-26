# internal/worker

Satu file per consumer atau job. Worker menerima dependency yang sudah dirakit
di `cmd/worker`, selalu meneruskan context, dan harus idempotent karena broker
dapat mengirim ulang pesan.

Jangan memakai global state untuk pool database. Gunakan instance yang sama dari
composition root.
