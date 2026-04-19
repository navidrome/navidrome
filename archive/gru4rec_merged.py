"""
GRU4Rec with In-Batch Sampled Softmax — 30Music Dataset
========================================================

Major changes vs previous version:
  - In-batch sampled softmax loss (replaces BPR-max + uniform negatives).
    This is the single biggest fix: the other positives in the batch act
    as popularity-weighted hard negatives, which is what GRU4Rec actually
    needs to learn good rankings.
  - Cross-entropy loss instead of BPR-max (better-behaved with in-batch negs)
  - Right-padding (was effectively left-padding; wasted GRU compute)
  - Embedding dropout (regularizes item reps, following sars_tutorial)
  - Playratio weighting off by default (was likely hurting)
  - Eval subsampling during training; full eval only at end
  - Fewer epochs by default; early stopping more aggressive
  - Lower default LR (stronger gradient signal from in-batch softmax)

Requirements
------------
    pip install torch mlflow pandas numpy tqdm optuna psutil
"""

import os
import json
import time
import logging
import random
import hashlib
import pickle
import platform
import subprocess
from collections import defaultdict

import numpy as np
import pandas as pd
import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.utils.data import Dataset, DataLoader
from tqdm import tqdm
import mlflow
import psutil

try:
    import optuna
    OPTUNA_AVAILABLE = True
except ImportError:
    OPTUNA_AVAILABLE = False

try:
    from minio_store import push_run_artifacts
    MINIO_AVAILABLE = True
except ImportError:
    MINIO_AVAILABLE = False

# ============================================================
# CONFIGURATION
# ============================================================

cfg = {
    # ---- Run mode ----
    "mode":                  "single",   # "single" | "tune"

    # ---- Dataset ----
    "dataset_root":          "data/",
    "sample_sessions":       None,

    # ---- Preprocessing ----
    "min_session_length":    3,
    "max_session_length":    100,
    "min_item_support":      5,
    "min_user_sessions":     3,
    "skip_ratio_threshold":  0.25,

    # ---- Model ----
    "embedding_dim":         64,
    "hidden_dim":            128,
    "num_layers":            1,
    "dropout":               0.2,
    "embedding_dropout":     0.25,       # NEW: dropout on input embeddings

    # ---- Training ----
    "epochs":                50,         # reduced: converges faster with sampled softmax
    "batch_size":            2048,       # reduced: in-batch negs already give ~B hard negs
    "lr":                    1e-3,       # reduced: stronger signal from sampled softmax
    "weight_decay":          1e-5,
    "use_playratio_weight":  False,      # OFF: was likely hurting
    "use_user_context":      False,
    "lr_step_size":          5,
    "lr_gamma":              0.5,
    "patience":              3,          # more aggressive early stop
    "label_smoothing":       0.0,

    # ---- Evaluation ----
    "top_n":                 20,
    "eval_every_n_epochs":   1,
    "eval_batch_size":       2048,
    "max_eval_sessions":     5000,       # NEW: subsample for speed during training
    "full_eval_at_end":      True,       # NEW: run full eval on final epoch

    # ---- Temporal split ----
    "test_fraction":         0.2,

    # ---- Hardware ----
    "device":                "cuda" if torch.cuda.is_available() else "cpu",
    "num_workers":           8,

    # ---- Cache ----
    "cache_dir":             ".cache_gru4rec",

    # ---- Optuna ----
    "n_trials":              20,
    "study_name":            "gru4rec_tuning",

    # ---- MLflow ----
    "mlflow_tracking_uri":   "http://129.114.25.168:8000",
    "mlflow_experiment":     "30music-session-recommendation",

    # ---- Dataset version (from data team Swift bucket) ----
    "dataset_version":       "v20260418-001",
}

ENTITIES_DIR  = os.path.join(cfg["dataset_root"], "entities")
RELATIONS_DIR = os.path.join(cfg["dataset_root"], "relations")

logging.basicConfig(level=logging.INFO, format="%(asctime)s | %(message)s", datefmt="%H:%M:%S")
log = logging.getLogger(__name__)


# ============================================================
# CACHE HELPERS
# ============================================================

def _cache_key(cfg: dict) -> str:
    relevant = {k: cfg[k] for k in (
        "sample_sessions", "min_session_length", "max_session_length",
        "min_item_support", "min_user_sessions", "skip_ratio_threshold",
        "test_fraction",
    )}
    raw = json.dumps(relevant, sort_keys=True, default=str).encode()
    return hashlib.md5(raw).hexdigest()[:10]


def _cache_path(stage: str, key: str, cache_dir: str) -> str:
    os.makedirs(cache_dir, exist_ok=True)
    return os.path.join(cache_dir, f"{stage}_{key}.pkl")


def _load_cache(stage: str, key: str, cache_dir: str = ".cache_gru4rec"):
    path = _cache_path(stage, key, cache_dir)
    if os.path.exists(path):
        log.info(f"[cache] Loading {stage} from {path}")
        with open(path, "rb") as f:
            return pickle.load(f)
    return None


def _save_cache(stage: str, key: str, data, cache_dir: str = ".cache_gru4rec"):
    path = _cache_path(stage, key, cache_dir)
    with open(path, "wb") as f:
        pickle.dump(data, f)
    log.info(f"[cache] Saved {stage} -> {path}")


# ============================================================
# ENVIRONMENT & COST TRACKING
# ============================================================

