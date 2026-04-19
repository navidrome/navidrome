"""
Smoke-test for cold-start blending — self-contained, no training dependencies.

Usage:
    python3 test_cold_start.py
    python3 test_cold_start.py --model /home/appuser/work/best_gru4rec.pt
    python3 test_cold_start.py --model /path/model.pt --pop /path/popularity.npy
"""

import argparse
import os
from pathlib import Path

import numpy as np
import torch
import torch.nn as nn
import torch.nn.utils.rnn as rnn_utils

# ── args ─────────────────────────────────────────────────────────────────────
HERE = Path(__file__).resolve().parent

parser = argparse.ArgumentParser()
parser.add_argument("--model",          default=str(HERE.parent / "best_gru4rec.pt"))
parser.add_argument("--pop",            default=str(HERE / ".cache_gru4rec" / "popularity.npy"))
parser.add_argument("--top-n",          type=int, default=10)
parser.add_argument("--ramp",           type=int, default=3)
# MinIO fallback — reads MINIO_URL/USER/PASSWORD/BUCKET from env
parser.add_argument("--minio-model-key",  default=os.environ.get("MINIO_MODEL_KEY", ""))
parser.add_argument("--minio-run-type",   default=os.environ.get("MINIO_MODEL_RUN_TYPE", "finetune"))
parser.add_argument("--minio-pop-key",    default=os.environ.get("MINIO_POPULARITY_KEY", ""))
args = parser.parse_args()

MINIO_URL      = os.environ.get("MINIO_URL", "")
MINIO_USER     = os.environ.get("MINIO_USER", "")
MINIO_PASSWORD = os.environ.get("MINIO_PASSWORD", "")
MINIO_BUCKET   = os.environ.get("MINIO_BUCKET", "gru4rec-models")


def _minio_client():
    import boto3
    return boto3.client(
        "s3",
        endpoint_url=MINIO_URL,
        aws_access_key_id=MINIO_USER,
        aws_secret_access_key=MINIO_PASSWORD,
        region_name="us-east-1",
    )


def _download_model_from_minio() -> str:
    """Auto-discover and download latest finetune (or pretrain) model.pt."""
    s3    = _minio_client()
    key   = args.minio_model_key
    if not key:
        best_key, best_ts = None, None
        paginator = s3.get_paginator("list_objects_v2")
        for run_type in [args.minio_run_type, "pretrain"]:
            for page in paginator.paginate(Bucket=MINIO_BUCKET, Prefix=f"{run_type}/"):
                for obj in page.get("Contents", []):
                    if obj["Key"].endswith("/model.pt"):
                        if best_ts is None or obj["LastModified"] > best_ts:
                            best_key, best_ts = obj["Key"], obj["LastModified"]
            if best_key:
                break
        if not best_key:
            raise RuntimeError(f"No model.pt found in MinIO bucket '{MINIO_BUCKET}'")
        key = best_key

    local = Path("/tmp") / key.replace("/", "_")
    print(f"Downloading model from MinIO: {key}")
    with open(local, "wb") as fh:
        s3.download_fileobj(MINIO_BUCKET, key, fh)
    print(f"Downloaded → {local} ({local.stat().st_size / 1e6:.1f} MB)")
    return str(local)


def _download_pop_from_minio(local_path: str):
    """Download popularity.npy from MinIO."""
    s3  = _minio_client()
    key = args.minio_pop_key
    if not key:
        raise RuntimeError("Set --minio-pop-key or MINIO_POPULARITY_KEY to download popularity.npy")
    print(f"Downloading popularity from MinIO: {key}")
    Path(local_path).parent.mkdir(parents=True, exist_ok=True)
    with open(local_path, "wb") as fh:
        s3.download_fileobj(MINIO_BUCKET, key, fh)
    print(f"Downloaded → {local_path}")


# ── resolve model path ────────────────────────────────────────────────────────
model_path = args.model
if not Path(model_path).exists():
    if MINIO_URL:
        model_path = _download_model_from_minio()
    else:
        raise FileNotFoundError(
            f"Model not found at {model_path}.\n"
            "Set MINIO_URL/MINIO_USER/MINIO_PASSWORD to pull from MinIO."
        )

# ── resolve popularity path ───────────────────────────────────────────────────
pop_path = args.pop
if not Path(pop_path).exists():
    if MINIO_URL and args.minio_pop_key:
        _download_pop_from_minio(pop_path)
    # else: will be flagged as unavailable below

print(f"\n{'='*60}")
print(f"Model : {model_path}")
print(f"Pop   : {pop_path}")
print(f"Ramp  : {args.ramp}")
print(f"Top-N : {args.top_n}")
print(f"{'='*60}\n")


