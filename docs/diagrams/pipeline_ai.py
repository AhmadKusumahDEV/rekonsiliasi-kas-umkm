import textwrap
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

stages = [
    ("1. UNGGAH", "ANDA", "#ffffff", "#1f2937", [
        "Excel, CSV, atau ekspor bank",
        "maks 10 MB per file",
        "jenis transaksi dideteksi",
    ]),
    ("2. NORMALISASI", "SISTEM", "#eef2ff", "#3730a3", [
        "kolom dari nama header",
        "Rp, tanggal, kanal dirapikan",
        "duplikat dan error per baris",
    ]),
    ("3. SKOR", "SISTEM + AI", "#f5f3ff", "#6d28d9", [
        "aturan pasti lebih dulu",
        "skor keyakinan 0 sampai 1",
        "alasan tiap pasangan",
    ]),
    ("4. KEPUTUSAN", "SISTEM", "#fff7ed", "#b45309", [
        "skor tinggi disimpan otomatis",
        "skor sedang ke antrean",
        "skor rendah tidak dipasangkan",
    ]),
    ("5. TINJAU", "ANDA", "#ffffff", "#1f2937", [
        "hanya sisa yang ambigu",
        "setujui semua atau satu per satu",
        "batalkan kapan saja",
    ]),
    ("6. SELESAI", "SISTEM", "#ecfdf5", "#047857", [
        "audit dan SSE diperbarui",
        "laporan dan ekspor siap",
        "fallback tanpa AI tetap jalan",
    ]),
]

fig, ax = plt.subplots(figsize=(16, 7.4))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Pipeline impor otomatis yang diusulkan", fontsize=18, fontweight="bold", color="#111827")
fig.text(0.045, 0.915, "Satu unggahan, keputusan otomatis untuk kasus yang yakin, manusia hanya meninjau sisanya", fontsize=10.5, color="#6b7280")

left, right = 4, 96
gap = 3.0
n = len(stages)
w = (right - left - gap * (n - 1)) / n
top, h = 86, 34

for i, (title, lane, fill, edge, bullets) in enumerate(stages):
    x = left + i * (w + gap)
    box = FancyBboxPatch((x, top - h), w, h, boxstyle="round,pad=0.02,rounding_size=0.9",
                         linewidth=1.5, facecolor=fill, edgecolor=edge, zorder=2)
    ax.add_patch(box)
    ax.text(x + w / 2, top - 3.8, lane, fontsize=7.4, fontweight="bold", color=edge, ha="center")
    ax.text(x + w / 2, top - 8.6, title, fontsize=10.0, fontweight="bold", color="#111827", ha="center")
    y = top - 14.0
    for line in bullets:
        for part in textwrap.wrap(line, 24):
            ax.text(x + 1.0, y, part, fontsize=7.1, color="#374151", ha="left", va="top")
            y -= 3.3
        y -= 1.2
    if i < n - 1:
        ax.add_patch(FancyArrowPatch((x + w + 0.35, top - h / 2), (x + w + gap - 0.35, top - h / 2),
                                     arrowstyle="-|>", mutation_scale=12, linewidth=1.2, color="#6b7280"))

band_y = 30
ax.text(left, band_y + 10, "Ambang keputusan (dapat dikonfigurasi)", fontsize=10.5, fontweight="bold", color="#111827")
bands = [
    ("Skor >= 0,95", "Simpan otomatis, masuk audit, bisa dibatalkan", "#d1fae5", "#047857", 41),
    ("0,60 - 0,95", "Antrean tinjauan, disarankan AI", "#fef3c7", "#b45309", 32),
    ("< 0,60", "Tidak dipasangkan, tampil sebagai selisih", "#fee2e2", "#b91c1c", 21),
]
x = left
for label, desc, fill, edge, width in bands:
    seg = FancyBboxPatch((x, band_y - 11), width, 11, boxstyle="round,pad=0.02,rounding_size=0.5",
                         linewidth=1.2, facecolor=fill, edgecolor=edge)
    ax.add_patch(seg)
    ax.text(x + width / 2, band_y - 4.0, label, fontsize=9.4, fontweight="bold", color=edge, ha="center")
    ax.text(x + width / 2, band_y - 8.0, desc, fontsize=7.8, color="#374151", ha="center")
    x += width + 0.8

ax.text(left, 6, "Selisih nominal, kandidat ganda, atau referensi berbeda selalu masuk tinjauan manusia.",
        fontsize=8.6, color="#6b7280")

fig.savefig("docs/diagrams/pipeline-ai.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