def collect_environment_info(device_str: str) -> dict:
    env = {
        "hostname":            platform.node(),
        "os":                  f"{platform.system()} {platform.release()}",
        "python_version":      platform.python_version(),
        "cpu_model":           platform.processor() or "unknown",
        "cpu_count_logical":   psutil.cpu_count(logical=True),
        "cpu_count_physical":  psutil.cpu_count(logical=False),
        "ram_total_gb":        round(psutil.virtual_memory().total / (1024 ** 3), 2),
        "pytorch_version":     torch.__version__,
        "cuda_available":      torch.cuda.is_available(),
    }

    if torch.cuda.is_available():
        env["cuda_version"]           = torch.version.cuda or "N/A"
        env["cudnn_version"]          = str(torch.backends.cudnn.version()) if torch.backends.cudnn.is_available() else "N/A"
        env["gpu_count"]              = torch.cuda.device_count()
        env["gpu_name"]               = torch.cuda.get_device_name(0)
        props                         = torch.cuda.get_device_properties(0)
        env["gpu_memory_total_gb"]    = round(props.total_memory / (1024 ** 3), 2)
        env["gpu_compute_capability"] = f"{props.major}.{props.minor}"
        try:
            r = subprocess.run(
                ["nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader"],
                capture_output=True, text=True, timeout=5,
            )
            env["nvidia_driver_version"] = r.stdout.strip().split("\n")[0]
        except Exception:
            env["nvidia_driver_version"] = "N/A"
    else:
        env["gpu_count"] = 0
        env["gpu_name"]  = "N/A (CPU only)"

    try:
        r = subprocess.run(
            ["git", "rev-parse", "--short", "HEAD"],
            capture_output=True, text=True, timeout=5,
            cwd=os.path.dirname(os.path.abspath(__file__)),
        )
        env["git_sha"] = r.stdout.strip() if r.returncode == 0 else "N/A"
    except Exception:
        env["git_sha"] = "N/A"

    return env


def log_environment_to_mlflow(env_info: dict):
    mlflow.set_tags({
        "env.hostname":        env_info.get("hostname", ""),
        "env.os":              env_info.get("os", ""),
        "env.gpu_name":        env_info.get("gpu_name", ""),
        "env.gpu_count":       str(env_info.get("gpu_count", 0)),
        "env.pytorch_version": env_info.get("pytorch_version", ""),
        "env.cuda_version":    env_info.get("cuda_version", "N/A"),
        "env.git_sha":         env_info.get("git_sha", "N/A"),
    })
    mlflow.log_params({
        "env_gpu_name":      env_info.get("gpu_name", "N/A"),
        "env_gpu_memory_gb": env_info.get("gpu_memory_total_gb", 0),
        "env_gpu_count":     env_info.get("gpu_count", 0),
        "env_cpu_count":     env_info.get("cpu_count_physical", 0),
        "env_ram_gb":        env_info.get("ram_total_gb", 0),
        "env_cuda_version":  env_info.get("cuda_version", "N/A"),
    })


def get_gpu_memory_stats() -> dict:
    if not torch.cuda.is_available():
        return {"gpu_mem_allocated_mb": 0, "gpu_mem_reserved_mb": 0, "gpu_mem_peak_mb": 0}
    return {
        "gpu_mem_allocated_mb": round(torch.cuda.memory_allocated() / (1024 ** 2), 1),
        "gpu_mem_reserved_mb":  round(torch.cuda.memory_reserved()  / (1024 ** 2), 1),
        "gpu_mem_peak_mb":      round(torch.cuda.max_memory_allocated() / (1024 ** 2), 1),
    }


# ============================================================
# PARSING
# ============================================================

def find_idomaar_file(directory: str, pattern: str) -> str:
    if not os.path.exists(directory):
        raise FileNotFoundError(f"Directory not found: {directory}")
    for fname in os.listdir(directory):
        if pattern.lower() in fname.lower() and fname.endswith(".idomaar"):
            return os.path.join(directory, fname)
    for fname in os.listdir(directory):
        if pattern.lower() in fname.lower():
            return os.path.join(directory, fname)
    raise FileNotFoundError(f"No file matching '{pattern}' in {directory}.")


def parse_sessions(filepath: str, cfg: dict) -> list:
    max_rows    = cfg["sample_sessions"]
    sessions    = []
    parse_errors = 0

    with open(filepath, "r", encoding="utf-8") as f:
        for i, line in enumerate(f):
            if max_rows and i >= max_rows:
                break
            line = line.strip()
            if not line:
                continue
            parts = line.split("\t")
            if len(parts) < 4:
                parse_errors += 1
                continue

            session_id = parts[1]
            try:
                session_ts = int(parts[2])
            except (ValueError, TypeError):
                session_ts = 0

            raw_props  = parts[3] if len(parts) > 3 else ""
            raw_linked = parts[4] if len(parts) > 4 else ""
            linked     = None

            if raw_linked:
                try:
                    linked = json.loads(raw_linked)
                except json.JSONDecodeError:
                    pass

            if linked is None and raw_props:
                brace_depth, split_pos = 0, -1
                for ci, ch in enumerate(raw_props):
                    if ch == "{":
                        brace_depth += 1
                    elif ch == "}":
                        brace_depth -= 1
                        if brace_depth == 0 and ci < len(raw_props) - 1:
                            split_pos = ci + 1
                            break
                if split_pos > 0:
                    second = raw_props[split_pos:].strip()
                    if second:
                        try:
                            linked = json.loads(second)
                        except json.JSONDecodeError:
                            pass

            if linked is None:
                parse_errors += 1
                continue

            subjects = linked.get("subjects", [])
            user_id  = subjects[0].get("id") if subjects else None

            track_sequence = []
            for obj in linked.get("objects", []):
                if obj.get("type") == "track":
                    track_sequence.append({
                        "track_id":  obj["id"],
                        "playstart": obj.get("playstart", 0),
                        "playtime":  obj.get("playtime", 0),
                        "playratio": obj.get("playratio"),
                        "action":    obj.get("action", "play"),
                    })

            track_sequence.sort(key=lambda x: x.get("playstart", 0))

            if user_id is not None and track_sequence:
                sessions.append({
                    "session_id": session_id,
                    "user_id":    user_id,
                    "timestamp":  session_ts,
                    "num_tracks": len(track_sequence),
                    "tracks":     track_sequence,
                })

    log.info(f"Parsed {len(sessions)} sessions ({parse_errors} parse errors)")
    return sessions


# ============================================================
# PREPROCESSING
# ============================================================

