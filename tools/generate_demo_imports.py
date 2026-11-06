#!/usr/bin/env python3
"""Generate demo import files for EFA. Run from repository root."""
import csv
import io
import os
import re
import zipfile
from datetime import datetime
from openpyxl import Workbook

OUT = os.environ.get("DEMO_OUT") or os.path.join("docs", "demo-files")
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

def amount_value(value):
    if isinstance(value, (int, float)):
        return int(value)
    digits = "".join(ch for ch in str(value) if ch.isdigit())
    return int(digits or 0)


def normalize_zip(path):
    # openpyxl menulis timestamp zip saat runtime; samakan agar byte stabil.
    with zipfile.ZipFile(path) as src:
        items = [(info, src.read(info.filename)) for info in src.infolist()]
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", zipfile.ZIP_DEFLATED) as dst:
        for info, data in items:
            if info.filename == "docProps/core.xml":
                data = re.sub(
                    rb"(<dcterms:modified[^>]*>)[^<]*(</dcterms:modified>)",
                    rb"\g<1>2026-01-01T00:00:00Z\g<2>",
                    data,
                )
            entry = zipfile.ZipInfo(info.filename, date_time=(2026, 1, 1, 0, 0, 0))
            entry.compress_type = zipfile.ZIP_DEFLATED
            entry.external_attr = info.external_attr
            dst.writestr(entry, data)
    with open(path, "wb") as handle:
        handle.write(buffer.getvalue())


def write_xlsx(path, header, rows, metadata_rows=0, extra_sheet=None):
    wb = Workbook()
    # Stempel waktu tetap supaya hasil generate tidak berubah tiap dijalankan.
    wb.properties.created = datetime(2026, 1, 1, 0, 0, 0)
    wb.properties.modified = datetime(2026, 1, 1, 0, 0, 0)
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
    normalize_zip(path)

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


# --- happy-33: 33 baris bersih, 29 pasangan aturan + 4 pasangan zona AI ---
clean_sales = []
clean_bank = []
clean_balance = 25_000_000


def clean_sale(ref, name, amount, channel, date):
    clean_sales.append({"tanggal": date, "referensi": ref, "nama": name, "nominal": amount, "kanal": channel})


def clean_receipt(date, desc, amount):
    global clean_balance
    clean_balance += amount
    clean_bank.append({"Tanggal": date, "Keterangan": desc, "Debit": 0, "Kredit": amount, "Saldo": clean_balance})


for i in range(1, 30):
    day = 1 + (i - 1) // 4
    date = f"2026-11-{day:02d}"
    ref = f"INV-2{i:02d}"
    name = names[(i - 1) % len(names)]
    amount = 60_000 + i * 7_000
    channel = "transfer" if i % 2 else "qris"
    clean_sale(ref, name, amount, channel, date)
    prefix = "TRF" if channel == "transfer" else "QRIS"
    clean_receipt(date, f"{prefix} {ref} {name.upper()}", amount)

# Zona AI: nominal, tanggal, dan referensi sama, tetapi nama pembayar berbeda.
ai_pairs_33 = [
    ("Budi", "SITI", 100_000),
    ("Citra", "DINA", 125_000),
    ("Sari", "RINA", 150_000),
    ("Dedi", "DEWI", 175_000),
]
for idx, (seller, payer, amount) in enumerate(ai_pairs_33, start=30):
    ref = f"INV-2{idx:02d}"
    clean_sale(ref, seller, amount, "transfer", "2026-11-08")
    clean_receipt("2026-11-08", f"TRF {ref} {payer}", amount)

excel_dir = os.path.join(OUT, "happy-excel")
os.makedirs(excel_dir, exist_ok=True)
write_xlsx(os.path.join(excel_dir, "penjualan-happy-33.xlsx"), sales_header, clean_sales)
write_xlsx(os.path.join(excel_dir, "mutasi-happy-33.xlsx"), bank_header, clean_bank)

hybrid_dir = os.path.join(OUT, "happy-hybrid")
os.makedirs(hybrid_dir, exist_ok=True)
write_csv(os.path.join(hybrid_dir, "penjualan-happy-33.csv"), sales_header, clean_sales)
write_xlsx(os.path.join(hybrid_dir, "mutasi-happy-33.xlsx"), bank_header, clean_bank)

# --- edge case: konten sulit, kolom tetap sesuai template resmi ---
edge_root = os.path.join(OUT, "edge-cases")


