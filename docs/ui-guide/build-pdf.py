"""Build a categorized, source-labelled guide from real browser screenshots."""
from pathlib import Path
from io import BytesIO
from collections import Counter
from datetime import datetime, timezone, timedelta
from html import escape
import hashlib
import json
import math
import os

from PIL import Image
from reportlab.pdfgen import canvas
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.lib.colors import HexColor, Color
from reportlab.lib.styles import ParagraphStyle
from reportlab.platypus import Paragraph
from reportlab.lib.utils import ImageReader
from pypdf import PdfReader

ROOT = Path(__file__).resolve().parent
PROJECT = ROOT.parent.parent
OUT = ROOT.parent / "EFA-UI-Guide.pdf"
TMP = PROJECT / ".cache" / "pdf-ui-guide"
TMP.mkdir(parents=True, exist_ok=True)
manifest = json.loads((ROOT / "manifest.json").read_text(encoding="utf-8"))
records = sorted(manifest["screenshots"], key=lambda r: (r["category"], r["slug"], r["device"]))
def font_path(env, choices):
    value = os.environ.get(env)
    if value:
        return value
    for choice in choices:
        if Path(choice).is_file():
            return str(choice)
    raise FileNotFoundError("Set " + env + " to an available TTF font")

pdfmetrics.registerFont(TTFont("Body", font_path("FONT_BODY", [
    "C:/Windows/Fonts/segoeui.ttf", "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
])))
pdfmetrics.registerFont(TTFont("BodyBold", font_path("FONT_BOLD", [
    "C:/Windows/Fonts/seguisb.ttf", "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
])))
pdfmetrics.registerFont(TTFont("Heading", str(PROJECT / "frontend/assets/poppins-semibold.ttf")))

W, H = 1120, 900
INK, MUTED = HexColor("#3D414A"), HexColor("#5C6570")
BLUE, CYAN, AMBER, CORAL = [HexColor(x) for x in ("#5AA2E0", "#54B0D9", "#F59E0B", "#EB7A7A")]
LINK, LINE, PALE = HexColor("#285F8C"), HexColor("#DCE5EC"), HexColor("#EDF5FC")
WHITE = HexColor("#FFFFFF")
captured = datetime.fromisoformat(manifest["capturedAt"].replace("Z", "+00:00")).astimezone(timezone(timedelta(hours=7)))
months = ["Januari","Februari","Maret","April","Mei","Juni","Juli","Agustus","September","Oktober","November","Desember"]
date_text = str(captured.day) + " " + months[captured.month - 1] + " " + str(captured.year)
doc = canvas.Canvas(str(OUT), pagesize=(W, H), pageCompression=1)
doc.setTitle("Easy Financial Audit - Panduan Antarmuka")
doc.setAuthor("Evan William")
doc.setSubject("Dokumentasi UI desktop dan HP, alur pengguna, dan status tampilan")
doc.setKeywords("EFA, Easy Financial Audit, UMKM, UI, desktop, mobile")

def paragraph(text, x, top, width, size=12, color=INK, font="Body", leading=None):
    style = ParagraphStyle("p", fontName=font, fontSize=size, leading=leading or size * 1.45, textColor=color)
    p = Paragraph(escape(text), style)
    _, height = p.wrap(width, H)
    if top - height < 46:
        raise ValueError("Text exceeds page: " + text[:100])
    p.drawOn(doc, x, top - height)
    return top - height

def footer(number, detail=""):
    doc.setStrokeColor(LINE)
    doc.setLineWidth(.7)
    doc.line(48, 47, W - 48, 47)
    doc.setFillColor(MUTED)
    doc.setFont("Body", 9)
    doc.drawString(48, 28, "EASY FINANCIAL AUDIT  /  PANDUAN UI")
    if detail:
        doc.drawCentredString(W / 2, 28, detail)
    doc.drawRightString(W - 48, 28, str(number).zfill(2))

def image_jpeg(im):
    data = BytesIO()
    if im.mode != "RGB":
        base = Image.new("RGB", im.size, "white")
        if im.mode == "RGBA":
            base.paste(im, mask=im.getchannel("A"))
        else:
            base.paste(im.convert("RGB"))
        im = base
    im.save(data, format="JPEG", quality=92, subsampling=0, optimize=True)
    data.seek(0)
    return ImageReader(data)

