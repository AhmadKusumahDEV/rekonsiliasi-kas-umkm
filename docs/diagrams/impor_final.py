import textwrap
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

stages = [
    ("1. SATU LAYAR", "ANDA", "#ffffff", "#1f2937", [
        "Slot file penjualan",
        "Slot file penerimaan",
        "Salah satu boleh kosong",
    ]),
    ("2. BANTU RAPUKAN", "OPSIONAL", "#f5f3ff", "#6d28d9", [
        "Excel/CSV apa adanya",
        "aturan dulu, AI bila perlu",
        "hasil: 7 kolom kanonik",
    ]),
    ("3. PRATINJAU", "ANDA", "#ffffff", "#1f2937", [
        "nilai asli vs hasil",
        "error per baris",
        "konfirmasi sebelum simpan",
    ]),
    ("4. SIMPAN", "SISTEM", "#eef2ff", "#3730a3", [
        "batch kanonik per jenis",
        "atomik, bisa dibatalkan",
        "audit dan SSE",
    ]),
    ("5. COCOKKAN", "SISTEM", "#ecfdf5", "#047857", [
        "cocok otomatis bila yakin",
        "sisanya masuk temuan",
        "satu sisi: menunggu",
    ]),
]

fig, ax = plt.subplots(figsize=(16, 7.6))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Bentuk akhir impor: satu layar, dua slot, konversi terpisah", fontsize=17.5, fontweight="bold", color="#111827")
fig.text(0.045, 0.912, "Format kanonik tetap ketat; keragaman file diselesaikan di fitur konversi, bukan di pencatatan", fontsize=10.5, color="#6b7280")

left, right = 4, 96
gap = 2.6
n = len(stages)
w = (right - left - gap * (n - 1)) / n
top, h = 84, 36

for i, (title, lane, fill, edge, bullets) in enumerate(stages):
    x = left + i * (w + gap)
    ax.add_patch(FancyBboxPatch((x, top - h), w, h, boxstyle="round,pad=0.02,rounding_size=0.9",
                                linewidth=1.5, facecolor=fill, edgecolor=edge))
    ax.text(x + w / 2, top - 3.6, lane, fontsize=7.2, fontweight="bold", color=edge, ha="center")
    ax.text(x + w / 2, top - 8.0, title, fontsize=9.6, fontweight="bold", color="#111827", ha="center")
    y = top - 13.4
    for b in bullets:
        for part in textwrap.wrap(b, 20):
            ax.text(x + 0.9, y, part, fontsize=7.0, color="#374151", ha="left", va="top")
            y -= 3.2
        y -= 1.0
    if i < n - 1:
        ax.add_patch(FancyArrowPatch((x + w + 0.3, top - h / 2), (x + w + gap - 0.3, top - h / 2),
                                     arrowstyle="-|>", mutation_scale=11, linewidth=1.1, color="#6b7280"))

ax.text(50, 34, "Dua sisi hanya untuk ketelitian per transaksi. Satu slot kosong tidak menghasilkan selisih, hanya status menunggu.",
        fontsize=9.2, color="#4b5563", ha="center")
ax.text(50, 22, "Yang tidak dibangun sekarang: deteksi otomatis jenis file dan pencocokan satu-unggahan penuh.",
        fontsize=9.2, color="#9ca3af", ha="center")

fig.savefig("docs/diagrams/impor-final.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

