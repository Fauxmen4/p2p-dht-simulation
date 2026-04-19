import os
import json

import matplotlib.pyplot as plt
import numpy as np


PROJECT_PATH = os.path.join(os.path.dirname(__file__), "..")

metrics_dir = os.path.join(PROJECT_PATH, "data", "metrics")
files = sorted(f for f in os.listdir(metrics_dir))
latest = files[-1]

with open(os.path.join(metrics_dir, latest)) as f:
    data = json.load(f)

print(f"Using metrics dump: {latest}")


# Data
handled_rpcs = data["handled_rpcs"]
sent_rpcs = data["sent_rpcs"]
hops_count = data["key_lookups"]["success_hops_count"]

fig, axes = plt.subplots(2, 2, figsize=(16, 12))

# Top left: handledRPCs
ax = axes[0][0]
n, bins, patches = ax.hist(handled_rpcs, bins=30, color="steelblue", edgecolor="black", alpha=0.8)
avg = np.mean(handled_rpcs)
ax.axvline(avg, color='black', linestyle='-', linewidth=2, label=f'avg = {avg:.1f}')
ax.axvline(np.max(handled_rpcs), color='black', linestyle='--', linewidth=2, label=f'max = {np.max(handled_rpcs)}')
ax.set_title("Распределение обработанных RPC", fontsize=14)
ax.set_xlabel("handled_rpcs_per_node")
ax.set_ylabel("Количество узлов")
ax.grid(axis="y", alpha=0.3)
ax.legend()

# Top right: sentRPCs
ax = axes[0][1]
n, bins, patches = ax.hist(sent_rpcs, bins=30, color="darkorange", edgecolor="black", alpha=0.8)
avg = np.mean(sent_rpcs)
ax.axvline(avg, color='black', linestyle='-', linewidth=2, label=f'avg = {avg:.1f}')
ax.axvline(np.max(sent_rpcs), color='black', linestyle='--', linewidth=2, label=f'max = {np.max(sent_rpcs)}')
ax.set_title("Распределение отправленных RPC", fontsize=14)
ax.set_xlabel("sent_rpcs_per_node")
ax.set_ylabel("Количество узлов")
ax.grid(axis="y", alpha=0.3)
ax.legend()

# Bottom left: hops_count
ax = axes[1][0]
bins_hops = np.arange(min(hops_count), max(hops_count) + 2) - 0.5
ax.hist(hops_count, bins=bins_hops, edgecolor='black', color="steelblue")
ax.set_title("Распределение числа переходов среди ищущих узлов", fontsize=14)
ax.set_xlabel("hop_count")
ax.set_ylabel("Количество узлов")
ax.set_xticks(np.arange(min(hops_count), max(hops_count) + 1, 1))
ax.grid(axis='y', alpha=0.3)

# Bottom right: empty
axes[1][1].axis('off')

# Export
output_path = os.path.join(PROJECT_PATH, "plotter", f"{latest}.png")
plt.tight_layout()
plt.savefig(output_path, format="png", dpi=150)
plt.show()


# import os
# import json
# import matplotlib.pyplot as plt
# import numpy as np

# PROJECT_PATH = os.path.join(os.path.dirname(__file__), "..")
# metrics_dir = os.path.join(PROJECT_PATH, "data", "metrics")
# files = sorted(f for f in os.listdir(metrics_dir))
# latest = files[-1]
# with open(os.path.join(metrics_dir, latest)) as f:
#     data = json.load(f)
# print(f"Using metrics dump: {latest}")

# # Data
# handled_rpcs = data["handled_rpcs"]
# hops_count_with_nulls = data["key_lookups"]["success_hops_count"]

# hops_count = []
# for hops in hops_count_with_nulls:
#     if hops == 0: continue
#     hops_count.append(hops)

# fig, axes = plt.subplots(1, 2, figsize=(16, 6))

# # Left: handledRPCs
# ax = axes[0]
# n, bins, patches = ax.hist(handled_rpcs, bins=30, color="steelblue", edgecolor="black", alpha=0.8)
# avg = np.mean(handled_rpcs)
# ax.axvline(avg, color='black', linestyle='-', linewidth=2, label=f'avg = {avg:.1f}')
# ax.axvline(np.max(handled_rpcs), color='black', linestyle='--', linewidth=2, label=f'max = {np.max(handled_rpcs)}')
# ax.set_title("Распределение обработанных RPC", fontsize=14)
# ax.set_xlabel("load_per_node")
# ax.set_ylabel("Количество узлов")
# ax.grid(axis="y", alpha=0.3)
# ax.legend()

# # Right: hops_count
# ax = axes[1]
# bins_hops = np.arange(min(hops_count), max(hops_count) + 2) - 0.5
# ax.hist(hops_count, bins=bins_hops, edgecolor='black', color="C1")
# ax.set_title("Распределение числа переходов среди ищущих узлов", fontsize=14)
# ax.set_xlabel("hop_count")
# ax.set_ylabel("Количество узлов")
# ax.set_xticks(np.arange(min(hops_count), max(hops_count) + 1, 1))
# ax.grid(axis='y', alpha=0.3)

# # Export
# output_path = os.path.join(PROJECT_PATH, "plotter", f"{latest}.png")
# plt.tight_layout()
# plt.savefig(output_path, format="png", dpi=150)
# plt.show()