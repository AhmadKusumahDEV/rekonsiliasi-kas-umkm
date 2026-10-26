# Inventaris Query PBL untuk Rekonsiliasi Kas UMKM

> Status implementasi: schema usulan di bagian 4 sudah dibuat sebagai
> `000002_umkm_normalized`, backfill sebagai `000003_backfill_umkm_state`, dan
> query di bagian 5 sudah ditulis di `scripts/postgres/queries/auth.sql` serta
> `scripts/postgres/queries/umkm.sql`. Hasil sqlc ada di
> `internal/repository/rekonsiliasi_repository_sqlc`. Service auth/UMKM dan
> handler `/api/v1` untuk alur inti sudah memakai query ini; SSE events, worker,
> dan frontend belum. Dokumen ini tetap menjadi inventaris perilaku prototipe.

Dokumen ini menelusuri aksi halaman Temu Kas pada prototipe **Tiga-Prototipe-PBL** sampai ke route HTTP, handler Go, aturan bisnis, dan persistence PostgreSQL. Prototipe diperlakukan read-only. Schema domain prototipe hanya berasal dari **internal/platform/store.go**; data usaha disimpan di **pblg_workspaces.state** sebagai JSONB.

Sumber utama: **internal/platform/web/app.js**, **design.js**, **server.go**, **auth.go**, **store.go**, **umkm.go**, **jobs.go**, **models.go**, **demo.go**, serta test di **internal/platform**. Baseline backend saat ini ada di **backend/scripts/postgres/migrations/000001_pbl_baseline.up.sql**.

## 1. Ringkasan

