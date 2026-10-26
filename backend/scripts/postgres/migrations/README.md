# migrations

Setiap perubahan schema memakai pasangan file bernomor enam digit:

```text
000001_create_users.up.sql
000001_create_users.down.sql
```

Buat dengan `make gen.pg.migration name=create_users`. Jangan mengubah migration
yang sudah pernah dijalankan di environment bersama. Folder ini menjadi schema
source untuk sqlc.
