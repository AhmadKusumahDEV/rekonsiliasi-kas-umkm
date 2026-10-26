# queries

Tulis query SQL di sini dengan anotasi sqlc, misalnya:

```sql
-- name: GetUserByID :one
SELECT id, name, created_at FROM users WHERE id = @id;
```

Gunakan named parameter `@snake_case`, tulis kolom secara eksplisit, dan hindari
`SELECT *`. Setelah query ditambahkan, jalankan `make gen.pg.models` dari root
`backend`. Hasil generate masuk ke
`internal/repository/rekonsiliasi_repository_sqlc` dan tidak boleh diedit manual.