def sessions_to_dataframe(sessions: list, cfg: dict):
    session_rows, interaction_rows = [], []

    for s in sessions:
        session_rows.append({
            "session_id": s["session_id"],
            "user_id":    s["user_id"],
            "timestamp":  s["timestamp"],
            "num_tracks": s["num_tracks"],
        })
        for pos, t in enumerate(s["tracks"]):
            pr      = t.get("playratio")
            skipped = pr is not None and pr <= cfg["skip_ratio_threshold"]
            interaction_rows.append({
                "session_id": s["session_id"],
                "user_id":    s["user_id"],
                "position":   pos,
                "track_id":   t["track_id"],
                "playtime":   t.get("playtime", 0),
                "playratio":  pr,
                "skipped":    skipped,
            })

    session_df     = pd.DataFrame(session_rows)
    interaction_df = pd.DataFrame(interaction_rows)

    session_df["timestamp"]    = pd.to_numeric(session_df["timestamp"],    errors="coerce").astype("Int64")
    session_df["user_id"]      = pd.to_numeric(session_df["user_id"],      errors="coerce").astype("Int64")
    interaction_df["track_id"] = pd.to_numeric(interaction_df["track_id"], errors="coerce").astype("Int64")
    interaction_df["user_id"]  = pd.to_numeric(interaction_df["user_id"],  errors="coerce").astype("Int64")

    return session_df, interaction_df


def filter_data(session_df, interaction_df, cfg: dict):
    log.info(f"Before filtering: {len(session_df)} sessions, {len(interaction_df)} interactions")

    engaged_df = interaction_df[~interaction_df["skipped"]].copy()

    lengths    = engaged_df.groupby("session_id").size().reset_index(name="engaged_length")
    session_df = session_df.merge(lengths, on="session_id", how="left")
    session_df["engaged_length"] = session_df["engaged_length"].fillna(0).astype(int)

    valid_sessions = session_df[
        (session_df["engaged_length"] >= cfg["min_session_length"]) &
        (session_df["engaged_length"] <= cfg["max_session_length"])
    ]["session_id"]
    engaged_df = engaged_df[engaged_df["session_id"].isin(valid_sessions)]

    for _ in range(5):
        prev = len(engaged_df)
        item_counts = engaged_df["track_id"].value_counts()
        engaged_df  = engaged_df[engaged_df["track_id"].isin(item_counts[item_counts >= cfg["min_item_support"]].index)]
        sess_lens   = engaged_df.groupby("session_id").size()
        engaged_df  = engaged_df[engaged_df["session_id"].isin(sess_lens[sess_lens >= cfg["min_session_length"]].index)]
        user_sess   = engaged_df.groupby("user_id")["session_id"].nunique()
        engaged_df  = engaged_df[engaged_df["user_id"].isin(user_sess[user_sess >= cfg["min_user_sessions"]].index)]
        if len(engaged_df) == prev:
            break

    session_df = session_df[session_df["session_id"].isin(engaged_df["session_id"].unique())]
    log.info(
        f"After filtering: {session_df['session_id'].nunique()} sessions, "
        f"{engaged_df['track_id'].nunique()} unique tracks, "
        f"{engaged_df['user_id'].nunique()} users, "
        f"{len(engaged_df)} interactions"
    )
    return session_df, engaged_df


def build_vocabs(interaction_df):
    items    = sorted(interaction_df["track_id"].unique())
    users    = sorted(interaction_df["user_id"].unique())
    item2idx = {item: idx + 1 for idx, item in enumerate(items)}
    user2idx = {user: idx + 1 for idx, user in enumerate(users)}
    log.info(f"Vocab: {len(item2idx)} items, {len(user2idx)} users")
    return item2idx, user2idx


def build_sequences(interaction_df, item2idx: dict, user2idx: dict) -> list:
    sequences = []
    for session_id, group in interaction_df.sort_values("position").groupby("session_id"):
        user_id   = group["user_id"].iloc[0]
        item_ids  = group["track_id"].tolist()
        ratios    = group["playratio"].tolist()
        item_idxs = [item2idx[i] for i in item_ids if i in item2idx]

        clean_ratios = []
        for r in ratios:
            try:
                v = float(r)
                clean_ratios.append(np.clip(v, 0.0, 1.0) if not np.isnan(v) else 1.0)
            except (TypeError, ValueError):
                clean_ratios.append(1.0)

        if len(item_idxs) >= 2:
            sequences.append({
                "session_id": session_id,
                "user_idx":   user2idx.get(user_id, 0),
                "item_idxs":  item_idxs,
                "playratios": clean_ratios,
            })
    return sequences


def temporal_split(session_df, sequences: list, cfg: dict):
    session_df = session_df.sort_values("timestamp")
    split_idx  = int(len(session_df) * (1 - cfg["test_fraction"]))
    train_ids  = set(session_df.iloc[:split_idx]["session_id"])
    test_ids   = set(session_df.iloc[split_idx:]["session_id"])

    train_seqs = [s for s in sequences if s["session_id"] in train_ids]
    test_seqs  = [s for s in sequences if s["session_id"] in test_ids]
    log.info(f"Train: {len(train_seqs)} sessions | Test: {len(test_seqs)} sessions")
    return train_seqs, test_seqs


# ============================================================
# DATASET
# ============================================================

class SessionDataset(Dataset):
    """
    One sample = one (prefix -> next_item) step within a session.
    No negative sampling here anymore — in-batch negatives are used in the loss.
    """
    def __init__(self, sequences: list, use_playratio_weight: bool):
        self.samples              = []
        self.use_playratio_weight = use_playratio_weight
        for seq in sequences:
            items  = seq["item_idxs"]
            ratios = seq["playratios"]
            user   = seq["user_idx"]
            for t in range(1, len(items)):
                weight = float(ratios[t]) if use_playratio_weight else 1.0
                weight = max(weight, 0.1)
                self.samples.append((items[:t], user, items[t], weight))

    def __len__(self):
        return len(self.samples)

    def __getitem__(self, idx):
        prefix, user, next_item, weight = self.samples[idx]
        return {
            "prefix":   torch.tensor(prefix,    dtype=torch.long),
            "user":     torch.tensor(user,       dtype=torch.long),
            "positive": torch.tensor(next_item,  dtype=torch.long),
            "weight":   torch.tensor(weight,     dtype=torch.float),
        }


