# GitHub Flow

`main` adalah branch yang selalu siap dipakai. Perubahan berikutnya dilakukan di
branch pendek, direview lewat pull request, lalu digabung setelah CI hijau.

## Alur kerja

1. Pastikan `main` terbaru:

   ```powershell
   git switch main
   git pull --ff-only
   ```

2. Buat branch dari `main`. Gunakan prefix `feat/`, `fix/`, `chore/`, atau
   `codex/` untuk perubahan dari agent:

   ```powershell
   git switch -c feat/nama-perubahan
   ```

3. Kerjakan dan commit perubahan. Commit mengikuti Conventional Commits,
   misalnya `feat(api): add health endpoint` atau
   `chore(ci): add backend workflow`.

4. Push branch lalu buka pull request ke `main`:

   ```powershell
   git push -u origin feat/nama-perubahan
   ```

5. Tunggu `Backend CI` lulus, minta review bila diperlukan, lalu merge. Branch
   pendek boleh dihapus setelah merge.

6. Setelah merge, kembali ke `main` dan tarik perubahan:

   ```powershell
   git switch main
   git pull --ff-only
   ```

## Apakah perlu git stash?

Tidak, untuk kasus biasa. Jika perubahan belum di-commit lalu kamu membuat branch
baru dengan `git switch -c`, perubahan itu ikut ke branch baru. Stash hanya
diperlukan saat berpindah ke branch yang punya perubahan konflik. Jika memang
perlu menyimpan perubahan termasuk file baru, gunakan `git stash -u`, lalu
`git stash pop` setelah berada di branch tujuan.

Scaffold backend dan workflow CI adalah commit baseline pertama di `main`.
Perubahan fitur setelah baseline tersebut mengikuti alur branch dan pull request
di atas.

## Pemeriksaan lokal

```powershell
cd backend
go mod download
go test ./...
go vet ./...
make lint
go build ./...
```

Project ini tidak memakai `vendor`. Jangan menambahkan `vendor/`, `go mod vendor`,
atau flag `-mod=vendor`.

## Database dan query

- Migration up/down berada di `backend/scripts/postgres/migrations`.
- Query sqlc berada di `backend/scripts/postgres/queries`.
- Generate type-safe query dengan `make gen.pg.models` dari folder `backend`.
- Hasil generate di `backend/internal/repository/rekonsiliasi_repository_sqlc`
  ikut di-commit dan tidak boleh diedit manual.
- Transaksi dibuka di `internal/service`, bukan di handler atau repository.

## Branch protection yang disarankan

Atur di GitHub Settings → Branches untuk branch `main`:

- Require a pull request before merging.
- Require status checks to pass, pilih `Backend CI / Test and build` dan
  `Backend CI / Lint` setelah workflow pernah berjalan.
- Require branches to be up to date before merging.
- Block force pushes dan deletion pada `main`.

Branch protection adalah pengaturan repository dan tidak bisa disimpan di file
source.