def fit_image(im, x, y, width, height, border=True):
    scale = min(width / im.width, height / im.height)
    iw, ih = im.width * scale, im.height * scale
    ix, iy = x + (width - iw) / 2, y + (height - ih) / 2
    doc.drawImage(image_jpeg(im), ix, iy, width=iw, height=ih)
    if border:
        doc.setStrokeColor(LINE)
        doc.setLineWidth(.8)
        doc.rect(ix, iy, iw, ih, fill=0, stroke=1)
    return ix, iy, iw, ih

def brand_header():
    doc.drawImage(str(PROJECT / "frontend/assets/icon-192.png"), 48, H - 69, 31, 31, mask="auto")
    doc.setFillColor(INK)
    doc.setFont("Heading", 12)
    doc.drawString(88, H - 48, "Easy Financial Audit")
    doc.setFillColor(MUTED)
    doc.setFont("Body", 9)
    doc.drawString(88, H - 63, "Dokumentasi tampilan dan alur pengguna")

plan = []
category_pages = {}
page_no = 3
for record in records:
    image = Image.open(ROOT / record["file"])
    cap = image.width * (3.2 if record["device"] == "mobile" else .78)
    count = max(1, math.ceil(image.height / cap))
    chunk_h = math.ceil(image.height / count)
    record["pdfPage"] = page_no
    record["pdfParts"] = count
    category_pages.setdefault(record["category"], page_no)
    for part in range(count):
        top = max(0, part * chunk_h - (24 if part else 0))
        bottom = min(image.height, (part + 1) * chunk_h + (24 if part + 1 < count else 0))
        plan.append((record, part + 1, count, (0, top, image.width, bottom)))
        page_no += 1
total_pages = page_no

# Cover.
doc.bookmarkPage("cover")
doc.addOutlineEntry("Panduan UI", "cover", level=0)
brand_header()
paragraph("PANDUAN ANTARMUKA", 48, 758, 550, 12, LINK, "BodyBold")
y = paragraph("Lihat setiap halaman.\nPahami setiap alur.", 48, 713, 620, 42, INK, "Heading", 57)
y = paragraph("Easy Financial Audit membantu pemilik usaha menelusuri penjualan, penerimaan, selisih dan keputusan pencocokan. Dokumen ini memperlihatkan UI yang benar-benar dirender oleh aplikasi, bukan mockup.", 48, y - 28, 560, 15)
paragraph(str(len(records)) + " screenshot  |  Desktop 1440 px dan HP 390 px", 48, y - 30, 600, 13, LINK, "BodyBold")
paragraph("Tampilan live dan skenario lokal dipisahkan. Data transaksi pada panduan adalah contoh sintetis.", 48, y - 65, 560, 12, MUTED)
logo = Image.open(PROJECT / "frontend/assets/efa-logo-hd.png")
fit_image(logo, 705, 420, 330, 300, border=False)
doc.setFillColor(PALE)
doc.rect(48, 140, W - 96, 118, stroke=0, fill=1)
paragraph("DEMO WEB", 70, 236, 400, 10, LINK, "BodyBold")
paragraph("https://efa.madgo.my.id/", 70, 212, 600, 21, INK, "Heading")
doc.linkURL("https://efa.madgo.my.id/", (68, 175, 550, 220), relative=0)
paragraph(date_text + "  |  Disusun oleh Evan William", 70, 174, 600, 11, MUTED)
for i, color in enumerate([BLUE, CYAN, AMBER, CORAL, INK, WHITE]):
    doc.setFillColor(color)
    doc.rect(48 + i * (W - 96) / 6, 95, (W - 96) / 6, 9, fill=1, stroke=0)
footer(1, "Data contoh sintetis")
doc.showPage()

