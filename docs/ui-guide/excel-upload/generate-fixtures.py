"""Build synthetic Excel/CSV cases for the EFA frontend handoff."""
from pathlib import Path
import csv
from datetime import date
from openpyxl import Workbook

root=Path(__file__).resolve().parent
out=root/'fixtures';out.mkdir(exist_ok=True)
sales=[['tanggal','referensi','nama','nominal','kanal']]
bank=[['Tanggal','Keterangan','Debit','Kredit','Saldo']]
for i in range(1,21):
    amount=200000 if i==20 else 100000+i*1000
    ref=f'INV-{900000+i}'
    sales.append([date(2026,11,5),ref,f'Pelanggan {i:02}',amount,'transfer'])
    bank.append([date(2026,11,5),f'{ref} Pelanggan {i:02}',0,190000 if i==20 else amount,5000000+i*amount])
bank.append([date(2026,11,5),'INV-999999 Penerimaan belum ada penjualan',0,75000,9000000])

def save(filename,sheets,metadata=False):
    wb=Workbook();wb.remove(wb.active)
    for name,rows in sheets.items():
        ws=wb.create_sheet(name)
        if metadata:
            for _ in range(7):ws.append(['Catatan simulasi EFA, bukan data usaha nyata'])
        for row in rows:ws.append(row)
        for row in ws:
            for cell in row:
                if isinstance(cell.value,date):cell.number_format='dd/mm/yyyy'
        ws.freeze_panes='A2'
    wb.save(out/filename)
save('penjualan-20.xlsx',{'Penjualan':sales},True)
save('mutasi-bank-21.xlsx',{'Mutasi Bank':bank})
save('dua-sheet.xlsx',{'Penjualan':sales,'Rekap':[['tanggal','referensi','nama','nominal','kanal'],[date(2026,11,5),'QA-REKAP','Penjualan harian simulasi',90000,'tunai']]})
save('lebih-20-sheet.xlsx',{f'Sheet {i}':sales[:2] for i in range(21)})
save('lebih-500-baris.xlsx',{'Penjualan':[sales[0]]+[[date(2026,11,5),f'T-{i}',f'Pelanggan {i}',1000,'transfer'] for i in range(501)]})
with (out/'csv-lama.csv').open('w',newline='',encoding='utf-8-sig') as f:
    w=csv.writer(f);w.writerow(['id','tanggal','referensi','nama','nominal','kanal','catatan']);w.writerow(['QA-CSV-1','2026-11-05','QA-CSV-1','Pelanggan CSV lama',42000,'tunai','Data sintetis'])
(out/'rusak.xlsx').write_bytes(b'PK\x03\x04bukan-workbook-valid')
(out/'format-lama.xls').write_bytes(bytes([0xD0,0xCF,0x11,0xE0])+b'data uji format xls')
(out/'dokumen.pdf').write_bytes(b'%PDF-1.4\n%%EOF\n')
# The oversize case is generated in ignored cache, never committed as an 11 MB fixture.
large=root.parents[2]/'.cache/excel-too-large.xlsx';large.parent.mkdir(exist_ok=True)
large.write_bytes(b'PK\x03\x04'+bytes(10*1024*1024-3))
try:
    from msoffcrypto.format.ooxml import OOXMLFile
    with (out/'penjualan-20.xlsx').open('rb') as src, (out/'terenkripsi.xlsx').open('wb') as dst:
        OOXMLFile(src).encrypt('EFA-test-only-2026',dst)
except ImportError:
    print('Install msoffcrypto-tool to regenerate the encrypted rejection fixture.')
print('Synthetic fixtures:',len(list(out.iterdir())))