def edge_dir(name):
    path = os.path.join(edge_root, name)
    os.makedirs(path, exist_ok=True)
    return path


def write_pair(folder, name, sale_rows, bank_rows, sale_fmt="csv", bank_fmt="csv", metadata_rows=0, total_row=False):
    if sale_fmt == "csv":
        write_csv(os.path.join(folder, f"{name}-penjualan.csv"), sales_header, sale_rows)
    else:
        write_xlsx(os.path.join(folder, f"{name}-penjualan.xlsx"), sales_header, sale_rows, metadata_rows=metadata_rows)
    rows = list(bank_rows)
    if total_row:
        total_value = sum(amount_value(r["Kredit"]) for r in bank_rows)
        if any(isinstance(r["Kredit"], str) for r in bank_rows):
            total_value = "Rp " + f"{total_value:,}".replace(",", ".")
        rows = rows + [{"Tanggal": "2026-11-30", "Keterangan": "TOTAL", "Debit": 0,
                        "Kredit": total_value, "Saldo": 0}]
    if bank_fmt == "csv":
        write_csv(os.path.join(folder, f"{name}-mutasi.csv"), bank_header, rows)
    else:
        write_xlsx(os.path.join(folder, f"{name}-mutasi.xlsx"), bank_header, rows)


# 01: biaya layanan memotong nominal (QRIS, marketplace, admin bank)
f = edge_dir("01-selisih-fee")
s = [
    {"tanggal": "2026-11-02", "referensi": "INV-301", "nama": "Budi", "nominal": 150_000, "kanal": "qris"},
    {"tanggal": "2026-11-02", "referensi": "INV-302", "nama": "Rani", "nominal": 200_000, "kanal": "marketplace"},
    {"tanggal": "2026-11-03", "referensi": "INV-303", "nama": "Sari", "nominal": 90_000, "kanal": "qris"},
]
b = [
    {"Tanggal": "2026-11-02", "Keterangan": "QRIS INV-301 BUDI", "Debit": 0, "Kredit": 143_500, "Saldo": 10_143_500},
    {"Tanggal": "2026-11-03", "Keterangan": "PENCAIRAN SHOPEE INV-302 RANI", "Debit": 0, "Kredit": 190_000, "Saldo": 10_333_500},
    {"Tanggal": "2026-11-03", "Keterangan": "QRIS INV-303 SARI", "Debit": 0, "Kredit": 83_500, "Saldo": 10_417_000},
    {"Tanggal": "2026-11-03", "Keterangan": "BIAYA ADMIN BULANAN", "Debit": 2_500, "Kredit": 0, "Saldo": 10_414_500},
]
write_pair(f, "fee", s, b)

# 02: satu penjualan dibayar dua kali (pembayaran sebagian)
f = edge_dir("02-bayar-sebagian")
s = [
    {"tanggal": "2026-11-04", "referensi": "INV-311", "nama": "Dedi", "nominal": 500_000, "kanal": "transfer"},
    {"tanggal": "2026-11-04", "referensi": "INV-312", "nama": "Ayu", "nominal": 250_000, "kanal": "transfer"},
]
b = [
    {"Tanggal": "2026-11-04", "Keterangan": "TRF INV-311 DEDI DP 1", "Debit": 0, "Kredit": 300_000, "Saldo": 11_300_000},
    {"Tanggal": "2026-11-05", "Keterangan": "TRF INV-311 DEDI PELUNASAN", "Debit": 0, "Kredit": 200_000, "Saldo": 11_500_000},
    {"Tanggal": "2026-11-04", "Keterangan": "TRF AYU", "Debit": 0, "Kredit": 250_000, "Saldo": 11_750_000},
]
write_pair(f, "sebagian", s, b)

# 03: penjualan tunai dan penjualan belum dibayar
f = edge_dir("03-tunai-dan-belum-bayar")
s = [
    {"tanggal": "2026-11-05", "referensi": "INV-321", "nama": "Eka", "nominal": 60_000, "kanal": "tunai"},
    {"tanggal": "2026-11-05", "referensi": "INV-322", "nama": "Fajar", "nominal": 75_000, "kanal": "tunai"},
    {"tanggal": "2026-11-06", "referensi": "INV-323", "nama": "Indra", "nominal": 400_000, "kanal": "transfer"},
    {"tanggal": "2026-11-06", "referensi": "INV-324", "nama": "Gita", "nominal": 320_000, "kanal": "qris"},
]
b = []
write_pair(f, "tunai", s, b)

