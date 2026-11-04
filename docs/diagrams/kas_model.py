import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

fig, ax = plt.subplots(figsize=(14, 7.6))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Model sederhana: penjualan, uang masuk, lalu temuan", fontsize=18, fontweight="bold", color="#111827")
fig.text(0.045, 0.912, "Pemilik usaha hanya menjawab dua hal: apa yang terjual, dan uang apa yang benar-benar diterima", fontsize=10.5, color="#6b7280")

def box(x, y, w, h, fill, edge, dashed=False):
    ax.add_patch(FancyBboxPatch((x, y), w, h, boxstyle="round,pad=0.02,rounding_size=1.0",
                                linewidth=1.5, facecolor=fill, edgecolor=edge,
                                linestyle="--" if dashed else "-"))

# three columns
box(4, 46, 28, 34, "#eef2ff", "#3730a3")
box(36, 46, 28, 34, "#f5f3ff", "#6d28d9")
box(68, 46, 28, 34, "#ecfdf5", "#047857")

ax.text(18, 74, "1. Penjualan", fontsize=12.5, fontweight="bold", color="#3730a3", ha="center")
ax.text(18, 70, "apa yang terjual hari ini", fontsize=8.4, color="#4b5563", ha="center")
for i, (label, sub) in enumerate([("Tunai", "uang langsung di tangan"), ("QRIS / Transfer", "masuk H+1, potong MDR"), ("Marketplace", "dicairkan H+7, gabungan"), ("COD / kurir", "diterima lewat kurir")]):
    yy = 65.5 - i * 5.4
    ax.text(6, yy, label, fontsize=9.2, fontweight="bold", color="#111827")
    ax.text(6, yy - 2.3, sub, fontsize=7.2, color="#6b7280")

ax.text(50, 74, "2. Uang masuk", fontsize=12.5, fontweight="bold", color="#6d28d9", ha="center")
ax.text(50, 70, "apa yang benar-benar diterima", fontsize=8.4, color="#4b5563", ha="center")
ax.text(38, 63, "Tunai: sudah diterima", fontsize=8.6, color="#111827")
ax.text(38, 59.5, "tidak perlu dicocokkan ke bank", fontsize=7.4, color="#6b7280")
ax.text(38, 55, "QRIS/Transfer: mutasi masuk", fontsize=8.6, color="#111827")
ax.text(38, 51.5, "selisih MDR dijelaskan sebagai biaya", fontsize=7.4, color="#6b7280")

ax.text(82, 74, "3. Temuan", fontsize=12.5, fontweight="bold", color="#047857", ha="center")
ax.text(82, 70, "hanya yang tidak cocok", fontsize=8.4, color="#4b5563", ha="center")
ax.text(70, 63, "Sudah sesuai", fontsize=9.4, fontweight="bold", color="#047857")
ax.text(70, 59.5, "tidak perlu tindakan", fontsize=7.4, color="#6b7280")
ax.text(70, 55, "Perlu dicek", fontsize=9.4, fontweight="bold", color="#b45309")
ax.text(70, 51.5, "dengan alasan yang bisa dibaca", fontsize=7.4, color="#6b7280")

# arrows
for x1, x2 in [(32.4, 35.6), (64.4, 67.6)]:
    ax.add_patch(FancyArrowPatch((x1, 63), (x2, 63), arrowstyle="-|>", mutation_scale=16, linewidth=1.4, color="#6b7280"))

# bottom rules
ax.text(4, 38, "Tiga aturan yang menghilangkan kebingungan", fontsize=10.5, fontweight="bold", color="#111827")
rules = [
    ("Tunai bukan piutang", "Penjualan tunai dianggap lunas saat dicatat, jadi tidak muncul sebagai selisih."),
    ("Setor ke bank bukan penjualan", "Memindahkan uang tunai ke rekening tidak menambah penjualan baru."),
    ("Biaya layanan bukan uang hilang", "Potongan MDR atau fee marketplace dicatat sebagai biaya, bukan selisih."),
]
for i, (head, body) in enumerate(rules):
    y = 31 - i * 8
    box(4, y - 5.4, 92, 6.6, "#ffffff", "#e5e7eb")
    ax.text(6, y - 0.6, head, fontsize=9.2, fontweight="bold", color="#111827")
    ax.text(6, y - 3.6, body, fontsize=7.8, color="#4b5563")

ax.text(50, 4, "Pemilik usaha tidak perlu belajar istilah akuntansi untuk tahu kasnya cocok atau tidak.", fontsize=8.8, color="#6b7280", ha="center")

fig.savefig("docs/diagrams/kas-model.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