# Contents and provenance.
brand_header()
doc.bookmarkPage("contents")
doc.addOutlineEntry("Isi dan cara membaca", "contents", level=0)
paragraph("Isi dan cara membaca", 48, 775, 900, 31, INK, "Heading")
counts = Counter(r["category"] for r in records)
labels = {r["category"]: r["categoryLabel"] for r in records}
y = 706
for i, category in enumerate(sorted(counts), 1):
    doc.setFillColor(LINK)
    doc.setFont("BodyBold", 12)
    doc.drawString(48, y, str(i).zfill(2))
    doc.setFillColor(INK)
    doc.setFont("Heading", 17)
    doc.drawString(95, y, labels[category])
    doc.setFillColor(MUTED)
    doc.setFont("Body", 11)
    doc.drawRightString(W - 48, y, str(counts[category]) + " screenshot  /  hlm. " + str(category_pages[category]))
    doc.linkRect("", "section-" + category, (45, y - 13, W - 45, y + 20), relative=0, thickness=0)
    doc.setStrokeColor(LINE)
    doc.line(48, y - 21, W - 48, y - 21)
    y -= 50
paragraph("LIVE: screenshot dari efa.madgo.my.id. Pengambilan hanya membuka halaman dan login normal; tidak mengubah catatan keuangan akun demo utama.", 48, 270, 485, 11)
paragraph("LOCAL: skenario berisi pembayaran sebagian, biaya, pembatalan, hasil kosong dan error dijalankan di database rku_ui_guide pada localhost. Ini bukan bukti skenario tersebut ditulis ke database live.", 585, 270, 485, 11)
paragraph("Halaman panjang dibagi menjadi beberapa cuplikan dengan sedikit area berulang agar teks tetap terbaca. File PNG asli tetap utuh di folder screenshot. Konfirmasi native browser dijelaskan melalui teks, bukan dibuat sebagai mockup.", 48, 159, 1020, 11, MUTED)
footer(2, str(total_pages) + " halaman termasuk penutup")
doc.showPage()

current_category = None
current_page = 3
for record, part, count, box in plan:
    if record["category"] != current_category:
        current_category = record["category"]
        doc.bookmarkPage("section-" + current_category)
        doc.addOutlineEntry(record["categoryLabel"], "section-" + current_category, level=0)
    if part == 1:
        key = "screen-" + str(current_page)
        doc.bookmarkPage(key)
        doc.addOutlineEntry(record["title"] + " - " + record["device"], key, level=1)
    doc.setFillColor(LINK)
    doc.setFont("BodyBold", 11)
    doc.drawString(48, H - 44, record["categoryLabel"].upper())
    suffix = "" if count == 1 else " / " + str(part) + " dari " + str(count)
    paragraph(record["title"] + suffix, 48, H - 67, 1015, 27, INK, "Heading", 35)
    device = "HP 390 px" if record["device"] == "mobile" else "Desktop 1440 px"
    source_label = "LIVE - efa.madgo.my.id" if record["source"] == "LIVE" else "LOCAL - database dokumentasi terpisah"
    doc.setFont("Body", 10)
    doc.setFillColor(MUTED)
    doc.drawString(48, H - 116, source_label + "  |  " + device)
    im = Image.open(ROOT / record["file"]).crop(box)
    if record["device"] == "mobile":
        paragraph("Fungsi tampilan", 48, 708, 475, 12, LINK, "BodyBold")
        bottom = paragraph(record["text"], 48, 679, 480, 15)
        bottom = paragraph("Alur penggunaan", 48, bottom - 34, 480, 12, LINK, "BodyBold")
        for index, step in enumerate(record["steps"], 1):
            bottom = paragraph(str(index) + ". " + step, 48, bottom - 15, 480, 13)
        paragraph("Alur teknis", 48, bottom - 40, 480, 12, LINK, "BodyBold")
        paragraph(record["endpoint"], 48, bottom - 68, 480, 11, MUTED)
        fit_image(im, 610, 95, 440, 650)
    else:
        fit_image(im, 48, 177, W - 96, 593)
        paragraph(record["text"], 48, 150, 690, 11.5)
        paragraph(" / ".join(record["steps"]), 786, 150, 285, 10.5, MUTED)
        paragraph("Alur: " + record["endpoint"], 48, 77, W - 96, 9.5, LINK)
    footer(current_page, record["source"] + "  /  " + record["device"].upper() + suffix)
    doc.showPage()
    current_page += 1

