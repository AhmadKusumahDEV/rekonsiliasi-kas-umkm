#!/usr/bin/env python3
"""Membuat file impor yang sengaja salah untuk menguji pesan error EFA."""
import os
import zipfile

OUT = os.environ.get("ERROR_OUT") or os.path.join("docs", "demo-files", "error-cases")
os.makedirs(OUT, exist_ok=True)

SALE_HEADER = "tanggal,referensi,nama,nominal,kanal"
BANK_HEADER = "Tanggal,Keterangan,Debit,Kredit,Saldo"


def write(name, data, mode="w"):
    path = os.path.join(OUT, name)
    if "b" in mode:
        with open(path, mode) as handle:
            handle.write(data)
        return
    with open(path, mode, encoding="utf-8", newline="") as handle:
        handle.write(data)


# 1. Bukan CSV sama sekali, tetapi diberi nama .txt
write("01-teks-biasa.txt", "Ini catatan belanja, bukan data transaksi.\n")

# 2. Isi PDF tetapi berekstensi .csv
write("02-pdf-palsu.csv", "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n%%EOF\n")

# 3. File .xls lama (OLE2)
write("03-xls-lama.xls", b"\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1" + b"\x00" * 512, "wb")

# 4. Gambar PNG tetapi berekstensi .xlsx
write("04-png-palsu.xlsx", b"\x89PNG\r\n\x1a\n" + b"\x00" * 256, "wb")

# 5. File kosong
write("05-kosong.csv", "")

# 6. Hanya header, tanpa baris data
write("06-hanya-header.csv", SALE_HEADER + "\n")

# 7. Header duplikat
write("07-header-duplikat.csv",
      "tanggal,referensi,nama,nominal,nominal,kanal\n2026-11-03,INV-900,Budi,100000,999999,transfer\n")

# 8. Tidak ada kolom wajib
write("08-tanpa-kolom.csv", "foo,bar,baz\nsatu,dua,tiga\n")

# 9. Nominal tidak terbaca
write("09-nominal-rusak.csv", SALE_HEADER + "\n2026-11-03,INV-901,Budi,seratus ribu,transfer\n")

# 10. Tanggal tidak terbaca
write("10-tanggal-rusak.csv", SALE_HEADER + "\nkemarin,INV-902,Budi,100000,transfer\n")

# 11. Melebihi batas 500 baris
rows = [SALE_HEADER]
for i in range(1, 502):
    rows.append(f"2026-11-03,INV-{900 + i:04d},Pelanggan {i},{50000 + i},transfer")
write("11-baris-501.csv", "\n".join(rows) + "\n")

# 12. Melebihi batas 1 MB CSV
pad = "x" * 900
bulk = [SALE_HEADER]
for i in range(1, 1300):
    bulk.append(f"2026-11-03,INV-{i:05d},Pelanggan {i},{50000 + i},transfer,{pad}")
write("12-besar-1mb.csv", "\n".join(bulk) + "\n")

# 13. Zip biasa, bukan workbook
zip_path = os.path.join(OUT, "13-zip-palsu.xlsx")
with zipfile.ZipFile(zip_path, "w") as archive:
    archive.writestr("readme.txt", "bukan workbook")

# 14. Kutip tidak seimbang
write("14-kutip-rusak.csv", SALE_HEADER + "\n2026-11-03,INV-903,\"Budi,100000,transfer\n")

# 15. Nominal negatif
write("15-nominal-negatif.csv", SALE_HEADER + "\n2026-11-03,INV-904,Budi,-150000,transfer\n")

# 16. Nominal nol
write("16-nominal-nol.csv", SALE_HEADER + "\n2026-11-03,INV-905,Budi,0,transfer\n")

# 17. Mutasi tanpa kolom kredit
write("17-mutasi-tanpa-kredit.csv", "Tanggal,Keterangan,Debit,Saldo\n2026-11-03,TRF INV-906 BUDI,0,1000000\n")

print("file dibuat:", len([n for n in os.listdir(OUT) if not n.endswith(".md")]))
