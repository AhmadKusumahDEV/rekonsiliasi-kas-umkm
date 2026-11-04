# Kontrak Frontend: Impor dan Verifikasi

Dokumen ini untuk frontend yang membangun layar impor dua slot dan halaman
Verifikasi. Backend sudah tersedia.

## Urutan panggilan

1. `GET /api/v1/config` untuk batas impor.
2. `POST /api/v1/umkm/imports/flexible` untuk setiap file.
3. `POST /api/v1/umkm/imports` dengan CSV kanonik dari respons fleksibel.
4. `GET /api/v1/umkm/state`, `/exceptions`, `/suggestions` setelah impor.

Commit penjualan lebih dulu, lalu penerimaan. Auto-match berjalan pada commit
kedua.

## Respons fleksibel

```json
{
  "csv": "id,tanggal,referensi,nama,nominal,kanal,catatan\n...",
  "count": 13,
  "total": 1661000,
  "checksum": "...",
  "preview": [ { "date": "2026-11-01", "reference": "INV-001", "amount": 100000 } ],
  "skipped": 0,
  "warnings": ["Baris 15 dilewati karena bukan uang masuk."]
}
```

Aturan:

- `preview` adalah array, bukan `{rows}`.
- `skipped` hanya duplikat workspace, bukan baris debit yang dilewati.
- `warnings` non-blocking dan tidak tersimpan.
- File yang semua barisnya keluar mengembalikan `200`, `count: 0`, dan warnings.

## Commit

```json
{"id":"...","imported":13,"skipped":0,"auto_matched":11,"ai_matched":0}
```

`auto_matched` menghitung pasangan, bukan baris. Tampilkan terpisah dari
`imported`.

`ai_matched` menghitung pasangan yang disarankan AI pada zona 0,60-0,95.
Nilainya 0 bila AI dimatikan, timeout, atau tidak ada kandidat ambigu.

## Data state

Entri sekarang memiliki:

```json
{"category":"cash","settled":true}
{"category":"non_operational"}
```

- `cash` dan `settled: true`: penjualan tunai sudah lunas; jangan dihitung
  sebagai sisa tagihan dan jangan tampilkan tombol alokasi.
- `non_operational`: bunga, biaya admin, transfer antar rekening; tampilkan di
  bagian terpisah, bukan antrean pencocokan.

## Exceptions

```json
{"entry":{"id":"...","reference":"INV-007","amount":90000,"category":""},
 "status":"unmatched","remaining":90000,
 "candidate_ids":["..."],"candidate_count":1,
 "suggestion_available":true,
 "reason":"Kandidat unik tersedia; belum dikonfirmasi."}
```

- Satu item per entri, bukan per pasangan. Gabungkan dua item yang saling
  menunjuk lewat `suggestions`.
- `remaining` adalah otoritas sisa saldo.
- `candidate_ids` maksimal 20; jika `candidate_ids_truncated`, sediakan
  pencarian manual.

## Setujui pasangan

Gunakan `POST /api/v1/umkm/allocations`:

```json
{"sale_id":"...","receipt_id":"...","amount":143500,"fee":6500,
 "reason":"Biaya QRIS Rp6.500","key":"32-hex"}
```

`amount` mengurangi sisa penjualan dan penerimaan; `fee` menambah pengurangan
di sisi penjualan. Untuk penjualan 150.000 dan penerimaan 143.500, gunakan
`amount=143500`, `fee=6500`.

Penjualan tunai dan penerimaan non-operasional akan ditolak `400`.

## Fee hint

Item exception sisi penjualan dapat membawa `fee_hint`:

```json
"fee_hint": {
  "amount": 6500,
  "source": "bank",
  "label": "Biaya bank atau penyedia pembayaran",
  "confidence": "tinggi",
  "reason": "Selisih Rp6500 dari Rp150000; kanal qris."
}
```

Aturan backend:

- Hanya muncul jika referensi sama dan hanya ada satu penerimaan yang lebih
  kecil.
- Selisih maksimal 5% dari nominal atau Rp25.000, mana yang lebih besar.
- `source` ditentukan dari kanal: `bank` untuk QRIS, transfer, e-wallet, kartu;
  `marketplace` untuk Shopee/Tokopedia/TikTok/Lazada; `unknown` bila tidak
  jelas.
- Ini saran, bukan potongan otomatis. Frontend cukup mengisi `amount` dan `fee`
  pada modal alokasi; pengguna tetap menyetujui.


## Layar

1. Dua slot: Penjualan dan Mutasi bank, masing-masing dengan pratinjau.
2. Ringkasan: baris siap, total, warnings, tombol simpan.
3. Setelah commit: “X baris tersimpan, Y pasangan otomatis, Z dilewati”.
4. Verifikasi: kartu pasangan siap setujui, kartu ambigu, kartu tanpa
   kandidat, dan bagian penerimaan non-operasional.
5. Setelah alokasi, muat ulang state dan exceptions.

## Jangan

- Commit CSV mentah hasil unggahan; selalu commit `response.csv`.
- Menampilkan “0 selisih” saat `/exceptions` gagal.
- Menghitung sisa saldo sendiri tanpa memperhatikan `settled`.
- Menganggap semua baris `preview` sudah tersimpan sebelum commit.