def collate_fn(batch):
    """
    Right-pads sequences. pack_padded_sequence handles variable lengths.
    No negative sampling — handled in-batch by the loss.
    """
    max_len  = max(b["prefix"].size(0) for b in batch)
    B        = len(batch)

    prefixes = torch.zeros(B, max_len, dtype=torch.long)
    for i, b in enumerate(batch):
        L = b["prefix"].size(0)
        prefixes[i, :L] = b["prefix"]   # right-pad

    return {
        "prefix":    prefixes,
        "user":      torch.stack([b["user"]     for b in batch]),
        "positive":  torch.stack([b["positive"] for b in batch]),
        "weight":    torch.stack([b["weight"]   for b in batch]),
    }


# ============================================================
# MODEL
# ============================================================

class GRU4Rec(nn.Module):
    def __init__(self, num_items: int, num_users: int, cfg: dict):
        super().__init__()
        self.use_user_context = cfg["use_user_context"]
        embed_dim             = cfg["embedding_dim"]
        hidden_dim            = cfg["hidden_dim"]

        self.item_emb     = nn.Embedding(num_items + 1, embed_dim, padding_idx=0)
        nn.init.xavier_uniform_(self.item_emb.weight[1:])
        self.emb_dropout  = nn.Dropout(cfg.get("embedding_dropout", 0.0))

        gru_input_dim = embed_dim
        if self.use_user_context:
            self.user_emb = nn.Embedding(num_users + 1, embed_dim, padding_idx=0)
            nn.init.xavier_uniform_(self.user_emb.weight[1:])
            gru_input_dim += embed_dim

        self.gru = nn.GRU(
            input_size=gru_input_dim,
            hidden_size=hidden_dim,
            num_layers=cfg["num_layers"],
            batch_first=True,
            dropout=cfg["dropout"] if cfg["num_layers"] > 1 else 0.0,
        )
        self.dropout     = nn.Dropout(cfg["dropout"])
        self.output_proj = nn.Linear(hidden_dim, embed_dim, bias=False)
        self.layer_norm  = nn.LayerNorm(embed_dim)

    def encode_session(self, prefix_items: torch.Tensor, user_idxs: torch.Tensor) -> torch.Tensor:
        x = self.item_emb(prefix_items)
        x = self.emb_dropout(x)

        if self.use_user_context:
            u = self.user_emb(user_idxs).unsqueeze(1).expand(-1, x.size(1), -1)
            x = torch.cat([x, u], dim=-1)

        lengths = (prefix_items != 0).sum(dim=1).cpu().clamp(min=1)

        packed       = nn.utils.rnn.pack_padded_sequence(x, lengths, batch_first=True, enforce_sorted=False)
        _, h_n       = self.gru(packed)
        h_last       = self.dropout(h_n[-1])
        session_repr = self.layer_norm(self.output_proj(h_last))
        return session_repr

    def forward_inbatch(self, prefix_items, user_idxs, positive_items):
        """
        In-batch sampled softmax forward.
        Returns logits (B, B) where diagonal is the positive score and
        off-diagonal entries are scores against other positives in the batch
        (which serve as popularity-weighted negatives).
        """
        session_repr = self.encode_session(prefix_items, user_idxs)   # (B, D)
        pos_emb      = self.item_emb(positive_items)                  # (B, D)
        logits       = session_repr @ pos_emb.T                       # (B, B)
        return logits

    @torch.no_grad()
    def predict_top_n(
        self,
        prefix_items: torch.Tensor,
        user_idxs: torch.Tensor,
        all_item_emb: torch.Tensor,
        top_n: int,
        exclude_sets: list,
    ) -> list:
        session_repr = self.encode_session(prefix_items, user_idxs)
        scores       = session_repr @ all_item_emb.T

        for b, excl in enumerate(exclude_sets):
            for item_idx in excl:
                scores[b, item_idx - 1] = float("-inf")

        top_indices = torch.topk(scores, top_n, dim=-1).indices.cpu().numpy()
        return [[int(i) + 1 for i in row] for row in top_indices]


# ============================================================
# LOSS — in-batch sampled softmax with optional label smoothing
# ============================================================

def inbatch_softmax_loss(logits: torch.Tensor, weights=None, label_smoothing: float = 0.0) -> torch.Tensor:
    """
    Cross-entropy with the diagonal as the correct class.
    logits: (B, B) — row i, col j = score of session_i against positive_j
    """
    B      = logits.size(0)
    labels = torch.arange(B, device=logits.device)

    # Mask out duplicate positives in the batch (if the same item appears as
    # positive for multiple samples, we shouldn't treat it as a negative).
    # This is cheap and avoids a known bug in naive in-batch softmax.
    # We detect duplicates by comparing against labels — if logits[i, j] refers
    # to the same item as logits[i, i], mask it. We approximate this by passing
    # a mask from outside; here we just handle the simple case.
    loss = F.cross_entropy(logits, labels, reduction="none", label_smoothing=label_smoothing)
    if weights is not None:
        loss = loss * weights
    return loss.mean()


def inbatch_softmax_loss_masked(
    logits: torch.Tensor,
    positives: torch.Tensor,
    weights=None,
    label_smoothing: float = 0.0,
) -> torch.Tensor:
    """
    Same as above but masks out off-diagonal entries where the 'negative'
    item id equals the true positive id for that row. Prevents accidentally
    pushing the correct item down.
    """
    B      = logits.size(0)
    labels = torch.arange(B, device=logits.device)

    # Build a mask: True where column j's item == row i's positive item and i != j
    same   = positives.unsqueeze(0) == positives.unsqueeze(1)   # (B, B)
    eye    = torch.eye(B, dtype=torch.bool, device=logits.device)
    mask   = same & ~eye
    logits = logits.masked_fill(mask, float("-inf"))

    loss = F.cross_entropy(logits, labels, reduction="none", label_smoothing=label_smoothing)
    if weights is not None:
        loss = loss * weights
    return loss.mean()


