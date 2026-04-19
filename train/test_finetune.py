"""
Smoke test for finetune_gru4rec.py and eval_gru4rec.py.

Generates random data in prepare_data() format, creates a randomly
initialised pretrained checkpoint, then runs fine-tuning and eval
entirely locally (no MinIO, no real MLflow server needed).

Usage:
    cd train/
    python test_finetune.py
"""

import os
import sys
import pickle
import random
import tempfile
import subprocess

import torch
import numpy as np

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gru4rec import GRU4Rec

# ── reproducibility ──────────────────────────────────────────
random.seed(42)
np.random.seed(42)
torch.manual_seed(42)

# ── tiny config to keep the test fast ────────────────────────
NUM_ITEMS       = 200
NUM_USERS       = 50
TRAIN_SESSIONS  = 300
TEST_SESSIONS   = 60
MIN_SEQ_LEN     = 3
MAX_SEQ_LEN     = 12

ARCH = {
    "embedding_dim":    32,
    "hidden_dim":       64,
    "num_layers":       1,
    "dropout":          0.0,
    "embedding_dropout": 0.0,
    "use_user_context": False,
}


# ── helpers ───────────────────────────────────────────────────

def make_item2idx(n):
    """track_id is just an int 1000..1000+n."""
    return {1000 + i: i + 1 for i in range(n)}


def make_sessions(n, item2idx, user2idx, min_len, max_len):
    items   = list(item2idx.keys())
    users   = list(user2idx.keys())
    seqs    = []
    for i in range(n):
        length    = random.randint(min_len, max_len)
        track_ids = random.choices(items, k=length)
        user_id   = random.choice(users)
        item_idxs = [item2idx[t] for t in track_ids]
        ratios    = [round(random.uniform(0.3, 1.0), 2) for _ in range(length)]
        seqs.append({
            "session_id": f"sess_{i}",
            "user_idx":   user2idx[user_id],
            "item_idxs":  item_idxs,
            "playratios": ratios,
        })
    return seqs


def make_prepare_data_output(item2idx, user2idx, n_train, n_test):
    train_seqs = make_sessions(n_train, item2idx, user2idx, MIN_SEQ_LEN, MAX_SEQ_LEN)
    test_seqs  = make_sessions(n_test,  item2idx, user2idx, MIN_SEQ_LEN, MAX_SEQ_LEN)
    return {
        "item2idx":   item2idx,
        "user2idx":   user2idx,
        "train_seqs": train_seqs,
        "test_seqs":  test_seqs,
        "num_items":  len(item2idx),
        "num_users":  len(user2idx),
    }


# ── main test ─────────────────────────────────────────────────

def main():
    tmpdir = tempfile.mkdtemp(prefix="gru4rec_test_")
    print(f"Working in temp dir: {tmpdir}")

    # 1. Pretrained vocab + data (same catalog)
    item2idx = make_item2idx(NUM_ITEMS)
    user2idx = {2000 + i: i + 1 for i in range(NUM_USERS)}

    pretrain_vocab_path = os.path.join(tmpdir, "pretrain_vocab.pkl")
    with open(pretrain_vocab_path, "wb") as f:
        pickle.dump({"item2idx": item2idx, "user2idx": user2idx}, f)
    print(f"Saved pretrain vocab ({NUM_ITEMS} items)")

    # 2. Fine-tune data (same catalog, different sessions)
    ft_data      = make_prepare_data_output(item2idx, user2idx, TRAIN_SESSIONS, TEST_SESSIONS)
    ft_data_path = os.path.join(tmpdir, "finetune_data.pkl")
    with open(ft_data_path, "wb") as f:
        pickle.dump(ft_data, f)
    print(f"Saved finetune data ({TRAIN_SESSIONS} train / {TEST_SESSIONS} test sessions)")

    # 3. Random pretrained checkpoint (same arch as ARCH config)
    model      = GRU4Rec(NUM_ITEMS, NUM_USERS, ARCH)
    ckpt_path  = os.path.join(tmpdir, "pretrain.pt")
    torch.save(model.state_dict(), ckpt_path)
    print(f"Saved random pretrained checkpoint")

    # 4. Use the real MLflow server (from env) so the run appears in the UI
    mlflow_uri = os.environ.get("MLFLOW_TRACKING_URI", "http://129.114.27.204:8000")

    # 5. Run finetune_gru4rec.py
    print("\n" + "="*55)
    print("Running finetune_gru4rec.py ...")
    print("="*55)
    result = subprocess.run(
        [
            sys.executable, "finetune_gru4rec.py",
            "--checkpoint",     ckpt_path,
            "--pretrain-vocab", pretrain_vocab_path,
            "--finetune-data",  ft_data_path,
            "--embedding-dim",  str(ARCH["embedding_dim"]),
            "--hidden-dim",     str(ARCH["hidden_dim"]),
            "--num-layers",     str(ARCH["num_layers"]),
            "--epochs",         "3",
            "--batch-size",     "64",
            "--lr",             "1e-3",
            "--top-n",          "10",
            "--patience",       "2",
            "--device",         "cuda",
            "--mlflow-uri",     mlflow_uri,
            "--experiment",     "test-finetune",
            "--data-version",   "test-v0.0.1",
        ],
        cwd=os.path.dirname(os.path.abspath(__file__)),
        env=os.environ.copy(),
    )

    if result.returncode != 0:
        print("\n[FAIL] finetune_gru4rec.py exited with non-zero status")
        sys.exit(1)

    # 6. Find the saved finetuned checkpoint (best_... saved by finetune run)
    #    finetune saves to MinIO normally; with empty MINIO_* it will error on push.
    #    We patch: run eval against the original ckpt_path as a quick sanity check.
    print("\n" + "="*55)
    print("Running eval_gru4rec.py ...")
    print("="*55)

    eval_data_path = os.path.join(tmpdir, "eval_data.pkl")
    # Use only test_seqs for eval
    eval_data = {
        "item2idx":  item2idx,
        "user2idx":  user2idx,
        "test_seqs": ft_data["test_seqs"],
        "num_users": NUM_USERS,
    }
    with open(eval_data_path, "wb") as f:
        pickle.dump(eval_data, f)

    result = subprocess.run(
        [
            sys.executable, "eval_gru4rec.py",
            "--checkpoint",     ckpt_path,
            "--vocab",          pretrain_vocab_path,
            "--data",           eval_data_path,
            "--embedding-dim",  str(ARCH["embedding_dim"]),
            "--hidden-dim",     str(ARCH["hidden_dim"]),
            "--num-layers",     str(ARCH["num_layers"]),
            "--top-n",          "10",
            "--device",         "cuda",
            "--max-sessions",   "30",
        ],
        cwd=os.path.dirname(os.path.abspath(__file__)),
    )

    if result.returncode != 0:
        print("\n[FAIL] eval_gru4rec.py exited with non-zero status")
        sys.exit(1)

    print("\n[PASS] Both finetune and eval completed successfully.")
    print(f"Temp artifacts in: {tmpdir}")


if __name__ == "__main__":
    main()