# ── minimal GRU4Rec (mirrors serving/_shared/model.py) ───────────────────────
class GRU4Rec(nn.Module):
    def __init__(self, num_items, cfg):
        super().__init__()
        ed, hd = cfg["embedding_dim"], cfg["hidden_dim"]
        self.item_emb    = nn.Embedding(num_items + 1, ed, padding_idx=0)
        self.emb_dropout = nn.Dropout(0.0)
        self.gru         = nn.GRU(ed, hd, num_layers=cfg["num_layers"], batch_first=True)
        self.dropout     = nn.Dropout(0.0)
        self.output_proj = nn.Linear(hd, ed, bias=False)
        self.layer_norm  = nn.LayerNorm(ed)

    def encode_session(self, prefix, user_idxs):
        x       = self.emb_dropout(self.item_emb(prefix))
        lengths = (prefix != 0).sum(dim=1).cpu().clamp(min=1)
        packed  = rnn_utils.pack_padded_sequence(x, lengths, batch_first=True, enforce_sorted=False)
        _, h_n  = self.gru(packed)
        return self.layer_norm(self.output_proj(self.dropout(h_n[-1])))


# ── load model ────────────────────────────────────────────────────────────────
state     = torch.load(model_path, map_location="cpu", weights_only=True)
num_items = state["item_emb.weight"].shape[0] - 1
ed        = state["item_emb.weight"].shape[1]
hd        = state["gru.weight_hh_l0"].shape[1]
nl        = sum(1 for k in state if k.startswith("gru.weight_hh_l"))

cfg   = {"embedding_dim": ed, "hidden_dim": hd, "num_layers": nl}
model = GRU4Rec(num_items, cfg)
model.load_state_dict(state, strict=True)
model.eval()
all_item_emb = model.item_emb.weight[1:].detach()

print(f"Model loaded: {num_items:,} items  embed={ed}  hidden={hd}  layers={nl}\n")


# ── load popularity ───────────────────────────────────────────────────────────
pop_available = Path(pop_path).exists()
if pop_available:
    pop_scores = torch.from_numpy(np.load(pop_path).astype("float32"))
    print(f"Popularity loaded: shape={tuple(pop_scores.shape)}  "
          f"nonzero={int((pop_scores > 0).sum()):,}\n")
else:
    print(f"[WARN] popularity.npy not found at {pop_path} — cold-start tests skipped\n")


# ── inference helpers ─────────────────────────────────────────────────────────
def alpha(session_len):
    return min(session_len / args.ramp, 1.0)


@torch.no_grad()
def predict(prefix_list, use_cold_start=True):
    prefix = torch.tensor([prefix_list], dtype=torch.long)
    users  = torch.zeros(1, dtype=torch.long)
    slen   = len(prefix_list)

    repr_     = model.encode_session(prefix, users)
    gru_scores = (repr_ @ all_item_emb.T)[0]

    if use_cold_start and pop_available:
        a      = alpha(slen)
        gru_l  = torch.log_softmax(gru_scores, dim=-1)
        # Align popularity to model vocab size (finetune model may be smaller)
        pop_aligned = pop_scores[:num_items]
        pop_l  = torch.log_softmax(pop_aligned, dim=-1)
        scores = a * gru_l + (1 - a) * pop_l
    else:
        a      = 1.0
        scores = gru_scores

    top = torch.topk(scores, args.top_n)
    return (
        [int(i) + 1 for i in top.indices.tolist()],
        [round(float(s), 4) for s in top.values.tolist()],
        round(a, 3),
    )


# ── tests ─────────────────────────────────────────────────────────────────────
ITEMS = [1, 2, 3, 4, 5]

cases = [
    ("Cold start  — session_len=1, alpha=0.33", ITEMS[:1]),
    ("Short       — session_len=2, alpha=0.67", ITEMS[:2]),
    ("Full ramp   — session_len=3, alpha=1.00", ITEMS[:3]),
    ("Normal      — session_len=5, alpha=1.00", ITEMS[:5]),
]

for label, prefix in cases:
    gru_idxs, gru_sc, _  = predict(prefix, use_cold_start=False)
    cs_idxs,  cs_sc,  a  = predict(prefix, use_cold_start=True)
    overlap = len(set(gru_idxs) & set(cs_idxs))
    mode    = "pure GRU" if a == 1.0 else ("pure popularity" if a == 0.0 else "blended")

    print(f"── {label} ──")
    print(f"  GRU-only  : {gru_idxs}  top-score={gru_sc[0]}")
    if pop_available:
        print(f"  Cold-start: {cs_idxs}  top-score={cs_sc[0]}")
        print(f"  alpha={a}  overlap={overlap}/{args.top_n}  [{mode}]")
    print()

# pure-popularity sanity (alpha=0)
if pop_available:
    print("── Popularity-only sanity (empty prefix → alpha=0.0) ──")
    pop_top = torch.topk(torch.log_softmax(pop_scores, dim=-1), args.top_n)
    pop_idxs = [int(i) + 1 for i in pop_top.indices.tolist()]
    print(f"  Top-{args.top_n} popular items: {pop_idxs}\n")

print("Done.")
