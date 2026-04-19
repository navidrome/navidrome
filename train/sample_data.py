"""
Generate gru4rec_input_sample.json and gru4rec_output_sample.json
from REAL data by running the actual GRU4Rec preprocessing pipeline.

Usage:
    python generate_real_samples.py

Requires:
    - Your data in data/entities/ and data/relations/ (same as gru4rec.py)
    - pip install torch pandas numpy tqdm
    - gru4rec.py in the same directory (or on PYTHONPATH)
"""

import json
import sys
import os

# ── Import your actual code ───────────────────────────────────────────────────
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gru4rec import cfg, prepare_data   # uses your real cfg and pipeline

# ── Run the real preprocessing pipeline ──────────────────────────────────────
print("Loading and preprocessing real data …")
data = prepare_data(cfg)   # reads .idomaar files, filters, builds vocabs, splits

item2idx    = data["item2idx"]
user2idx    = data["user2idx"]
idx2item    = {v: k for k, v in item2idx.items()}
train_seqs  = data["train_seqs"]
test_seqs   = data["test_seqs"]

print(f"  {len(item2idx)} items, {len(user2idx)} users")
print(f"  {len(train_seqs)} train sessions, {len(test_seqs)} test sessions")

# ── Pick a representative test session (at least 6 items long) ───────────────
sample_seq = next(
    (s for s in test_seqs if len(s["item_idxs"]) >= 6),
    test_seqs[0]   # fallback: first test session
)

prefix_idxs  = sample_seq["item_idxs"][:5]   # 5-item prefix
prefix_tracks = [idx2item[i] for i in prefix_idxs]

# playratio values come from build_sequences → stored in the sequence
playratios = sample_seq["playratios"][:5]

user_idx = sample_seq["user_idx"]
# Reverse-lookup raw user_id from user2idx
idx2user = {v: k for k, v in user2idx.items()}
user_id  = idx2user.get(user_idx, "unknown")

# ── INPUT SAMPLE ─────────────────────────────────────────────────────────────
input_sample = {
    "_schema_version": "1.0",
    "_description": (
        "One inference request for GRU4Rec (generated from real 30Music data). "
        "'prefix_item_idxs' are vocab-mapped (1-based) track indices. "
        "'prefix_track_ids' are the original 30Music track IDs for traceability. "
        "'playratios' are normalised play-ratio weights in [0.1, 1.0]. "
        "'top_n' must match cfg['top_n'] used at training."
    ),
    "session_id":        sample_seq["session_id"],
    "user_id":           int(user_id) if str(user_id).isdigit() else user_id,
    "user_idx":          user_idx,
    "request_timestamp": "FILL_IN_AT_SERVING_TIME",
    "prefix_track_ids":  [int(t) for t in prefix_tracks],
    "prefix_item_idxs":  prefix_idxs,
    "playratios":        [round(float(r), 4) for r in playratios],
    "exclude_item_idxs": prefix_idxs,   # mask out already-heard tracks
    "top_n":             cfg["top_n"],
}

# ── OUTPUT SAMPLE — run a real forward pass ───────────────────────────────────
import torch
from gru4rec import GRU4Rec

num_items = len(item2idx)
num_users = len(user2idx)

# Build model (random weights — replace state_dict load below if you have a checkpoint)
model = GRU4Rec(num_items=num_items, num_users=num_users, cfg=cfg)

# ── Load trained weights if available ────────────────────────────────────────
checkpoint = "best_gru4rec.pt"
if os.path.exists(checkpoint):
    model.load_state_dict(torch.load(checkpoint, map_location="cpu"))
    print(f"  Loaded weights from {checkpoint}")
else:
    print(f"  WARNING: {checkpoint} not found — using random weights. "
          "Scores will be meaningless but structure is correct.")

model.eval()
device = torch.device("cpu")

all_item_emb = model.item_emb.weight[1:].to(device)   # shape: (num_items, embed_dim)

max_len = len(prefix_idxs)
prefix_t = torch.zeros(1, max_len, dtype=torch.long)
for i, idx in enumerate(prefix_idxs):
    prefix_t[0, max_len - len(prefix_idxs) + i] = idx   # left-pad

user_t       = torch.tensor([user_idx], dtype=torch.long)
exclude_sets = [set(prefix_idxs)]

import time
t0 = time.perf_counter()
with torch.no_grad():
    top20_idxs = model.predict_top_n(prefix_t, user_t, all_item_emb, cfg["top_n"], exclude_sets)[0]
latency_ms = round((time.perf_counter() - t0) * 1000, 3)

# Compute raw scores for the top-20
with torch.no_grad():
    session_repr = model.encode_session(prefix_t, user_t)
    all_scores   = (session_repr @ all_item_emb.T).squeeze(0)
top20_scores = [round(float(all_scores[i - 1]), 4) for i in top20_idxs]
top20_track_ids = [int(idx2item[i]) for i in top20_idxs]

output_sample = {
    "_schema_version": "1.0",
    "_description": (
        "One inference response from GRU4Rec (generated from real 30Music data). "
        "'recommended_item_idxs' are vocab-mapped (1-based) indices in rank order. "
        "'recommended_track_ids' are the corresponding original 30Music track IDs. "
        "'scores' are raw dot-product similarities (higher = more relevant). "
        "'latency_ms' is the wall-clock time for this predict_top_n call."
    ),
    "session_id":           sample_seq["session_id"],
    "user_id":              int(user_id) if str(user_id).isdigit() else user_id,
    "request_timestamp":    "FILL_IN_AT_SERVING_TIME",
    "response_timestamp":   "FILL_IN_AT_SERVING_TIME",
    "recommended_item_idxs":  top20_idxs,
    "recommended_track_ids":  top20_track_ids,
    "scores":                 top20_scores,
    "top_n":                  cfg["top_n"],
    "model_version":          "gru4rec_30music_v1",
    "latency_ms":             latency_ms,
}

# ── Write files ───────────────────────────────────────────────────────────────
with open("gru4rec_input_sample.json", "w") as f:
    json.dump(input_sample, f, indent=2)

with open("gru4rec_output_sample.json", "w") as f:
    json.dump(output_sample, f, indent=2)

print("\n✅  gru4rec_input_sample.json")
print("✅  gru4rec_output_sample.json")
print(f"\n   session : {sample_seq['session_id']}")
print(f"   prefix  : {prefix_tracks}")
print(f"   top-5   : {top20_track_ids[:5]} …")
print(f"   latency : {latency_ms} ms")