# ============================================================
# TRAINING
# ============================================================

def train_epoch(model, loader, optimizer, scaler, device, run_cfg: dict):
    model.train()
    total_loss    = 0.0
    num_batches   = 0
    total_samples = 0
    use_amp       = (device.type == "cuda")
    label_smooth  = run_cfg.get("label_smoothing", 0.0)

    for step, batch in enumerate(loader):
        prefix    = batch["prefix"].to(device, non_blocking=True)
        user      = batch["user"].to(device, non_blocking=True)
        positive  = batch["positive"].to(device, non_blocking=True)
        weight    = batch["weight"].to(device, non_blocking=True) if run_cfg["use_playratio_weight"] else None

        optimizer.zero_grad()

        with torch.cuda.amp.autocast(enabled=use_amp):
            logits = model.forward_inbatch(prefix, user, positive)
            loss   = inbatch_softmax_loss_masked(logits, positive, weight, label_smoothing=label_smooth)

        if use_amp:
            scaler.scale(loss).backward()
            scaler.unscale_(optimizer)
            nn.utils.clip_grad_norm_(model.parameters(), max_norm=5.0)
            scaler.step(optimizer)
            scaler.update()
        else:
            loss.backward()
            nn.utils.clip_grad_norm_(model.parameters(), max_norm=5.0)
            optimizer.step()

        total_loss    += loss.item()
        num_batches   += 1
        total_samples += prefix.size(0)

    return total_loss / max(num_batches, 1), total_samples


# ============================================================
# EVALUATION
# ============================================================

@torch.no_grad()
def evaluate(model, test_sequences: list, run_cfg: dict, device: torch.device,
             max_sessions: int = None) -> dict:
    """
    Batched evaluation. If max_sessions is set, subsamples test sequences for speed.
    """
    model.eval()
    top_n      = run_cfg["top_n"]
    eval_bs    = run_cfg.get("eval_batch_size", 2048)
    raw_model  = model.module if isinstance(model, nn.DataParallel) else model

    if max_sessions is not None and len(test_sequences) > max_sessions:
        rng       = random.Random(42)
        test_sequences = rng.sample(test_sequences, max_sessions)

    all_item_emb = raw_model.item_emb.weight[1:].to(device)

    eval_steps = []
    for seq in test_sequences:
        items = seq["item_idxs"]
        user  = seq["user_idx"]
        for split in range(1, len(items)):
            eval_steps.append((
                items[:split],
                user,
                items[split],
                set(items[split:]),
            ))

    strict_hits, strict_mrr   = 0, 0.0
    session_hits, session_mrr = 0, 0.0
    session_prec, session_rec = 0.0, 0.0
    all_recommended           = set()
    predict_latencies_ms      = []

    t_eval_start = time.time()

    for start in tqdm(range(0, len(eval_steps), eval_bs), desc="Evaluating", leave=False):
        chunk   = eval_steps[start: start + eval_bs]
        max_len = max(len(c[0]) for c in chunk)
        B       = len(chunk)

        prefix_t = torch.zeros(B, max_len, dtype=torch.long, device=device)
        user_t   = torch.zeros(B,          dtype=torch.long, device=device)
        for i, (prefix, user, _, _) in enumerate(chunk):
            L = len(prefix)
            prefix_t[i, :L] = torch.tensor(prefix, dtype=torch.long)   # right-pad
            user_t[i]       = user

        exclude_sets = [set(c[0]) for c in chunk]

        if device.type == "cuda":
            torch.cuda.synchronize(device)
        t_pred = time.perf_counter()

        predicted_batch = raw_model.predict_top_n(
            prefix_t, user_t, all_item_emb, top_n, exclude_sets
        )

        if device.type == "cuda":
            torch.cuda.synchronize(device)
        batch_ms = (time.perf_counter() - t_pred) * 1000

        per_pred_ms = batch_ms / B
        predict_latencies_ms.extend([per_pred_ms] * B)

        for (_, _, next_item, remaining), predicted in zip(chunk, predicted_batch):
            all_recommended.update(predicted)

            if next_item in predicted:
                strict_hits += 1
                strict_mrr  += 1.0 / (predicted.index(next_item) + 1)

            hit_positions = [i for i, p in enumerate(predicted) if p in remaining]
            if hit_positions:
                session_hits += 1
                session_mrr  += 1.0 / (hit_positions[0] + 1)

            n_rel = len(hit_positions)
            session_prec += n_rel / top_n
            session_rec  += n_rel / len(remaining) if remaining else 0.0

    elapsed = time.time() - t_eval_start
    n       = max(len(eval_steps), 1)
    lat     = np.array(predict_latencies_ms) if predict_latencies_ms else np.array([0.0])

    results = {
        "strict_HR":         strict_hits / n,
        "strict_MRR":        strict_mrr  / n,
        "session_HR":        session_hits / n,
        "session_MRR":       session_mrr  / n,
        "session_precision": session_prec / n,
        "session_recall":    session_rec  / n,
        "coverage":          len(all_recommended),
        "total_predictions": n,
        "latency_mean_ms":   float(np.mean(lat)),
        "latency_p50_ms":    float(np.percentile(lat, 50)),
        "latency_p95_ms":    float(np.percentile(lat, 95)),
        "latency_p99_ms":    float(np.percentile(lat, 99)),
        "latency_max_ms":    float(np.max(lat)),
        "throughput_qps":    float(n / elapsed) if elapsed > 0 else 0.0,
    }

    log.info(
        f"  Serving speed:  mean={results['latency_mean_ms']:.3f}ms  "
        f"p50={results['latency_p50_ms']:.3f}ms  "
        f"p95={results['latency_p95_ms']:.3f}ms  "
        f"p99={results['latency_p99_ms']:.3f}ms  "
        f"max={results['latency_max_ms']:.3f}ms  "
        f"QPS={results['throughput_qps']:.1f}"
    )

    return results


