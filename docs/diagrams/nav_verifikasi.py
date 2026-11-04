import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

fig, ax = plt.subplots(figsize=(13.5, 7.6))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Dari halaman mencocokkan menjadi antrean verifikasi", fontsize=18, fontweight="bold", color="#111827")
fig.text(0.045, 0.912, "Pengguna tidak lagi mencocokkan transaksi satu per satu; mereka hanya memeriksa yang tidak yakin", fontsize=10.4, color="#6b7280")

def panel(x, title, color, items):
    ax.add_patch(FancyBboxPatch((x, 30), 40, 56, boxstyle="round,pad=0.02,rounding_size=1.2",
                                linewidth=1.5, facecolor="#ffffff", edgecolor="#d1d5db"))
    ax.text(x + 20, 80, title, fontsize=11.6, fontweight="bold", color=color, ha="center")
    y = 72
    for name, note in items:
        ax.text(x + 3, y, name, fontsize=9.4, color="#111827", fontweight="bold")
        if note:
            ax.text(x + 3, y - 3.0, note, fontsize=7.6, color="#6b7280")
        y -= 6.6

panel(4, "Sekarang", "#b91c1c", [
    ("Ringkasan", ""),
    ("Transaksi", ""),
    ("Cocokkan kas", "konfirmasi tiap pasangan"),
    ("Perlu diperiksa", "alokasi manual"),
    ("Impor CSV", "dua file terpisah"),
    ("Riwayat", ""),
])

panel(56, "Usulan", "#047857", [
    ("Ringkasan", "tiga angka: sesuai, dicek, selesai"),
    ("Verifikasi", "hanya sisa yang ambigu, dengan badge"),
    ("Impor", "dua slot dalam satu layar"),
    ("Transaksi", "riwayat lengkap, bisa difilter"),
    ("Laporan", "ekspor dan rekap"),
    ("Riwayat", "audit, termasuk keputusan otomatis"),
])

ax.add_patch(FancyArrowPatch((45, 56), (55, 56), arrowstyle="-|>", mutation_scale=18, linewidth=1.6, color="#6b7280"))

ax.text(50, 19, "Halaman Cocokkan kas dihapus. Verifikasi hanya menampilkan yang butuh keputusan manusia.", fontsize=9.4, color="#4b5563", ha="center")
ax.text(50, 12, "Keputusan otomatis tetap terlihat di Riwayat dan bisa dibatalkan kapan saja.", fontsize=9.4, color="#9ca3af", ha="center")

fig.savefig("docs/diagrams/nav-verifikasi.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