# Closing reference.
brand_header()
doc.bookmarkPage("coverage")
doc.addOutlineEntry("Cakupan dan sumber", "coverage", level=0)
paragraph("Cakupan dan sumber", 48, 775, 1024, 32, INK, "Heading")
y = paragraph("Semua halaman utama serta dialog penting tercakup: masuk, daftar, ringkasan, transaksi, pencocokan, impor, riwayat, laporan, pengaturan, menu HP, ganti kata sandi dan fallback navigasi.", 48, 710, 1015, 15)
y = paragraph("Keadaan tambahan: login ditolak, usaha kosong, pencarian tanpa hasil, filter penerimaan, pagination, preview CSV valid, CSV tidak valid, pembayaran sebagian, biaya layanan, penerimaan gabungan, dan riwayat koreksi.", 48, y - 28, 1015, 15)
y = paragraph("Sumber screenshot", 48, y - 40, 1015, 18, LINK, "Heading")
y = paragraph("Halaman LIVE berasal dari web demo yang memakai backend prototipe Go dan endpoint /api. Source integrasi tim memakai /api/v1 dan didokumentasikan pada architecture.md. Logo dan paletnya sama, tetapi endpoint kedua versi tidak boleh tertukar.", 48, y - 20, 1015, 13)
y = paragraph("Skenario LOCAL memakai source UI deployment yang sama dan database khusus dokumentasi. Screenshot menampilkan kondisi aplikasi nyata; label LOCAL selalu dipertahankan agar tidak dianggap bukti perubahan database produksi.", 48, y - 20, 1015, 13)
y = paragraph("File pendamping", 48, y - 36, 1015, 18, LINK, "Heading")
for text in [
    "docs/ui-guide/screenshots: PNG utuh, dibagi kategori dan perangkat.",
    "docs/ui-guide/manifest.json: sumber, ukuran layar, waktu dan deskripsi setiap gambar.",
    "docs/ui-guide/README.md: indeks gambar dan cara memperbarui panduan.",
    "architecture.md, docs/BRAND.md: alur program, kontrak endpoint dan identitas visual.",
]:
    y = paragraph(text, 48, y - 15, 1015, 12)
footer(current_page, "Tidak ada klaim penggunaan UMKM nyata atau produk bebas bug")
doc.save()

reader = PdfReader(OUT)
if len(reader.pages) != total_pages:
    raise AssertionError((len(reader.pages), total_pages))
page_text = [page.extract_text() or "" for page in reader.pages]
if any(not text.strip() for text in page_text):
    raise AssertionError("Empty PDF page")
index = []
for record in records:
    digest = hashlib.sha256((ROOT / record["file"]).read_bytes()).hexdigest()
    index.append({**record, "sha256": digest})
report = {
    "pdf": OUT.name, "pages": len(reader.pages), "screenshots": len(records),
    "live": sum(r["source"] == "LIVE" for r in records),
    "local": sum(r["source"] == "LOCAL" for r in records),
    "bytes": OUT.stat().st_size, "runtimeErrors": manifest["runtimeErrors"],
    "categories": dict(counts), "index": index,
}
(ROOT / "pdf-index.json").write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
lines = [
    "# Indeks screenshot EFA",
    "",
    "[Buka PDF lengkap](../EFA-UI-Guide.pdf). Nomor halaman berikut mengikuti PDF yang dibangun bersama indeks ini.",
    "",
    "LIVE = demo online tanpa perubahan transaksi. LOCAL = database dokumentasi lokal terpisah.",
    "",
]
for category in sorted(counts):
    lines.extend(["## " + labels[category], "", "| Tampilan | Perangkat | Sumber | PDF | PNG utuh |", "| --- | --- | --- | --- | --- |"])
    for record in records:
        if record["category"] == category:
            lines.append("| " + record["title"] + " | " + record["device"] + " | " + record["source"] +
                         " | " + str(record["pdfPage"]) + " | [Buka](" + record["file"] + ") |")
    lines.append("")
(ROOT / "SCREENSHOT-INDEX.md").write_text("\n".join(lines), encoding="utf-8")
print(json.dumps({k:v for k,v in report.items() if k != "index"}, indent=2))