# ============================================================
# DATA PREPARATION
# ============================================================

def prepare_data(cfg: dict) -> dict:
    """
    Loads pre-built training data from Swift bucket.
    Replaces idomaar parsing — data already filtered, vocab built, sequences ready.

    Swift URL: https://chi.uc.chameleoncloud.org:7480/swift/v1/AUTH_7c0a7a1952e44c94aa75cae1ff5dc9b4/navidrome-bucket-proj05/datasets/{version}/
    Files: train_sequences.pkl, test_sequences.pkl, item2idx.json, user2idx.json
    """
    import pickle, json, requests, os as _os

    version   = cfg.get("dataset_version", "v20260418-001")
    base_url  = f"https://chi.uc.chameleoncloud.org:7480/swift/v1/AUTH_7c0a7a1952e44c94aa75cae1ff5dc9b4/navidrome-bucket-proj05/datasets/{version}"
    cache_dir = cfg.get("cache_dir", ".cache_gru4rec")
    _os.makedirs(cache_dir, exist_ok=True)

    def download(fname):
        local = _os.path.join(cache_dir, f"{version}_{fname}")
        if _os.path.exists(local):
            log.info(f"[cache] Using cached {fname}")
            return local
        log.info(f"Downloading {fname} from Swift...")
        r = requests.get(f"{base_url}/{fname}", timeout=300, stream=True)
        r.raise_for_status()
        with open(local, "wb") as fh:
            for chunk in r.iter_content(chunk_size=8192):
                fh.write(chunk)
        log.info(f"Downloaded {fname} ({_os.path.getsize(local)/1e6:.1f} MB)")
        return local

    train_path    = download("train_sequences.pkl")
    test_path     = download("test_sequences.pkl")
    item2idx_path = download("item2idx.json")
    user2idx_path = download("user2idx.json")

    log.info("Loading train_sequences.pkl...")
    with open(train_path, "rb") as fh:
        train_seqs = pickle.load(fh)

    log.info("Loading test_sequences.pkl...")
    with open(test_path, "rb") as fh:
        test_seqs = pickle.load(fh)

    log.info("Loading item2idx.json...")
    with open(item2idx_path) as fh:
        item2idx_raw = json.load(fh)
    item2idx = {int(k): int(v) for k, v in item2idx_raw.items()}

    log.info("Loading user2idx.json...")
    with open(user2idx_path) as fh:
        user2idx_raw = json.load(fh)
    user2idx = {int(k): int(v) for k, v in user2idx_raw.items()}

    log.info(
        f"Dataset {version} loaded: "
        f"{len(train_seqs):,} train seqs, "
        f"{len(test_seqs):,} test seqs, "
        f"{len(item2idx):,} items, "
        f"{len(user2idx):,} users"
    )

    train_seqs = train_seqs
    test_seqs  = test_seqs

    # expect this output from the data member - HASHIRRRRR
    return {
        "item2idx":   item2idx,
        "user2idx":   user2idx,
        "train_seqs": train_seqs,
        "test_seqs":  test_seqs,
        "num_items":  len(item2idx),
        "num_users":  len(user2idx),
    }


# ============================================================
# TRAINING RUN
# ============================================================

