import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import FancyBboxPatch, FancyArrowPatch

fig, ax = plt.subplots(figsize=(13.5, 7.2))
ax.set_xlim(0, 100)
ax.set_ylim(0, 100)
ax.axis("off")
fig.text(0.045, 0.955, "Pola container: API, matcher, AI gateway, dan database", fontsize=17.5, fontweight="bold", color="#111827")
fig.text(0.045, 0.912, "Matcher dan AI gateway bersifat opsional; keduanya berjalan di jaringan internal", fontsize=10.4, color="#6b7280")

def box(x, y, w, h, fill, edge, title, items):
    ax.add_patch(FancyBboxPatch((x, y), w, h, boxstyle="round,pad=0.02,rounding_size=1.0",
                                linewidth=1.5, facecolor=fill, edgecolor=edge))
    ax.text(x + w / 2, y + h - 4.5, title, fontsize=10.6, fontweight="bold", color=edge, ha="center")
    ty = y + h - 10
    for item in items:
        ax.text(x + 2, ty, item, fontsize=7.8, color="#374151", ha="left", va="top")
        ty -= 3.6

box(4, 52, 24, 28, "#eef2ff", "#3730a3", "API Go", ["impor dan validasi", "commit ke ledger", "SSE ke browser"])
box(33, 52, 24, 28, "#f5f3ff", "#6d28d9", "Matcher opsional", ["skor dan kandidat", "assignment", "batch atau sinkron"])
box(62, 52, 24, 28, "#ecfdf5", "#047857", "Postgres", ["sumber kebenaran", "audit dan keputusan", "dibagi antar container"])

box(33, 12, 24, 24, "#fff7ed", "#b45309", "AI gateway opsional", ["kunci hanya di sini", "kuota dan cache", "provider netral"])
box(62, 12, 24, 24, "#fef2f2", "#b91c1c", "Gemini", ["hanya kandidat ambigu", "JSON terstruktur", "timeout dan fallback"])

ax.add_patch(FancyArrowPatch((28.4, 66), (32.6, 66), arrowstyle="-|>", mutation_scale=14, linewidth=1.3, color="#6b7280"))
ax.add_patch(FancyArrowPatch((57.4, 66), (61.6, 66), arrowstyle="-|>", mutation_scale=14, linewidth=1.3, color="#6b7280"))
ax.add_patch(FancyArrowPatch((45, 52), (45, 36.5), arrowstyle="-|>", mutation_scale=14, linewidth=1.3, color="#6b7280"))
ax.add_patch(FancyArrowPatch((57.4, 24), (61.6, 24), arrowstyle="-|>", mutation_scale=14, linewidth=1.3, color="#6b7280"))

ax.text(50, 6, "Matcher dan AI gateway tidak membuka port publik; hanya API yang dapat menjangkau keduanya.", fontsize=8.8, color="#6b7280", ha="center")

fig.savefig("docs/diagrams/container-matching.png", dpi=200, bbox_inches="tight", facecolor="white")
plt.close(fig)