# 04: nominal kembar tanpa referensi -> harus masuk Verifikasi
f = edge_dir("04-nominal-kembar")
s = [
    {"tanggal": "2026-11-07", "referensi": "", "nama": "Hana", "nominal": 95_000, "kanal": "transfer"},
    {"tanggal": "2026-11-07", "referensi": "", "nama": "Hana", "nominal": 95_000, "kanal": "transfer"},
]
b = [
    {"Tanggal": "2026-11-07", "Keterangan": "TRF HANA", "Debit": 0, "Kredit": 95_000, "Saldo": 12_045_000},
    {"Tanggal": "2026-11-07", "Keterangan": "TRF HANA", "Debit": 0, "Kredit": 95_000, "Saldo": 12_140_000},
]
write_pair(f, "kembar", s, b)

# 05: baris non-operasional di mutasi bank
f = edge_dir("05-non-operasional")
s = [
    {"tanggal": "2026-11-09", "referensi": "INV-331", "nama": "Joko", "nominal": 180_000, "kanal": "transfer"},
]
b = [
    {"Tanggal": "2026-11-09", "Keterangan": "TRF INV-331 JOKO", "Debit": 0, "Kredit": 180_000, "Saldo": 12_320_000},
    {"Tanggal": "2026-11-09", "Keterangan": "BUNGA TABUNGAN", "Debit": 0, "Kredit": 2_500, "Saldo": 12_322_500},
    {"Tanggal": "2026-11-09", "Keterangan": "TRANSFER ANTAR REKENING SENDIRI", "Debit": 0, "Kredit": 500_000, "Saldo": 12_822_500},
    {"Tanggal": "2026-11-09", "Keterangan": "BIAYA ADMIN", "Debit": 6_500, "Kredit": 0, "Saldo": 12_816_000},
]
write_pair(f, "non-operasional", s, b)

# 06: format berantakan tetapi kolom tetap resmi
f = edge_dir("06-format-berantakan")
s = [
    {"tanggal": "01/11/2026", "referensi": "INV-341", "nama": "Kirana", "nominal": "Rp 1.250.000", "kanal": "transfer"},
    {"tanggal": "02/11/2026", "referensi": "INV-342", "nama": "Lina", "nominal": "Rp 875.000", "kanal": "qris"},
    {"tanggal": "03/11/2026", "referensi": "INV-343", "nama": "Miko", "nominal": "Rp 1.100.000", "kanal": "transfer"},
]
b = [
    {"Tanggal": "01/11/2026", "Keterangan": "TRF INV-341 KIRANA", "Debit": 0, "Kredit": "Rp 1.250.000", "Saldo": "Rp 13.250.000"},
    {"Tanggal": "02/11/2026", "Keterangan": "QRIS INV-342 LINA", "Debit": 0, "Kredit": "Rp 868.500", "Saldo": "Rp 14.118.500"},
    {"Tanggal": "03/11/2026", "Keterangan": "TRF INV-343 MIKO", "Debit": 0, "Kredit": "Rp 1.100.000", "Saldo": "Rp 15.218.500"},
]
write_pair(f, "berantakan", s, b, sale_fmt="xlsx", bank_fmt="xlsx", metadata_rows=7, total_row=True)

print("happy-33 sales:", len(clean_sales), "receipts:", len(clean_bank))



# --- minimal kolom: hanya tanggal, nama/keterangan, nominal ---
minim_dir = os.path.join(OUT, "minim-column")
os.makedirs(minim_dir, exist_ok=True)

minim_sales = []
for i in range(1, 34):
    day = 12 + (i - 1) // 12
    minim_sales.append((
        f"2026-11-{day:02d}",
        names[(i - 1) % len(names)],
        55_000 + i * 7_500,
    ))

write_csv(os.path.join(minim_dir, "penjualan-minim.csv"), ["tanggal", "nama", "nominal"], [
    {"tanggal": date, "nama": name, "nominal": amount} for date, name, amount in minim_sales
])

write_csv(os.path.join(minim_dir, "mutasi-minim.csv"), ["tanggal", "keterangan", "nominal"], [
    {"tanggal": date, "keterangan": name.upper(), "nominal": amount} for date, name, amount in minim_sales
])

print("minim-column: 33 penjualan, 33 penerimaan")