def run_training(run_cfg: dict, data: dict, env_info: dict, is_tuning: bool = False) -> dict:
    num_items  = data["num_items"]
    num_users  = data["num_users"]
    train_seqs = data["train_seqs"]
    test_seqs  = data["test_seqs"]

    device = torch.device(run_cfg["device"])

    train_dataset = SessionDataset(train_seqs, run_cfg["use_playratio_weight"])
    train_loader  = DataLoader(
        train_dataset,
        batch_size=run_cfg["batch_size"],
        shuffle=True,
        num_workers=run_cfg["num_workers"],
        collate_fn=collate_fn,
        pin_memory=(device.type == "cuda"),
        persistent_workers=(run_cfg["num_workers"] > 0),
        drop_last=True,   # important for in-batch softmax: avoid tiny final batch
    )

    model    = GRU4Rec(num_items, num_users, run_cfg).to(device)
    n_params = sum(p.numel() for p in model.parameters())
    log.info(f"Model parameters: {n_params:,}")

    optimizer = torch.optim.Adam(model.parameters(), lr=run_cfg["lr"], weight_decay=run_cfg["weight_decay"])
    scheduler = torch.optim.lr_scheduler.StepLR(optimizer, step_size=run_cfg["lr_step_size"], gamma=run_cfg["lr_gamma"])
    scaler    = torch.cuda.amp.GradScaler(enabled=(device.type == "cuda"))

    if device.type == "cuda":
        torch.cuda.reset_peak_memory_stats()

    run_name = (
        f"gru4rec_inbatch_e{run_cfg['embedding_dim']}_h{run_cfg['hidden_dim']}_l{run_cfg['num_layers']}"
        f"{'_ratio' if run_cfg['use_playratio_weight'] else ''}"
        f"{'_tune' if is_tuning else ''}"
    )

    best_session_hr   = 0.0
    best_state_dict   = None   # kept in memory; pushed to MinIO at end
    best_results      = {}
    epochs_no_improve = 0
    final_results     = {}
    patience          = run_cfg.get("patience", 3)
    eval_every        = run_cfg.get("eval_every_n_epochs", 2)
    max_eval          = run_cfg.get("max_eval_sessions", None)

    with mlflow.start_run(run_name=run_name, nested=is_tuning) as run:
        mlflow.log_params({k: str(v) for k, v in run_cfg.items()})
        mlflow.log_params({
            "num_items":        num_items,
            "num_users":        num_users,
            "train_sessions":   len(train_seqs),
            "test_sessions":    len(test_seqs),
            "train_samples":    len(train_dataset),
            "model_parameters": n_params,
            "loss_type":        "inbatch_sampled_softmax",
        })
        mlflow.set_tags({
            "model_type":   "GRU4Rec-InBatch",
            "dataset":      "30Music",
            "tuning_trial": str(is_tuning),
        })
        log_environment_to_mlflow(env_info)

        t_start     = time.time()
        epoch_times = []

        for epoch in range(1, run_cfg["epochs"] + 1):
            t0 = time.time()

            avg_loss, samples = train_epoch(model, train_loader, optimizer, scaler, device, run_cfg)
            scheduler.step()

            epoch_time = time.time() - t0
            epoch_times.append(epoch_time)
            throughput = samples / epoch_time if epoch_time > 0 else 0
            gpu_mem    = get_gpu_memory_stats()

            mlflow.log_metrics({
                "train_loss":                 avg_loss,
                "epoch_time_sec":             round(epoch_time, 2),
                "wall_time_sec":              round(time.time() - t_start, 2),
                "throughput_samples_per_sec": round(throughput, 1),
                "learning_rate":              optimizer.param_groups[0]["lr"],
                "gpu_mem_allocated_mb":       gpu_mem["gpu_mem_allocated_mb"],
                "gpu_mem_peak_mb":            gpu_mem["gpu_mem_peak_mb"],
            }, step=epoch)

            if not is_tuning:
                log.info(
                    f"Epoch {epoch:02d}/{run_cfg['epochs']} | "
                    f"loss={avg_loss:.4f} | {epoch_time:.1f}s | "
                    f"{throughput:.0f} samp/s | "
                    f"peak={gpu_mem['gpu_mem_peak_mb']:.0f}MB"
                )

            is_last = (epoch == run_cfg["epochs"])
            if epoch % eval_every == 0 or is_last:
                t_eval = time.time()
                # Use subsampled eval except on last epoch (if configured)
                eval_limit = None if (is_last and run_cfg.get("full_eval_at_end", True)) else max_eval
                results = evaluate(model, test_seqs, run_cfg, device, max_sessions=eval_limit)
                eval_time = time.time() - t_eval

                top_n = run_cfg["top_n"]
                mlflow.log_metrics({
                    f"strict_HR{top_n}":         results["strict_HR"],
                    f"strict_MRR{top_n}":        results["strict_MRR"],
                    f"session_HR{top_n}":        results["session_HR"],
                    f"session_MRR{top_n}":       results["session_MRR"],
                    f"session_precision{top_n}": results["session_precision"],
                    f"session_recall{top_n}":    results["session_recall"],
                    "coverage":                  results["coverage"],
                    "eval_time_sec":             round(eval_time, 2),
                    "latency_mean_ms":           results["latency_mean_ms"],
                    "latency_p50_ms":            results["latency_p50_ms"],
                    "latency_p95_ms":            results["latency_p95_ms"],
                    "latency_p99_ms":            results["latency_p99_ms"],
                    "latency_max_ms":            results["latency_max_ms"],
                    "throughput_qps":            results["throughput_qps"],
                }, step=epoch)

                if not is_tuning:
                    log.info(
                        f"  Strict  HR{top_n}={results['strict_HR']:.4f}  MRR={results['strict_MRR']:.4f}\n"
                        f"  Session HR{top_n}={results['session_HR']:.4f}  MRR={results['session_MRR']:.4f}  "
                        f"P={results['session_precision']:.4f}  R={results['session_recall']:.4f}\n"
                        f"  Eval: {eval_time:.1f}s  (subsampled={eval_limit is not None})"
                    )

                if results["session_HR"] > best_session_hr:
                    best_session_hr   = results["session_HR"]
                    best_results      = results
                    epochs_no_improve = 0
                    if not is_tuning:
                        state = model.module.state_dict() if isinstance(model, nn.DataParallel) else model.state_dict()
                        best_state_dict = {k: v.cpu().clone() for k, v in state.items()}
                        log.info(f"  -> New best session HR{top_n}: {best_session_hr:.4f}")

                        if MINIO_AVAILABLE:
                            from datetime import datetime, timezone
                            _meta = {
                                "run_type":      "pretrain",
                                "mlflow_run_id": run.info.run_id,
                                "timestamp":     datetime.now(timezone.utc).isoformat(),
                                "session_HR":    round(best_session_hr, 6),
                                "session_MRR":   round(results.get("session_MRR", 0), 6),
                                "strict_HR":     round(results.get("strict_HR", 0), 6),
                                "num_items":     data["num_items"],
                                "embedding_dim": run_cfg["embedding_dim"],
                                "hidden_dim":    run_cfg["hidden_dim"],
                                "num_layers":    run_cfg["num_layers"],
                                "epoch":         epoch,
                                "gpu_name":      env_info.get("gpu_name", ""),
                                "git_sha":       env_info.get("git_sha", ""),
                            }
                            _vocab = {
                                "item2idx": data["item2idx"],
                                "user2idx": data["user2idx"],
                            }
                            try:
                                _keys = push_run_artifacts(
                                    state_dict=best_state_dict,
                                    run_type="pretrain",
                                    run_id=run.info.run_id,
                                    metadata=_meta,
                                    vocab=_vocab,
                                )
                                mlflow.set_tags({
                                    "minio_model_key":    _keys["model_key"],
                                    "minio_vocab_key":    _keys.get("vocab_key", ""),
                                    "minio_metadata_key": _keys["metadata_key"],
                                })
                                log.info(f"  [minio] model    → {_keys['model_key']}")
                                log.info(f"  [minio] vocab    → {_keys.get('vocab_key', '')}")
                            except Exception as _e:
                                log.warning(f"  [minio] Upload skipped — {_e}")
                else:
                    epochs_no_improve += 1
                    if epochs_no_improve >= patience:
                        log.info(f"  Early stopping at epoch {epoch} (no improvement for {patience} eval rounds)")
                        break

                final_results = results

        total_time    = time.time() - t_start
        gpu_hours     = (total_time / 3600) * max(env_info.get("gpu_count", 1), 1) if env_info.get("cuda_available") else 0
        final_gpu_mem = get_gpu_memory_stats()

        mlflow.log_metrics({
            "total_train_seconds": round(total_time, 2),
            "total_train_minutes": round(total_time / 60, 2),
            "gpu_hours":           round(gpu_hours, 4),
            "avg_epoch_time_sec":  round(np.mean(epoch_times), 2),
            "peak_gpu_memory_mb":  final_gpu_mem["gpu_mem_peak_mb"],
            "best_session_HR":     best_session_hr,
        })

        if not is_tuning:
            log.info(
                f"\nDone. Total: {total_time:.1f}s ({total_time/60:.1f} min) | "
                f"GPU-hours: {gpu_hours:.4f} | "
                f"Peak VRAM: {final_gpu_mem['gpu_mem_peak_mb']:.0f}MB | "
                f"Best HR{run_cfg['top_n']}: {best_session_hr:.4f} | "
                f"MLflow: {run.info.run_id}"
            )

    final_results.update({
        "best_session_HR":     best_session_hr,
        "total_train_seconds": total_time,
        "gpu_hours":           gpu_hours,
        "peak_gpu_memory_mb":  final_gpu_mem["gpu_mem_peak_mb"],
    })
    return final_results


