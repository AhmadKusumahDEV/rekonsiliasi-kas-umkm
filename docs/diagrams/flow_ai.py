import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

LANE = {
    "ANDA": ("#ffffff", "#1f2937", "#e5e7eb"),
    "EFA": ("#eef2ff", "#3730a3", "#c7d2fe"),
    "AI": ("#f5f3ff", "#6d28d9", "#ddd6fe"),
    "HASIL": ("#ecfdf5", "#047857", "#a7f3d0"),
}

def flow(nodes, notes, path, title):
    n = len(nodes)
    height = 2.0 + n * 1.18
    fig, ax = plt.subplots(figsize=(11, height))
    ax.set_xlim(0, 10)
    ax.set_ylim(0, 100)
    ax.axis("off")
    fig.text(0.06, 0.965, title, fontsize=17, fontweight="bold", color="#111827")
    fig.text(0.06, 0.932, "Alur EFA — impor dan pencocokan", fontsize=10, color="#6b7280")
    top, bottom = 88, 6
    spacing = (top - bottom) / (n - 1)
    box_h = min(6.2, spacing * 0.64)
    box_w = 6.0
    cx = 3.5
    centers = []
    for i, (lane, label) in enumerate(nodes):
        y = top - i * spacing
        centers.append(y)
        fill, edge, _ = LANE[lane]
        box = FancyBboxPatch(
            (cx - box_w / 2, y - box_h / 2), box_w, box_h,
            boxstyle="round,pad=0.02,rounding_size=0.55",
            linewidth=1.4, facecolor=fill, edgecolor=edge, zorder=2,
        )
        ax.add_patch(box)
        ax.text(cx - box_w / 2 + 0.18, y + box_h / 2 - 0.55, lane,
                fontsize=7.5, fontweight="bold", color=edge, ha="left", va="center")
        ax.text(cx, y - 0.28, label, fontsize=10.6, color="#111827",
                ha="center", va="center", linespacing=1.35)
        ax.text(cx - box_w / 2 - 0.45, y, str(i + 1), fontsize=9,
                color="#9ca3af", ha="center", va="center")
    for i in range(n - 1):
        y1 = centers[i] - box_h / 2
        y2 = centers[i + 1] + box_h / 2
        ax.add_patch(FancyArrowPatch(
            (cx, y1), (cx, y2),
            arrowstyle="-|>", mutation_scale=14,
            linewidth=1.2, color="#6b7280", zorder=1,
        ))
    for idx, note, tone in notes:
        y = centers[idx]
        ax.plot([cx + box_w / 2, 7.55], [y, y], color="#d1d5db", linewidth=0.8, zorder=0)
        ax.text(7.7, y, note, fontsize=8.8, color=tone, va="center", ha="left", wrap=True)
    fig.savefig(path, dpi=200, bbox_inches="tight", facecolor="white")
    plt.close(fig)

current = [
    ("ANDA", "Unduh template CSV"),
    ("ANDA", "Susun 7 kolom di Excel,\nsimpan sebagai CSV UTF-8"),
    ("ANDA", "Unggah file\nmaks 500 baris, 1 MB"),
    ("EFA", "Validasi header, tanggal,\nnominal, ID unik"),
    ("EFA", "Pratinjau: baris baru,\nduplikat, error per baris"),
    ("ANDA", "Simpan impor"),
    ("ANDA", "Buka Cocokkan kas"),
    ("EFA", "Hitung kandidat: referensi, kanal,\nnominal, selisih maksimal 3 hari"),
    ("ANDA", "Konfirmasi usulan\nsatu per satu"),
    ("ANDA", "Selesaikan Perlu diperiksa:\nalokasi manual, biaya, kandidat ganda"),
    ("HASIL", "Laporan dan ekspor tersedia"),
]

current_notes = [
    (1, "Konversi manual:\nformat harus persis", "#dc2626"),
    (3, "Satu sel salah\nmenolak seluruh file", "#dc2626"),
    (8, "Sembilan transaksi,\nsembilan kartu keputusan", "#dc2626"),
    (9, "Pekerjaan berulang\ntiap bulan", "#dc2626"),
]

flow(current, current_notes, "docs/diagrams/alur-sekarang.png",
     "Alur sekarang: impor lalu cocokkan manual")

ai = [
    ("ANDA", "Unggah file apa adanya\n(Excel, CSV, ekspor bank)"),
    ("AI", "Normalisasi kolom dan nilai:\nRp, tanggal, kanal, duplikat"),
    ("EFA", "Pratinjau ringkas:\n92 siap, 3 perlu koreksi"),
    ("AI", "Cocokkan dan beri skor keyakinan\nbeserta alasan"),
    ("EFA", "Simpan otomatis pasangan\nkeyakinan tinggi, tercatat di audit"),
    ("ANDA", "Tinjau hanya yang ambigu\natau berisiko (mis. 5 kartu)"),
    ("ANDA", "Setujui semua saran\natau periksa satu per satu"),
    ("HASIL", "Laporan dan ekspor siap;\nkeputusan AI bisa dibatalkan"),
]

ai_notes = [
    (1, "Tanpa konversi manual", "#047857"),
    (4, "Hanya 5 dari 92\nperlu dilihat", "#047857"),
    (5, "Satu klik untuk\nsisa yang aman", "#047857"),
    (7, "Alasan dan jejak audit\ntetap tersedia", "#047857"),
]

flow(ai, ai_notes, "docs/diagrams/alur-dengan-ai.png",
     "Alur dengan AI: unggah, AI bereskan, Anda tinjau sisanya")

