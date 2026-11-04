import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch

fig, ax = plt.subplots(figsize=(13.5, 8.2))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Tiga layar yang dilihat pemilik UMKM", fontsize=18, fontweight="bold", color="#111827")
fig.text(0.045, 0.915, "Tanpa istilah teknis: masukkan data, tunggu diperiksa, putuskan yang perlu dicek", fontsize=10.5, color="#6b7280")

panels = [
    ("1. Masukkan data", "#eef2ff", "#3730a3"),
    ("2. EFA memeriksa", "#f5f3ff", "#6d28d9"),
    ("3. Yang perlu dicek", "#ecfdf5", "#047857"),
]

w, h = 27, 62
gap = 6
left = (100 - (w * 3 + gap * 2)) / 2
top = 86

for i, (title, fill, edge) in enumerate(panels):
    x = left + i * (w + gap)
    ax.add_patch(FancyBboxPatch((x, top - h), w, h, boxstyle="round,pad=0.02,rounding_size=1.2",
                                linewidth=1.6, facecolor="#ffffff", edgecolor="#d1d5db"))
    ax.add_patch(FancyBboxPatch((x, top - 9), w, 9, boxstyle="round,pad=0.02,rounding_size=1.0",
                                linewidth=0, facecolor=fill, edgecolor=fill))
    ax.text(x + w / 2, top - 4.6, title, fontsize=10.4, fontweight="bold", color=edge, ha="center")

# panel 1
x1 = left
ax.add_patch(FancyBboxPatch((x1 + 2.2, top - 33), w - 4.4, 18, boxstyle="round,pad=0.02,rounding_size=1.0",
                            linewidth=1.2, linestyle="--", facecolor="#f9fafb", edgecolor="#9ca3af"))
ax.text(x1 + w / 2, top - 23.5, "Tarik file ke sini", fontsize=9.6, color="#374151", ha="center")
ax.text(x1 + w / 2, top - 27.2, "Excel, CSV, atau ekspor bank", fontsize=7.4, color="#6b7280", ha="center")
ax.add_patch(FancyBboxPatch((x1 + 4.2, top - 41), w - 8.4, 5.2, boxstyle="round,pad=0.02,rounding_size=0.8",
                            linewidth=0, facecolor="#1f2937"))
ax.text(x1 + w / 2, top - 38.4, "Periksa kas", fontsize=8.8, color="white", ha="center", fontweight="bold")
ax.text(x1 + w / 2, top - 47, "Tidak perlu ubah format", fontsize=7.6, color="#6b7280", ha="center")
ax.text(x1 + w / 2, top - 51, "Data lama tidak diubah", fontsize=7.6, color="#6b7280", ha="center")

# panel 2
x2 = left + w + gap
ax.text(x2 + 2.2, top - 14.5, "92 transaksi dibaca", fontsize=9.2, fontweight="bold", color="#111827")
ax.add_patch(FancyBboxPatch((x2 + 2.2, top - 20), w - 4.4, 2.6, boxstyle="round,pad=0.02,rounding_size=0.4",
                            linewidth=0, facecolor="#e5e7eb"))
ax.add_patch(FancyBboxPatch((x2 + 2.2, top - 20), (w - 4.4) * 0.94, 2.6, boxstyle="round,pad=0.02,rounding_size=0.4",
                            linewidth=0, facecolor="#7c3aed"))
ax.text(x2 + 2.2, top - 26, "87 sudah sesuai", fontsize=8.4, color="#047857", fontweight="bold")
ax.text(x2 + 2.2, top - 31, "5 perlu dicek", fontsize=8.4, color="#b45309", fontweight="bold")
ax.text(x2 + 2.2, top - 36, "0 gagal dibaca", fontsize=8.4, color="#6b7280")
ax.text(x2 + w / 2, top - 44, "Boleh ditinggal", fontsize=7.6, color="#6b7280", ha="center")
ax.text(x2 + w / 2, top - 48, "Hasil tersimpan otomatis", fontsize=7.6, color="#6b7280", ha="center")

# panel 3
x3 = left + 2 * (w + gap)
ax.text(x3 + 2.2, top - 14, "87 sesuai", fontsize=8.6, color="#047857", fontweight="bold")
ax.text(x3 + 2.2, top - 19, "5 perlu dicek", fontsize=9.4, color="#b45309", fontweight="bold")
for j in range(3):
    yy = top - 24 - j * 6.4
    ax.add_patch(FancyBboxPatch((x3 + 2.2, yy - 5), w - 4.4, 5, boxstyle="round,pad=0.02,rounding_size=0.6",
                                linewidth=1.0, facecolor="#fffbeb", edgecolor="#fcd34d"))
    ax.text(x3 + 3.2, yy - 2.5, ["INV-104 Rp80.000", "TRF-88 Rp250.000", "INV-102 Rp100.000"][j],
            fontsize=7.4, color="#111827", va="center")
ax.add_patch(FancyBboxPatch((x3 + 2.2, top - 48), w - 4.4, 5, boxstyle="round,pad=0.02,rounding_size=0.8",
                            linewidth=0, facecolor="#047857"))
ax.text(x3 + w / 2, top - 45.5, "Setujui semua yang aman", fontsize=7.6, color="white", ha="center", fontweight="bold")
ax.text(x3 + w / 2, top - 53, "atau tinjau satu per satu", fontsize=7.4, color="#6b7280", ha="center")

ax.text(50, 9, "Selesai dalam hitungan menit, bukan satu sesi penuh.", fontsize=9.4, color="#6b7280", ha="center")

fig.savefig("docs/diagrams/ux-sederhana.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