# ============================================================
# OPTUNA TUNING
# ============================================================

def create_optuna_trial_config(trial, base_cfg: dict) -> dict:
    trial_cfg = base_cfg.copy()
    trial_cfg["embedding_dim"]     = trial.suggest_categorical("embedding_dim",  [32, 64, 128])
    trial_cfg["hidden_dim"]        = trial.suggest_categorical("hidden_dim",     [64, 128, 256])
    trial_cfg["num_layers"]        = trial.suggest_int("num_layers", 1, 2)
    trial_cfg["dropout"]           = trial.suggest_float("dropout",  0.1, 0.5, step=0.1)
    trial_cfg["embedding_dropout"] = trial.suggest_float("embedding_dropout", 0.0, 0.4, step=0.1)
    trial_cfg["lr"]                = trial.suggest_float("lr",       1e-4, 3e-3, log=True)
    trial_cfg["weight_decay"]      = trial.suggest_float("weight_decay", 1e-6, 1e-3, log=True)
    trial_cfg["batch_size"]        = trial.suggest_categorical("batch_size",    [512, 1024, 2048, 4096])
    trial_cfg["label_smoothing"]   = trial.suggest_float("label_smoothing", 0.0, 0.1, step=0.05)
    trial_cfg["lr_step_size"]      = trial.suggest_int("lr_step_size", 3, 10)
    trial_cfg["lr_gamma"]          = trial.suggest_float("lr_gamma", 0.3, 0.8, step=0.1)
    trial_cfg["epochs"]            = 10
    trial_cfg["max_eval_sessions"] = 3000   # faster tuning
    return trial_cfg


def run_optuna_tuning(base_cfg: dict, data: dict, env_info: dict):
    if not OPTUNA_AVAILABLE:
        log.error("Optuna not installed: pip install optuna")
        return

    def objective(trial):
        trial_cfg = create_optuna_trial_config(trial, base_cfg)
        try:
            results = run_training(trial_cfg, data, env_info, is_tuning=True)
            return results.get("best_session_HR", 0.0)
        except Exception as e:
            log.error(f"Trial {trial.number} failed: {e}")
            return 0.0

    sampler = optuna.samplers.TPESampler(seed=42)
    study   = optuna.create_study(study_name=base_cfg["study_name"], direction="maximize", sampler=sampler)

    with mlflow.start_run(run_name=f"optuna_{base_cfg['study_name']}"):
        mlflow.log_params({"n_trials": base_cfg["n_trials"], "sampler": "TPE"})
        mlflow.set_tags({"run_type": "hyperparameter_tuning"})
        log_environment_to_mlflow(env_info)
        study.optimize(objective, n_trials=base_cfg["n_trials"], show_progress_bar=True)

        best = study.best_trial
        log.info(f"Best trial {best.number}: session_HR={best.value:.4f}")
        log.info(f"Best params: {json.dumps(best.params, indent=2, default=str)}")

        mlflow.log_metrics({"best_session_HR": best.value})
        mlflow.log_params({f"best_{k}": str(v) for k, v in best.params.items()})

        trials_df = study.trials_dataframe()
        trials_df.to_csv("optuna_trials.csv", index=False)
        mlflow.log_artifact("optuna_trials.csv")

    return study


# ============================================================
# MAIN
# ============================================================

def main():
    log.info("=" * 60)
    log.info(json.dumps(cfg, indent=2, default=str))
    log.info("=" * 60)

    env_info = collect_environment_info(cfg["device"])
    log.info(f"Device: {cfg['device']}")
    if env_info.get("cuda_available"):
        log.info(f"GPU: {env_info['gpu_name']} ({env_info.get('gpu_memory_total_gb', '?')} GB)")
    log.info(f"CPU: {env_info['cpu_count_physical']} cores | RAM: {env_info['ram_total_gb']} GB")

    mlflow.set_tracking_uri(cfg["mlflow_tracking_uri"])
    mlflow.set_experiment(cfg["mlflow_experiment"])

    data = prepare_data(cfg)

    if cfg["mode"] == "tune":
        run_optuna_tuning(base_cfg=cfg, data=data, env_info=env_info)
    else:
        results = run_training(run_cfg=cfg, data=data, env_info=env_info, is_tuning=False)
        log.info(f"Final best session HR{cfg['top_n']}: {results['best_session_HR']:.4f}")

    log.info(f"MLflow UI: {cfg['mlflow_tracking_uri']}")

    # reload Redis vocab with latest trained vocab
    try:
        import subprocess
        log.info("Reloading Redis vocab from MinIO...")
        subprocess.run(["python3", "data/pipeline/reload_vocab.py", "--latest"], check=True)
        log.info("Redis vocab reloaded successfully")
    except Exception as e:
        log.warning(f"Redis vocab reload failed (non-fatal): {e}")


if __name__ == "__main__":
    main()
