#!/usr/bin/env python3
"""Generate demo import files for EFA. Run from repository root."""
import csv
import os
from openpyxl import Workbook

OUT = os.path.join("docs", "demo-files")
os.makedirs(OUT, exist_ok=True)

names = ["Budi","Rani","Sari","Dedi","Ayu","Toko A","Citra","Eka","Fajar","Gita",
         "Hana","Indra","Joko","Kirana","Lina","Miko","Nia","Oki","Putri","Rizky"]

sales = []
bank = []
balance = 10_000_000

def sale(ref, name, amount, channel, date):
    sales.append({"tanggal": date, "referensi": ref, "nama": name, "nominal": amount, "kanal": channel})

def receipt(date, desc, amount, ref=""):
    global balance
    balance += amount
    bank.append({"Tanggal": date, "Keterangan": desc, "Debit": 0, "Kredit": amount, "Saldo": balance})

for i in range(1, 21):
    day = 1 + (i - 1) // 4
    date = f"2026-11-{day:02d}"
    ref = f"INV-{100+i}"
    name = names[i-1]
    amount = 50_000 + i * 7_500
    channel = "transfer" if i % 2 else "qris"
    sale(ref, name, amount, channel, date)
    prefix = "TRF" if channel == "transfer" else "QRIS"
    receipt(date, f"{prefix} {ref} {name.upper()}", amount, ref)

ai_names = [("Budi","SITI"),("Citra","DINA"),("Sari","RINA"),("Dedi","DEWI")]
for idx, (seller, payer) in enumerate(ai_names, start=21):
    ref = f"INV-{100+idx}"
    amount = 100_000 + (idx - 21) * 25_000
    sale(ref, seller, amount, "transfer", "2026-11-04")
    receipt("2026-11-04", f"TRF {ref} {payer}", amount, ref)

for idx, amount in zip(range(25,28), [150_000, 200_000, 250_000]):
    ref = f"INV-{100+idx}"
    name = names[(idx - 1) % len(names)]
    sale(ref, name, amount, "qris", "2026-11-05")
    receipt("2026-11-05", f"QRIS {ref} {name.upper()}", amount - 6_500, ref)

for _ in (28,29):
    sale("", "Hana", 95_000, "transfer", "2026-11-05")
    receipt("2026-11-05", "TRF HANA", 95_000, "")

sale("INV-130", "Eka", 60_000, "tunai", "2026-11-06")
sale("INV-131", "Fajar", 70_000, "tunai", "2026-11-06")
sale("INV-132", "Indra", 400_000, "transfer", "2026-11-06")
sale("INV-133", "Putri", 300_000, "marketplace", "2026-11-06")
receipt("2026-11-06", "PENCAIRAN SHOPEE INV-133 PUTRI", 290_000, "INV-133")
receipt("2026-11-06", "BUNGA TABUNGAN", 2_500, "")
receipt("2026-11-06", "TRANSFER ANTAR REKENING SENDIRI", 500_000, "")

sales_header = ["tanggal", "referensi", "nama", "nominal", "kanal"]
bank_header = ["Tanggal", "Keterangan", "Debit", "Kredit", "Saldo"]

def write_csv(path, header, rows, delimiter=","):
    with open(path, "w", newline="", encoding="utf-8") as f:
        w = csv.writer(f, delimiter=delimiter)
        w.writerow(header)
        for row in rows:
            w.writerow([row[h] for h in header])

def write_xlsx(path, header, rows, metadata_rows=0, extra_sheet=None):
    wb = Workbook()
    ws = wb.active
    ws.title = "Penjualan" if header == sales_header else "Mutasi"
    for _ in range(metadata_rows):
        ws.append(["Rekap data usaha"])
    ws.append(header)
    for row in rows:
        ws.append([row[h] for h in header])
    if extra_sheet:
        sh = wb.create_sheet("Rekap")
        sh.append(header)
        sh.append([extra_sheet[h] for h in header])
    wb.save(path)

write_csv(os.path.join(OUT, "penjualan-33.csv"), sales_header, sales)
write_csv(os.path.join(OUT, "penjualan-33-semicolon.csv"), sales_header, sales, ";")
write_csv(os.path.join(OUT, "mutasi-32.csv"), bank_header, bank)
write_csv(os.path.join(OUT, "mutasi-32-semicolon.csv"), bank_header, bank, ";")
write_xlsx(os.path.join(OUT, "penjualan-33.xlsx"), sales_header, sales)
write_xlsx(os.path.join(OUT, "penjualan-33-header8.xlsx"), sales_header, sales, metadata_rows=7)
write_xlsx(os.path.join(OUT, "mutasi-32.xlsx"), bank_header, bank)
write_xlsx(os.path.join(OUT, "mutasi-32-multisheet.xlsx"), bank_header, bank, extra_sheet=bank[0])

with open(os.path.join(OUT, "penjualan-header-duplikat.csv"), "w", newline="", encoding="utf-8") as f:
    w = csv.writer(f)
    w.writerow(["tanggal", "nama", "nominal", "nominal", "kanal"])
    w.writerow(["2026-11-03", "Budi", "100000", "999999", "transfer"])

total_rows = list(bank)
total_rows.append({"Tanggal": "2026-11-06", "Keterangan": "TOTAL", "Debit": 0, "Kredit": sum(r["Kredit"] for r in bank), "Saldo": 0})
write_csv(os.path.join(OUT, "mutasi-total-row.csv"), bank_header, total_rows)

happy_dir = os.path.join(OUT, "happy")
os.makedirs(happy_dir, exist_ok=True)
for src_name, dst_name in [
    ("penjualan-33.csv", "penjualan-happy-24.csv"),
    ("mutasi-32.csv", "mutasi-happy-24.csv"),
]:
    with open(os.path.join(OUT, src_name), newline="", encoding="utf-8") as src:
        rows = list(csv.reader(src))
    with open(os.path.join(happy_dir, dst_name), "w", newline="", encoding="utf-8") as dst:
        writer = csv.writer(dst)
        writer.writerow(rows[0])
        writer.writerows(rows[1:25])

print("files:", len(os.listdir(OUT)))
print("sales:", len(sales), "receipts:", len(bank))

