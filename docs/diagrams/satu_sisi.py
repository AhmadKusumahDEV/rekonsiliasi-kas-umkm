import textwrap
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch

fig, ax = plt.subplots(figsize=(14, 7.4))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Kalau hanya ada satu sisi data", fontsize=18, fontweight="bold", color="#111827")
fig.text(0.045, 0.912, "Kekurangan data bukan selisih. Aplikasi menunggu sisi kedua, lalu mencocokkan otomatis.", fontsize=10.5, color="#6b7280")

panels = [
    ("Hanya penjualan", "#eef2ff", "#3730a3", [
        "Tunai langsung lunas",
        "QRIS/transfer menunggu bukti masuk",
        "Status: belum lengkap",
        "Saat mutasi diunggah, cocok otomatis",
    ]),
    ("Hanya mutasi bank", "#f5f3ff", "#6d28d9", [
        "Uang masuk tercatat",
        "Penjualan belum ada",
        "Status: belum lengkap",
        "Catat penjualan harian atau impor nanti",
    ]),
    ("Kedua sisi ada", "#ecfdf5", "#047857", [
        "Cocok per transaksi",
        "Sisa masuk temuan",
        "Bisa disetujui massal",
        "Laporan siap",
    ]),
]

w, h = 28, 36
gap = 4
left = (100 - (w * 3 + gap * 2)) / 2
top = 74
for i, (title, fill, edge, bullets) in enumerate(panels):
    x = left + i * (w + gap)
    ax.add_patch(FancyBboxPatch((x, top - h), w, h, boxstyle="round,pad=0.02,rounding_size=1.0",
                                linewidth=1.5, facecolor=fill, edgecolor=edge))
    ax.text(x + w / 2, top - 5, title, fontsize=11.4, fontweight="bold", color=edge, ha="center")
    y = top - 11.5
    for b in bullets:
        for part in textwrap.wrap(b, 30):
            ax.text(x + 1.4, y, part, fontsize=7.6, color="#374151", ha="left", va="top")
            y -= 3.1
        y -= 1.0

ax.text(50, 31.5, "Pencocokan dikerjakan pada tingkat yang tersedia", fontsize=10.5, fontweight="bold", color="#111827", ha="center")
rows = [
    ("Dua sisi lengkap", "cocok per transaksi, sisanya masuk temuan"),
    ("Satu sisi saja", "tampilkan sebagai menunggu, bukan selisih"),
    ("Hanya angka harian", "cocok pada total per hari dan per metode"),
]
for i, (k, v) in enumerate(rows):
    y = 25 - i * 5.0
    ax.text(24, y, k, fontsize=8.8, fontweight="bold", color="#111827", ha="right")
    ax.text(27, y, v, fontsize=8.8, color="#4b5563", ha="left")

ax.text(50, 5, "Pengguna tidak perlu menyiapkan dua file sebelum mulai; data yang ada disimpan dulu.", fontsize=8.8, color="#6b7280", ha="center")

fig.savefig("docs/diagrams/satu-sisi.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