Ditemukan **7 halaman UMKM** (#overview, #entries, #reconcile, #imports, #history, #reports, #settings) dan **2 mode halaman auth** (#login, #register), sehingga 9 tampilan. Ada **16 route HTTP** yang relevan untuk UMKM dan operasional bersama; 15 dipakai UI, sedangkan **GET /api/health** hanya operasional. Tabel aksi di bawah memuat **26 baris aksi**, termasuk aksi yang murni berjalan di browser.

Perbedaan lapisan yang penting:

- **UI** memakai hash route, modal, pencarian/filter/pagination di browser, dan menghitung saldo/metrik dari state yang sudah dimuat. Banyak tombol hanya membuka modal dan belum memanggil API (app.js:379-399, 842-876, 1223-1361).
- **API** mengirim seluruh state workspace dalam satu respons **GET /api/state**, lalu **GET /api/suggestions** untuk UMKM. Tidak ada endpoint pagination, filter, atau detail transaksi (auth.go:205-248; app.js:764-791).
- **Aturan bisnis** ada di Go: parsing CSV, validasi, deduplikasi impor, usulan pasangan, aritmetika alokasi, pembatalan, dan ekspor (umkm.go:27-433).
- **SQL persistence** hanya mencakup workspace, user, sesi, audit, job, dan foto. Entries, imports, dan allocations tidak punya tabel; semuanya elemen array di **pblg_workspaces.state**. Mutasi domain memakai satu transaksi dengan **SELECT ... FOR UPDATE** pada baris workspace, lalu menulis state, menaikkan version, dan menulis audit (store.go:87-118).

Konsekuensinya, satu aksi halaman UMKM biasanya hanya memicu satu atau dua query generik, bukan query per entity. Normalisasi baru diperlukan agar query sqlc punya tabel dan constraint yang jelas.

## 2. Peta halaman ke API dan query

Kolom **Current persistence** menyebut lokasi data hari ini. Kolom **Proposed sqlc** menyebut nama query target; anotasi dan parameter ada di bagian 5. Tanda **—** berarti tidak ada request atau query.

### 2.1 Auth dan operasi bersama

| Page/hash route | UI action/button/form | HTTP method dan path | Go handler/function | Data read | Data written | Current persistence | Proposed sqlc query name(s) | Evidence (file:line) |
|---|---|---|---|---|---|---|---|---|
| Semua halaman (boot) | init(): muat config lalu cek sesi | GET /api/config; GET /api/me | closure GET /api/config; closure GET /api/me | brand/tagline/roles statis; user + csrf dari sesi | — | in-memory config; pblg_sessions + pblg_users | GetSessionUser | app.js:1447-1462; server.go:184-186,207-213 |
| Semua halaman | Muat ulang data setelah login, mutasi, atau tombol refresh | GET /api/state | state (auth.go:205) | workspace, entries, imports, allocations, users, 120 audit | — | pblg_workspaces.state JSONB; pblg_users; pblg_audit | GetWorkspaceForUser; ListUsersByWorkspace; ListAuditEventsByWorkspace; ListUmkmEntriesByWorkspace; ListUmkmImportsByWorkspace; ListUmkmAllocationsByWorkspace | app.js:764-791; auth.go:205-248 |
| Semua halaman | EventSource otomatis | GET /api/events | events (server.go:266) | version workspace tiap 4 detik; sesi diperiksa ulang | — | pblg_workspaces.version; pblg_sessions | GetWorkspaceVersion; GetSessionUser | app.js:140-182; server.go:266-306 |
| #login | Submit form Masuk | POST /api/login | login (auth.go:135) | user + password_hash berdasarkan app dan email | baris sesi baru | pblg_users; pblg_sessions | GetUserForLogin; CreateSession | app.js:801,1047-1066; auth.go:135-154 |
| #register (owner) | Submit Daftar akun; role owner; isi nama usaha | POST /api/register | register (auth.go:77) | — | workspace, user, sesi | pblg_workspaces; pblg_users; pblg_sessions | CreateWorkspace; CreateUser; CreateSession | app.js:803-810,1047-1066; auth.go:93-133 |
| #register (join) | Isi kode bergabung; shared tetapi tidak terjangkau UMKM | POST /api/register | register (auth.go:109-113) | workspace berdasarkan app + code | user, sesi | pblg_workspaces; pblg_users; pblg_sessions | GetWorkspaceByJoinCode; CreateUser; CreateSession | app.js:809,1394-1399; auth.go:109-115 |
| Header / #settings | Keluar akun | POST /api/logout | logout (auth.go:155) | cookie sesi | hapus satu sesi | pblg_sessions | DeleteSessionByTokenHash | app.js:727,1242-1250; auth.go:155-164 |
| #settings | Ganti kata sandi | POST /api/password | changePassword (auth.go:165) | password_hash lama | hash baru; hapus semua sesi akun | pblg_users; pblg_sessions | GetUserPasswordHash; UpdateUserPasswordHash; DeleteSessionsByUserID | app.js:1181-1191,1286-1290; auth.go:165-203 |
| #reports | Unduh rekap CSV | GET /api/export | export (umkm.go:388) | entries + allocations; sisa dihitung ulang | hanya respons CSV | pblg_workspaces.state | ListUmkmEntriesByWorkspace; ListUmkmAllocationsByWorkspace | app.js:684; jobs.go:246; umkm.go:388-433 |
| Operasional (tidak dipakai UI) | Cek kesehatan | GET /api/health | closure GET /api/health | ping koneksi PostgreSQL | — | koneksi database | — (db.Ping, bukan sqlc) | server.go:187-194 |

### 2.2 Halaman UMKM

| Page/hash route | UI action/button/form | HTTP method dan path | Go handler/function | Data read | Data written | Current persistence | Proposed sqlc query name(s) | Evidence (file:line) |
|---|---|---|---|---|---|---|---|---|
| #overview | Buka ringkasan; metrik, 6 tagihan terbuka, total per kanal | GET /api/state; GET /api/suggestions | state; closure suggestions (jobs.go:254) | entries, allocations, imports, users, audit; kandidat pasangan | — | JSONB state | state queries + ListUmkmUnallocatedEntries | design.js:64-100; app.js:283-310,764-776 |
| #overview | Tombol Cocokkan pada tagihan | — (client) | allocateModal (app.js:842) | entries + saldo dari state | — | in-memory | — | design.js:76; app.js:1253 |
| #overview, #entries, #imports | Tombol Impor CSV | — (client) | importModal (app.js:828) | — | — | in-memory | — | design.js:89; app.js:424,491,1252 |
| #entries | Buka daftar; cari, filter, ukuran halaman, navigasi halaman | GET /api/state | state | entries + allocations | — | JSONB state | ListUmkmEntriesByWorkspace; ListUmkmAllocationsByWorkspace | app.js:379-436 |
| #reconcile | Buka usulan pasangan dan riwayat keputusan | GET /api/state; GET /api/suggestions | state; closure suggestions | entries, allocations, kandidat | — | JSONB state | state queries + ListUmkmUnallocatedEntries | app.js:446-463,764-776 |
| #reconcile | Periksa pasangan atau Alokasi manual | — (client) | allocateModal (app.js:842) | suggestion/entries dari state | — | in-memory | — | app.js:452,455,1253-1257 |
| #reconcile modal | Simpan pasangan | POST /api/allocations | allocate (umkm.go:277) | entries sale/receipt; alokasi aktif; alokasi berdasarkan key | allocation baru; audit; version | JSONB allocations; pblg_audit; pblg_workspaces.version | LockWorkspaceForUpdate; GetUmkmAllocationByID; GetUmkmEntriesByIDs; ListUmkmActiveAllocationsByEntryIDs; CreateUmkmAllocation; InsertAuditEvent; BumpWorkspaceVersion | app.js:858-875,1090-1101; umkm.go:277-316; store.go:87-118 |
| #reconcile riwayat | Batalkan pasangan | POST /api/allocations/{id}/undo | undo (umkm.go:317) | allocation berdasarkan id | reversed=true; audit; version | JSONB allocations; pblg_audit; version | LockWorkspaceForUpdate; MarkUmkmAllocationReversed; InsertAuditEvent; BumpWorkspaceVersion | app.js:444,1340-1350; umkm.go:317-331 |
| #imports | Buka riwayat impor | GET /api/state | state | imports + entries | — | JSONB imports | ListUmkmImportsByWorkspace; ListUmkmEntriesByWorkspace | app.js:464-496 |
| #imports modal | Periksa CSV | POST /api/imports/preview | previewImport (umkm.go:204) | teks CSV; tidak membaca database | — (tidak menyimpan) | in-memory ParseCSV | — | app.js:828-839,1068-1088; umkm.go:27-89,204-218 |
| #imports modal | Simpan impor ini | POST /api/imports | commitImport (umkm.go:219) | import berdasarkan checksum; entries berdasarkan source ID; jumlah entries | import, entries; audit; version; duplikat tidak menambah baris | JSONB imports/entries; pblg_audit; version | LockWorkspaceForUpdate; GetUmkmImportByChecksum; CountUmkmEntries; ListUmkmEntriesByKindAndSourceIDs; CreateUmkmImport; CreateUmkmEntry; InsertAuditEvent; BumpWorkspaceVersion | app.js:1325-1339; umkm.go:219-276 |
| #imports baris | Batalkan impor | POST /api/imports/{id}/cancel | cancelImport (umkm.go:332) | import; entries; alokasi aktif | import dan entries ditandai batal; audit; version | JSONB imports/entries; pblg_audit; version | LockWorkspaceForUpdate; GetUmkmImportByID; CountUmkmActiveAllocationsForImport; CancelUmkmImport; CancelUmkmImportEntries; InsertAuditEvent; BumpWorkspaceVersion | app.js:483,1351-1361; umkm.go:332-364 |
| #history | Buka riwayat aktivitas | GET /api/state | state | 120 audit terbaru | — | pblg_audit | ListAuditEventsByWorkspace | app.js:603-623; auth.go:214-217 |
| #reports | Buka ringkasan keputusan dan tautan ekspor | GET /api/state | state | entries, allocations | — | JSONB state | ListUmkmEntriesByWorkspace; ListUmkmAllocationsByWorkspace | app.js:641-689 |
| #settings | Buka info akun dan ruang kerja | GET /api/state; GET /api/me | state; closure /api/me | user, workspace | — | pblg_users; pblg_workspaces | GetWorkspaceForUser; GetSessionUser | app.js:721-729 |
| Header semua halaman | Data baru · muat ulang | GET /api/state; GET /api/suggestions | state; closure suggestions | seluruh state + kandidat | — | JSONB state | state queries + ListUmkmUnallocatedEntries | app.js:239,1241,764-791 |

Catatan kontrak respons: preview mengembalikan 200; login 200; register 201; impor baru 201; impor duplikat tetap 201 dengan **duplicate:true**; alokasi 201 termasuk retry idempoten; undo dan cancel 200; validasi 400; sesi/CSRF/origin 401/403; konflik 409; tidak ditemukan 404 (umkm.go:219-331; integration_test.go:101-145).

## 3. Query persistence yang sudah ada

### 3.1 Read statements

| Kegunaan | SQL aktual | Lokasi | Catatan |
|---|---|---|---|
| Ambil user + csrf dari token sesi | SELECT u.id,u.workspace_id,u.app,u.name,u.email,u.role,s.csrf FROM pblg_sessions s JOIN pblg_users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.app=$2 | auth.go:59 | Dipakai setiap endpoint privat; /api/me memanggil session lagi di handler |
| Ambil user untuk login | SELECT id,workspace_id,app,name,email,role,password_hash FROM pblg_users WHERE app=$1 AND email=$2 | auth.go:143 | Login tidak memakai role dari body |
| Ambil hash password | SELECT password_hash FROM pblg_users WHERE id=$1 | auth.go:177 | Untuk ganti kata sandi |
| Cari workspace dari join code | SELECT id FROM pblg_workspaces WHERE app=$1 AND code=$2 | auth.go:110 | Jalur non-owner; tidak terjangkau UMKM |
| Cek user berhasil dibuat | SELECT EXISTS(SELECT 1 FROM pblg_users WHERE id=$1) | auth.go:120 | Setelah INSERT ON CONFLICT DO NOTHING |
| Baca workspace + state | SELECT id,name,code,version,state FROM pblg_workspaces WHERE id=$1 AND app=$2 | store.go:77 | State di-unmarshal ke struct State |
| Daftar user workspace | SELECT id,workspace_id,app,name,email,role FROM pblg_users WHERE workspace_id=$1 AND app=$2 ORDER BY name | store.go:120 | Untuk /api/state |
| Daftar audit | SELECT id,actor,action,detail,created_at FROM pblg_audit WHERE workspace_id=$1 ORDER BY id DESC LIMIT 120 | store.go:136 | Untuk /api/state dan #history |
| Versi workspace untuk SSE | SELECT version FROM pblg_workspaces WHERE id=$1 | server.go:280 | Polling 4 detik; hanya versi, bukan data privat |

### 3.2 Write statements dan transaksi

| Kegunaan | SQL aktual | Lokasi | Transaksi/locking |
|---|---|---|---|
| Kunci workspace dan baca state | SELECT state FROM pblg_workspaces WHERE id=$1 AND app=$2 FOR UPDATE | store.go:94 | Awal Mutate; mengunci satu baris workspace |
| Tulis state dan naikkan versi | UPDATE pblg_workspaces SET state=$1,version=version+1 WHERE id=$2 | store.go:109 | Dalam Mutate |
| Tulis audit mutasi | INSERT INTO pblg_audit(workspace_id,actor,action,detail) VALUES($1,$2,$3,$4) | store.go:113 | Dalam Mutate, setelah update state |
| Buat sesi | INSERT INTO pblg_sessions(token_hash,user_id,csrf,expires_at) VALUES($1,$2,$3,$4) | auth.go:70 | Di luar transaksi register; login juga tanpa transaksi |
| Buat workspace | INSERT INTO pblg_workspaces(id,app,name,code) VALUES($1,$2,$3,$4) | auth.go:105 | Dalam transaksi register |
| Buat user | INSERT INTO pblg_users(...,role) VALUES($1,...,$7) ON CONFLICT(app,email) DO NOTHING | auth.go:115 | Dalam transaksi register |
| Hapus sesi saat logout | DELETE FROM pblg_sessions WHERE token_hash=$1 | auth.go:158 | Tanpa transaksi |
| Update hash password | UPDATE pblg_users SET password_hash=$1 WHERE id=$2 | auth.go:193 | Dalam transaksi ganti password |
| Hapus semua sesi akun | DELETE FROM pblg_sessions WHERE user_id=$1 | auth.go:196 | Dalam transaksi ganti password |

Pola transaksi domain: **BeginTx -> SELECT state FOR UPDATE -> callback bisnis -> marshal state -> UPDATE state/version -> INSERT audit -> Commit** (store.go:87-118). Semua mutasi UMKM memakai pola ini: commitImport (umkm.go:229), allocate (umkm.go:293), undo (umkm.go:318), cancelImport (umkm.go:333). Tidak ada network call di dalam lock; komentar store.go:86 menyatakan lock hanya untuk mutasi finansial/workflow.

Pola register: transaksi membuat workspace/user, commit, lalu membuat sesi di luar transaksi (auth.go:93-133). Pola ganti password: transaksi update hash dan hapus semua sesi, commit, lalu logout menghapus sesi saat ini lagi (auth.go:188-202). Login hanya SELECT user lalu INSERT sesi.

Dua pola yang mudah terlewat:

1. Mutasi idempoten tetap menulis state/version dan audit. Jika checksum impor sudah ada, callback mengembalikan tanpa perubahan data, tetapi Mutate tetap menjalankan UPDATE version+1 dan INSERT audit (umkm.go:231-239; store.go:109-116).
2. Tidak ada unique constraint domain di SQL. Deduplikasi checksum, source ID, dan key alokasi seluruhnya dilakukan di Go atas array JSONB, meskipun terlindungi oleh workspace lock.

Schema runtime prototipe ada di store.go:17-26: pblg_workspaces, pblg_users, pblg_sessions, pblg_jobs, pblg_audit, pblg_photos dan dua index. Seed startup juga menulis workspace/user/state (auth.go:249-303; demo.go:15-39), tetapi bukan aksi halaman.

## 4. Kebutuhan schema untuk sqlc

Prinsip: jangan mengubah migration **000001**. Tabel **pblg_workspaces**, **pblg_users**, **pblg_sessions**, dan **pblg_audit** tetap dipakai untuk identitas, sesi, workspace, dan audit. Tambahkan migration baru, misalnya **000002_umkm_normalized**, yang membuat tabel domain **umkm_imports**, **umkm_entries**, dan **umkm_allocations**.

### 4.1 DDL minimum

~~~sql
CREATE TABLE umkm_imports (
    id           text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
    workspace_id text NOT NULL REFERENCES pblg_workspaces(id) ON DELETE CASCADE,
    kind         text NOT NULL CHECK (kind IN ('sale', 'receipt', 'contoh')),
    checksum     text NOT NULL,
    row_count    integer NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    created_at   timestamptz NOT NULL DEFAULT now(),
    canceled_at  timestamptz,
    UNIQUE (workspace_id, checksum)
);

CREATE TABLE umkm_entries (
    id           text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
    workspace_id text NOT NULL REFERENCES pblg_workspaces(id) ON DELETE CASCADE,
    batch_id     text NOT NULL REFERENCES umkm_imports(id) ON DELETE CASCADE,
    source_id    text NOT NULL CHECK (btrim(source_id) <> ''),
    kind         text NOT NULL CHECK (kind IN ('sale', 'receipt')),
    occurred_on  date NOT NULL,
    reference    text NOT NULL DEFAULT '',
    name         text NOT NULL CHECK (btrim(name) <> ''),
    amount       bigint NOT NULL CHECK (amount > 0 AND amount <= 1000000000000),
    channel      text NOT NULL CHECK (btrim(channel) <> ''),
    note         text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    canceled_at  timestamptz,
    UNIQUE (workspace_id, kind, source_id)
);

CREATE TABLE umkm_allocations (
    id               text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{32}$'),
    workspace_id     text NOT NULL REFERENCES pblg_workspaces(id) ON DELETE CASCADE,
    sale_entry_id    text NOT NULL REFERENCES umkm_entries(id),
    receipt_entry_id text NOT NULL REFERENCES umkm_entries(id),
    amount           bigint NOT NULL CHECK (amount > 0 AND amount <= 1000000000000),
    fee              bigint NOT NULL DEFAULT 0 CHECK (fee >= 0 AND fee <= 1000000000000),
    reason           text NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL DEFAULT now(),
    reversed_at      timestamptz,
    CHECK (sale_entry_id <> receipt_entry_id)
);

CREATE INDEX umkm_imports_workspace_created_idx
    ON umkm_imports (workspace_id, created_at DESC);

CREATE INDEX umkm_entries_workspace_date_idx
    ON umkm_entries (workspace_id, occurred_on, id);

CREATE INDEX umkm_entries_match_idx
    ON umkm_entries (workspace_id, kind, channel, lower(reference), amount, occurred_on)
    WHERE canceled_at IS NULL;

CREATE INDEX umkm_entries_batch_idx
    ON umkm_entries (batch_id);

CREATE INDEX umkm_allocations_active_sale_idx
    ON umkm_allocations (sale_entry_id)
    WHERE reversed_at IS NULL;

CREATE INDEX umkm_allocations_active_receipt_idx
    ON umkm_allocations (receipt_entry_id)
    WHERE reversed_at IS NULL;

CREATE INDEX umkm_allocations_workspace_created_idx
    ON umkm_allocations (workspace_id, created_at DESC);

-- Index tambahan pada baseline; bukan perubahan definisi tabel 000001.
CREATE INDEX IF NOT EXISTS pblg_sessions_user_idx ON pblg_sessions (user_id);
CREATE INDEX IF NOT EXISTS pblg_sessions_expiry_idx ON pblg_sessions (expires_at);
CREATE INDEX IF NOT EXISTS pblg_users_workspace_app_idx ON pblg_users (workspace_id, app, name);
~~~

Alasan tipe dan constraint:

- **bigint Rupiah**: prototipe menyimpan **int64** dan memvalidasi nominal positif tanpa desimal, maksimal 1 triliun (models.go:43; umkm.go:76-79,172-173). Tidak ada sen atau mata uang lain.
- **timestamptz + date**: tanggal transaksi adalah string YYYY-MM-DD (models.go:40; umkm.go:73-75), sedangkan created/canceled/reversed adalah waktu. Gunakan **occurred_on date** untuk tanggal bisnis.
- **canceled_at / reversed_at**: status diturunkan dari timestamp, bukan boolean terpisah. Aktif berarti **canceled_at IS NULL** atau **reversed_at IS NULL**. Ini menghindari dua sumber kebenaran.
- **idempotency**: checksum impor unik per workspace; **umkm_allocations.id** adalah key 32 hex dari klien (umkm.go:289-309) sehingga primary key sekaligus idempotency key. **umkm_entries** memakai natural key **(workspace_id, kind, source_id)**.
- **workspace lock**: semua mutasi tetap diawali **LockWorkspaceForUpdate** agar validasi saldo, cek duplikat, dan cancel tidak balapan, meniru lock store.go:94.
- **version**: **pblg_workspaces.version** tetap dinaikkan dalam transaksi yang sama agar SSE (server.go:280-290) tetap bekerja.

### 4.2 Pemetaan JSONB ke tabel

| Sumber JSONB | Tabel/kolom target | Catatan |
|---|---|---|
| state.entries[].id | umkm_entries.id | ID 32 hex |
| state.entries[].source_id | umkm_entries.source_id | Unik per workspace + kind |
| state.entries[].batch_id | umkm_entries.batch_id | FK ke umkm_imports |
| state.entries[].kind | umkm_entries.kind | sale atau receipt |
| state.entries[].date | umkm_entries.occurred_on | date |
| state.entries[].reference | umkm_entries.reference | Boleh kosong; kosong tidak diusulkan |
| state.entries[].name | umkm_entries.name | Wajib |
| state.entries[].amount | umkm_entries.amount | bigint Rupiah |
| state.entries[].channel | umkm_entries.channel | Huruf kecil dari CSV |
| state.entries[].note | umkm_entries.note | Boleh kosong |
| state.entries[].canceled | umkm_entries.canceled_at | Boolean menjadi timestamp |
| state.imports[].id | umkm_imports.id | ID 32 hex |
| state.imports[].kind | umkm_imports.kind | sale, receipt, atau contoh |
| state.imports[].checksum | umkm_imports.checksum | Idempotency impor |
| state.imports[].rows | umkm_imports.row_count | Jumlah baris baru, bukan total file |
| state.imports[].created_at | umkm_imports.created_at | timestamptz |
| state.imports[].canceled | umkm_imports.canceled_at | Boolean menjadi timestamp |
| state.allocations[].id | umkm_allocations.id | Key klien 32 hex |
| state.allocations[].sale_id | umkm_allocations.sale_entry_id | FK entry sale |
| state.allocations[].receipt_id | umkm_allocations.receipt_entry_id | FK entry receipt |
| state.allocations[].amount | umkm_allocations.amount | Mengurangi saldo kedua sisi |
| state.allocations[].fee | umkm_allocations.fee | Hanya mengurangi saldo penjualan |
| state.allocations[].reason | umkm_allocations.reason | Wajib minimal 5 karakter jika fee > 0 |
| state.allocations[].created_at | umkm_allocations.created_at | timestamptz |
| state.allocations[].reversed | umkm_allocations.reversed_at | Boolean menjadi timestamp |
| state.assignments/submissions/tickets | tidak dimigrasikan | Milik Tilik/Ruang, di luar slice UMKM |
| pblg_workspaces.version | tetap di pblg_workspaces | Untuk SSE, bukan data domain |

### 4.3 Batas desain yang perlu disadari

- FK sederhana tidak bisa memaksa **sale_entry_id** benar-benar berjenis sale dan **receipt_entry_id** berjenis receipt. Pemeriksaan kind tetap di Go, atau tambahkan trigger/composite FK bila nanti diperlukan.
- **umkm_imports.kind = 'contoh'** dipertahankan karena seed prototipe membuat satu batch contoh berisi campuran sale dan receipt dengan Rows=8 (demo.go:20-28; app.js:473-477). Ini bukan perilaku impor nyata.
- Backfill dari state JSONB sebaiknya menjadi migration terpisah, bukan bagian DDL. Jangan menebak data yang tidak ada; backfill harus memetakan array secara eksplisit dan memverifikasi jumlah baris.
- **pblg_workspaces.state** tetap ada untuk kompatibilitas, tetapi untuk UMKM sebaiknya menjadi legacy/read-only setelah cutover. Jangan dual-write tanpa keputusan eksplisit karena dua sumber kebenaran memperbesar drift.

## 5. Daftar query sqlc yang harus dibuat

Nama di bawah mengikuti gaya Go sqlc. Parameter memakai named parameter **@snake_case**; array PostgreSQL memakai tipe **[]string**. Return shape menyebut field penting, bukan seluruh kolom.

### 5.1 Auth, sesi, workspace, dan audit

| Query | Annotation | Parameter | Return shape | Dipakai oleh |
|---|---|---|---|---|
| GetSessionUser | :one | @token_hash, @app | user_id, workspace_id, app, name, email, role, csrf | Semua endpoint privat; /api/me; cek SSE |
| GetUserForLogin | :one | @app, @email | id, workspace_id, app, name, email, role, password_hash | POST /api/login |
| CreateSession | :exec | @token_hash, @user_id, @csrf, @expires_at | — | Login dan register |
| DeleteSessionByTokenHash | :execrows | @token_hash | rows affected | POST /api/logout |
| GetUserPasswordHash | :one | @user_id | password_hash | POST /api/password |
| UpdateUserPasswordHash | :execrows | @user_id, @password_hash | rows affected | POST /api/password |
| DeleteSessionsByUserID | :execrows | @user_id | rows affected | POST /api/password |
| CreateWorkspace | :one | @id, @app, @name, @code | id, version | Register owner |
| GetWorkspaceByJoinCode | :one | @app, @code | id | Register join; shared, tidak terjangkau UMKM |
| CreateUser | :execrows | @id, @workspace_id, @app, @name, @email, @password_hash, @role | rows affected | Register |
| GetWorkspaceForUser | :one | @workspace_id, @app | id, name, code, version | GET /api/state; export |
| ListUsersByWorkspace | :many | @workspace_id, @app | id, workspace_id, app, name, email, role | GET /api/state |
| ListAuditEventsByWorkspace | :many | @workspace_id, @limit | id, actor, action, detail, created_at | GET /api/state; #history |
| GetWorkspaceVersion | :one | @workspace_id | version | GET /api/events |
| InsertAuditEvent | :exec | @workspace_id, @actor, @action, @detail | — | Setiap mutasi domain |
| BumpWorkspaceVersion | :execrows | @workspace_id | rows affected | Setiap mutasi domain; SSE |
| LockWorkspaceForUpdate | :one | @workspace_id, @app | version | Awal setiap transaksi tulis UMKM |

### 5.2 UMKM

| Query | Annotation | Parameter | Return shape | Dipakai oleh |
|---|---|---|---|---|
| ListUmkmEntriesByWorkspace | :many | @workspace_id | seluruh kolom entry | GET /api/state; #entries; #reports; export |
| ListUmkmImportsByWorkspace | :many | @workspace_id | seluruh kolom import | GET /api/state; #imports |
| ListUmkmAllocationsByWorkspace | :many | @workspace_id | seluruh kolom allocation | GET /api/state; #reconcile; #reports; export |
| ListUmkmUnallocatedEntries | :many | @workspace_id | entry aktif, reference tidak kosong, tanpa alokasi aktif | GET /api/suggestions |
| GetUmkmImportByChecksum | :one | @workspace_id, @checksum | import | POST /api/imports; deteksi duplikat |
| GetUmkmImportByID | :one | @workspace_id, @id | import | POST /api/imports/{id}/cancel |
| CountUmkmEntries | :one | @workspace_id | bigint jumlah entry | POST /api/imports; batas 10.000 |
| ListUmkmEntriesByKindAndSourceIDs | :many | @workspace_id, @kind, @source_ids []string | entry yang sudah ada | POST /api/imports; cek source ID |
| CreateUmkmImport | :one | @id, @workspace_id, @kind, @checksum, @row_count | import | POST /api/imports |
| CreateUmkmEntry | :one | @id, @workspace_id, @batch_id, @source_id, @kind, @occurred_on, @reference, @name, @amount, @channel, @note | entry | POST /api/imports; satu insert per baris baru |
| GetUmkmAllocationByID | :one | @workspace_id, @id | allocation | POST /api/allocations; retry idempoten |
| GetUmkmEntriesByIDs | :many | @workspace_id, @ids []string | entry sale/receipt | POST /api/allocations; validasi kind dan saldo |
| ListUmkmActiveAllocationsByEntryIDs | :many | @workspace_id, @entry_ids []string | alokasi aktif yang menyentuh entry | POST /api/allocations; hitung sisa |
| CreateUmkmAllocation | :one | @id, @workspace_id, @sale_entry_id, @receipt_entry_id, @amount, @fee, @reason | allocation | POST /api/allocations |
| MarkUmkmAllocationReversed | :execrows | @workspace_id, @id | rows affected | POST /api/allocations/{id}/undo |
| CountUmkmActiveAllocationsForImport | :one | @workspace_id, @import_id | bigint jumlah alokasi aktif | POST /api/imports/{id}/cancel |
| CancelUmkmImport | :execrows | @workspace_id, @id | rows affected | POST /api/imports/{id}/cancel |
| CancelUmkmImportEntries | :execrows | @workspace_id, @import_id | rows affected | POST /api/imports/{id}/cancel |

Unit of work yang disarankan sama untuk semua mutasi UMKM: **BeginTx -> LockWorkspaceForUpdate -> baca/validasi dengan query di atas -> tulis -> InsertAuditEvent -> BumpWorkspaceVersion -> Commit**. Validasi saldo memakai **GetUmkmEntriesByIDs** dan **ListUmkmActiveAllocationsByEntryIDs**, bukan satu query agregasi yang menyembunyikan aturan bisnis. Untuk import, panggil **GetUmkmImportByChecksum** lebih dulu, lalu **CountUmkmEntries** dan **ListUmkmEntriesByKindAndSourceIDs**, lalu **CreateUmkmImport** dan **CreateUmkmEntry** per baris baru.

## 6. Aturan bisnis yang harus tetap di Go

- **Validasi CSV**: UTF-8 valid; BOM dibuang; header tepat **id,tanggal,referensi,nama,nominal,kanal,catatan**; 7 kolom; maksimal 1 MB dan 500 baris; setiap sel maksimal 400 karakter tanpa NUL; ID sumber wajib, maksimal 80 karakter, dan unik di dalam file; tanggal YYYY-MM-DD; nominal integer positif maksimal 1 triliun; nama dan kanal wajib (umkm.go:27-89; domain_test.go:10-48).
- **Deduplikasi impor**: checksum SHA-256 dari kind + teks CSV; checksum sama dan belum batal berarti duplicate tanpa baris baru; checksum sama tetapi sudah batal berarti konflik; source ID sama dengan isi identik dilewati; source ID sama dengan isi berbeda atau sudah batal berarti konflik seluruh file; total entries maksimal 10.000 (umkm.go:219-276; integration_test.go:101-137).
- **Usulan pasangan**: hanya entry aktif, reference tidak kosong, dan belum punya alokasi aktif; kunci referensi lowercase + kanal + nominal; penerimaan harus 0 sampai 3 hari setelah penjualan; kandidat harus tunggal di kedua arah; usulan tidak menulis apa pun (umkm.go:90-170; domain_test.go:83-107).
- **Alokasi**: amount > 0, fee >= 0, keduanya maksimal 1 triliun; sale dan receipt harus ada, kind sesuai, dan belum dibatalkan; amount + fee tidak boleh melebihi sisa penjualan; amount tidak boleh melebihi sisa penerimaan; fee > 0 mewajibkan alasan minimal 5 karakter; alasan maksimal 400 karakter tanpa NUL; key wajib 32 hex dan payload retry harus identik (umkm.go:171-203,277-316; domain_test.go:52-82).
- **Aritmetika saldo**: sisa penjualan = amount - (amount alokasi + fee) untuk semua alokasi aktif; sisa penerimaan = amount - amount alokasi aktif. Fee tidak menambah uang diterima (umkm.go:93-110; app.js:259-282).
- **Undo**: jangan hapus alokasi; set reversed dan kembalikan saldo; ID tidak ada berarti 404; pengulangan undo tetap 200. Audit tetap ditulis (umkm.go:317-331; app.js:1340-1350).
- **Cancel import**: tolak jika ada alokasi aktif pada entry batch; tandai import dan semua entry batal; jangan hapus audit; pengulangan cancel tetap 200 (umkm.go:332-364; integration_test.go:134-137).
- **Ekspor**: sertakan entry aktif dan batal, hitung sisa alokasi, netralkan sel yang diawali =, +, -, atau @ agar tidak menjadi formula spreadsheet, dan tulis BOM UTF-8. Ini rekap, bukan laba (umkm.go:365-433; domain_test.go:108-117).
- **Auth/sesi**: bcrypt cost 12; password 12-72 byte; email valid; role harus diizinkan app; role login tidak boleh diambil dari body; cookie HttpOnly/SameSite Strict, 12 jam; CSRF dibandingkan constant-time untuk non-GET; origin cross-site ditolak; rate limit login/register 25 per 10 menit (auth.go:26-76,135-203; server.go:147-181,238-264).
- **Isolasi workspace**: workspace dan user selalu berasal dari sesi, bukan parameter klien; semua query domain memakai workspace_id dari sesi. /api/state menyembunyikan kode workspace untuk non-owner (auth.go:205-248; integration_test.go:101-145).
- **Audit**: teks aksi/detail berasal dari handler domain; actor adalah nama user, bukan ID. Mutasi domain menulis audit dalam transaksi yang sama (store.go:113; umkm.go:229,293,318,333).
- **Waktu dan uang**: gunakan Rupiah integer, tanpa pecahan, tanpa nilai negatif/refund. Prototipe memakai UTC untuk created_at import/allocation; backend baru wajib memakai **utils.NowJakarta()** atau **pkg/utils.Jakarta()** sesuai AGENTS.md:32, sehingga kebijakan waktu harus diputuskan eksplisit.

## 7. Gap dan keputusan yang belum pasti

1. **Tidak ada schema domain di prototipe.** Hanya **store.go:17-26** yang menjadi sumber schema; entries/imports/allocations adalah elemen JSONB tanpa tipe SQL, foreign key, atau unique constraint. DDL di atas adalah usulan, bukan hasil pembacaan migration lama.
2. **Jalur join code tidak terjangkau untuk UMKM.** UI menyembunyikan join code untuk owner, dan server hanya mengizinkan role **owner** untuk app umkm (app.js:803-810,1394-1399; server.go:310; auth.go:40-43,98-114). Route register tetap shared, tetapi tidak ada alur UMKM yang sah memakainya.
3. **Seed contoh bukan impor nyata.** Satu import ber-kind **contoh** berisi campuran sale dan receipt dengan Rows=8 (demo.go:20-28). Schema harus mengakomodasi nilai ini atau memigrasikan seed secara khusus; jangan menganggap semua batch satu kind.
4. **Tidak ada API pagination/filter/detail.** UI memfilter dan memotong halaman di browser dari seluruh state (app.js:379-399). Jika backend mempertahankan kontrak ini, **GET /api/state** bisa mengirim sampai 10.000 entries. Jika backend menambah endpoint pagination, itu endpoint baru, bukan hasil tracing prototipe.
5. **Jumlah baris yang dilewati tidak disimpan.** Import hanya menyimpan **rows** berisi jumlah baris baru (umkm.go:264; models.go:48-55). Respons menyebut **skipped**, tetapi nilainya tidak persisten. Jangan menambahkan kolom skipped tanpa keputusan produk.
6. **Retry idempoten tetap menaikkan version dan menulis audit.** Mutate selalu UPDATE state/version dan INSERT audit meski callback tidak mengubah data (store.go:109-116; umkm.go:231-239). Apakah perilaku ini dipertahankan setelah normalisasi belum diputuskan; menghilangkannya mengubah sinyal SSE dan isi audit.
7. **Audit menyimpan nama, bukan user ID.** Actor adalah **u.Name** (store.go:113). Perubahan nama atau nama ganda membuat audit sulit dikaitkan ke akun. Menambah kolom user_id/entity_id adalah perubahan schema baru, bukan kebutuhan minimum.
8. **Auth tidak diaudit.** Login, register, logout, dan ganti password tidak menulis pblg_audit. Jika kebutuhan kepatuhan mencakup auth, ini gap produk/keamanan.
9. **State dan export tidak dibaca dalam satu snapshot.** **GET /api/state** menjalankan tiga query terpisah (workspace/state, users, audit) tanpa transaksi (auth.go:205-248), dan export membaca state lalu menghitung ulang di memori (umkm.go:388-433). Konsistensi snapshot belum dijamin.
10. **Kind alokasi tidak bisa dipaksa FK sederhana.** Tabel allocation hanya mereferensikan entry; constraint bahwa sale_entry_id harus sale dan receipt_entry_id harus receipt tetap di Go atau memerlukan trigger/composite FK.
11. **Baseline belum punya constraint domain.** pblg_users.role adalah text tanpa CHECK, pblg_workspaces.code unik global (bukan per app), dan tidak ada index untuk delete sesi per user. Index tambahan diusulkan, tetapi perubahan constraint pada data lama berisiko dan belum diputuskan.
12. **Kebijakan waktu belum seragam.** Prototipe memakai UTC untuk created_at import/allocation dan time.Now untuk expiry sesi; pedoman backend meminta zona Asia/Jakarta. Pilihan konversi/penyimpanan harus ditetapkan sebelum implementasi.
13. **Tidak ada refund, nilai negatif, multi-currency, atau rekonsiliasi sebagian lintas mata uang.** Semua di luar bukti source; jangan menambahkan query untuk fitur tersebut.

## 8. Rekomendasi urutan implementasi

1. Buat migration **000002_umkm_normalized** berisi tiga tabel dan index di atas, tanpa mengubah 000001.
2. Tambahkan query auth/shared lebih dulu, lalu query read UMKM, lalu query tulis UMKM.
3. Implementasikan service dengan transaksi **LockWorkspaceForUpdate** dan audit/version dalam unit of work yang sama.
4. Pindahkan handler tipis ke Echo; handler hanya bind, validasi bentuk, mengambil identitas, memanggil service, dan menulis respons.
5. Tambahkan backfill dari JSONB sebagai migration terpisah dan verifikasi jumlah entries/imports/allocations sebelum mematikan pembacaan state untuk UMKM.
