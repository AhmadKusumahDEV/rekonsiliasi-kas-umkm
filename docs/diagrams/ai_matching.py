import textwrap
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

fig, ax = plt.subplots(figsize=(14, 7.6))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Dua jalur pencocokan: aturan untuk data rapi, AI untuk data acak", fontsize=17.5, fontweight="bold", color="#111827")
fig.text(0.045, 0.912, "AI tidak pernah membaca seluruh file; ia hanya memilih di antara kandidat yang sudah disaring", fontsize=10.4, color="#6b7280")

def box(x, y, w, h, fill, edge, title, items, wrap=24):
    ax.add_patch(FancyBboxPatch((x, y), w, h, boxstyle="round,pad=0.02,rounding_size=0.9",
                                linewidth=1.4, facecolor=fill, edgecolor=edge))
    ax.text(x + w / 2, y + h - 4.2, title, fontsize=10.0, fontweight="bold", color=edge, ha="center")
    ty = y + h - 10.2
    for item in items:
        for part in textwrap.wrap(item, wrap):
            ax.text(x + 1.2, ty, part, fontsize=7.0, color="#374151", ha="left", va="top")
            ty -= 3.1
        ty -= 0.8

box(4, 62, 29, 26, "#eef2ff", "#3730a3", "Data rapi", [
    "kolom dan format konsisten",
    "referensi, nominal, tanggal jelas",
])
box(37, 62, 29, 26, "#f5f3ff", "#6d28d9", "Data acak", [
    "nama kolom bebas",
    "Rp dan tanggal campur",
    "keterangan tanpa referensi",
])

box(4, 32, 29, 26, "#ffffff", "#3730a3", "Aturan deterministik", [
    "referensi sama",
    "nominal sama",
    "kanal dan tanggal dekat",
    "unik kedua sisi",
])
box(37, 32, 29, 26, "#ffffff", "#6d28d9", "Normalisasi lalu kandidat", [
    "petakan kolom dan nilai",
    "ambil INV/TRX",
    "saring nominal, 0-7 hari",
])
box(37, 6, 29, 22, "#faf5ff", "#6d28d9", "AI memilih kandidat", [
    "hanya skor 0,60-0,95",
    "alasan wajib",
    "ID tertutup",
])

box(70, 10, 26, 50, "#ecfdf5", "#047857", "Hasil", [
    "pasangan tersimpan otomatis",
    "atau masuk Verifikasi",
    "keputusan dapat dibatalkan",
])

ax.add_patch(FancyArrowPatch((18.5, 62), (18.5, 58.5), arrowstyle="-|>", mutation_scale=14, linewidth=1.3, color="#6b7280"))
ax.add_patch(FancyArrowPatch((51.5, 62), (51.5, 58.5), arrowstyle="-|>", mutation_scale=14, linewidth=1.3, color="#6b7280"))
ax.add_patch(FancyArrowPatch((51.5, 32), (51.5, 28.5), arrowstyle="-|>", mutation_scale=14, linewidth=1.3, color="#6b7280"))
ax.add_patch(FancyArrowPatch((33.4, 45), (36.6, 45), arrowstyle="-|>", mutation_scale=13, linewidth=1.2, color="#9ca3af"))
ax.add_patch(FancyArrowPatch((66.4, 45), (69.6, 45), arrowstyle="-|>", mutation_scale=13, linewidth=1.2, color="#9ca3af"))
ax.add_patch(FancyArrowPatch((66.4, 17), (69.6, 17), arrowstyle="-|>", mutation_scale=13, linewidth=1.2, color="#9ca3af"))

fig.savefig("docs/diagrams/ai-matching.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

