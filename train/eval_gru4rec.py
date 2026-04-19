"""
Evaluation script for GRU4Rec.

Downloads the model + vocab from MinIO, runs test sequences,
and reports session HR and supporting metrics.

Set MinIO env vars:
    MINIO_URL       e.g. http://minio:9000
    MINIO_USER      your-access-key
    MINIO_PASSWORD  your-secret-key
    MINIO_BUCKET    (optional) defaults to "gru4rec-models"

Expects data in prepare_data() output format (pickle):
    {
        "item2idx":  dict[track_id -> int],
        "test_seqs": list[{session_id, user_idx, item_idxs, playratios}],
        ...
    }

Usage:
    # Load from MinIO (standard):
    python eval_gru4rec.py \\
        --model-key  pretrain/2026-04-18/{run_id}/model.pt \\
        --vocab-key  pretrain/2026-04-18/{run_id}/vocab.pkl \\
        --data       data.pkl

    # Load from local files (quick dev check):
    python eval_gru4rec.py \\
        --checkpoint best_gru4rec.pt \\
        --vocab      pretrain_vocab.pkl \\
        --data       data.pkl
"""

import json
import logging
import os
import sys
import argparse
import pickle

import torch

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gru4rec import GRU4Rec, evaluate
from minio_store import get_client, download_model, download_vocab

logging.basicConfig(level=logging.INFO, format="%(asctime)s | %(message)s", datefmt="%H:%M:%S")
log = logging.getLogger(__name__)


DEFAULT_CFG = {
    "embedding_dim":     64,
    "hidden_dim":        128,
    "num_layers":        1,
    "dropout":           0.2,
    "embedding_dropout": 0.25,
    "use_user_context":  False,
    "top_n":             20,
    "eval_batch_size":   2048,
    "max_eval_sessions": None,
    "device":            "cuda" if torch.cuda.is_available() else "cpu",
}


# ============================================================
# SEQUENCE REMAPPING
# ============================================================

def remap_sequences(seqs: list, data_item2idx: dict, model_item2idx: dict) -> list:
    """Reindex sequences from data indices to model indices.

    No-op fast path if both vocabs are identical.
    """
    if data_item2idx == model_item2idx:
        return seqs

    idx2track = {v: k for k, v in data_item2idx.items()}
    remapped  = []
    skipped   = 0

    for seq in seqs:
        new_items = []
        for data_idx in seq["item_idxs"]:
            track_id = idx2track.get(data_idx)
            if track_id in model_item2idx:
                new_items.append(model_item2idx[track_id])

        if len(new_items) >= 2:
            remapped.append({
                "session_id": seq["session_id"],
                "user_idx":   seq["user_idx"],
                "item_idxs":  new_items,
                "playratios": seq["playratios"][: len(new_items)],
            })
        else:
            skipped += 1

    if skipped:
        log.warning(f"Dropped {skipped} sequences with < 2 mappable items")

    return remapped


# ============================================================
# EVALUATION RUN
# ============================================================

def run_eval(
    model_key: str,      # MinIO key OR None
    vocab_key: str,      # MinIO key OR None
    checkpoint: str,     # local .pt  OR None
    vocab_path: str,     # local pkl  OR None
    data_path: str,
    eval_cfg: dict,
) -> dict:

    device = torch.device(eval_cfg["device"])

    # Load vocab
    if vocab_key:
        s3    = get_client()
        vocab = download_vocab(s3, vocab_key)
    else:
        with open(vocab_path, "rb") as f:
            vocab = pickle.load(f)

    model_item2idx = vocab["item2idx"]
    num_items      = len(model_item2idx)
    log.info(f"Model vocab: {num_items} items")

    # Load test data
    with open(data_path, "rb") as f:
        data = pickle.load(f)

    test_seqs = remap_sequences(
        data["test_seqs"],
        data.get("item2idx", model_item2idx),
        model_item2idx,
    )
    log.info(f"Evaluating on {len(test_seqs)} test sessions")

    if not test_seqs:
        raise RuntimeError("No test sequences after remapping.")

    # Load model
    num_users = data.get("num_users", 1)
    model     = GRU4Rec(num_items, num_users, eval_cfg).to(device)

    if model_key:
        if "s3" not in dir():
            s3 = get_client()
        tmp_path = download_model(s3, model_key)
        try:
            model.load_state_dict(torch.load(tmp_path, map_location=device))
        finally:
            os.unlink(tmp_path)
        log.info(f"Loaded model from MinIO: {model_key}")
    else:
        model.load_state_dict(torch.load(checkpoint, map_location=device))
        log.info(f"Loaded model from local: {checkpoint}")

    # Evaluate
    results = evaluate(
        model, test_seqs, eval_cfg, device,
        max_sessions=eval_cfg["max_eval_sessions"],
    )

    # Print summary
    top_n = eval_cfg["top_n"]
    log.info(f"\n{'='*52}")
    log.info(f"  session_HR@{top_n}:        {results['session_HR']:.4f}")
    log.info(f"  session_MRR@{top_n}:       {results['session_MRR']:.4f}")
    log.info(f"  session_precision@{top_n}: {results['session_precision']:.4f}")
    log.info(f"  session_recall@{top_n}:    {results['session_recall']:.4f}")
    log.info(f"  strict_HR@{top_n}:         {results['strict_HR']:.4f}")
    log.info(f"  strict_MRR@{top_n}:        {results['strict_MRR']:.4f}")
    log.info(f"  coverage:              {results['coverage']}")
    log.info(f"  total_predictions:     {results['total_predictions']}")
    log.info(f"  latency_mean_ms:       {results['latency_mean_ms']:.3f}")
    log.info(f"  latency_p95_ms:        {results['latency_p95_ms']:.3f}")
    log.info(f"  throughput_qps:        {results['throughput_qps']:.1f}")
    log.info(f"{'='*52}")

    return results


# ============================================================
# CLI
# ============================================================

def parse_args():
    p = argparse.ArgumentParser(description="Evaluate GRU4Rec — outputs session HR")

    # Model source
    mod = p.add_mutually_exclusive_group(required=True)
    mod.add_argument("--model-key",   help="MinIO key for model.pt")
    mod.add_argument("--checkpoint",  help="Local model checkpoint (.pt)")

    # Vocab source
    voc = p.add_mutually_exclusive_group(required=True)
    voc.add_argument("--vocab-key",  help="MinIO key for vocab.pkl")
    voc.add_argument("--vocab",      help="Local vocab pickle (item2idx)")

    p.add_argument("--data",          required=True, help="Data pickle (prepare_data() output)")
    p.add_argument("--top-n",         type=int, default=DEFAULT_CFG["top_n"])
    p.add_argument("--embedding-dim", type=int, default=DEFAULT_CFG["embedding_dim"])
    p.add_argument("--hidden-dim",    type=int, default=DEFAULT_CFG["hidden_dim"])
    p.add_argument("--num-layers",    type=int, default=DEFAULT_CFG["num_layers"])
    p.add_argument("--device",        default=DEFAULT_CFG["device"])
    p.add_argument("--max-sessions",  type=int, default=None,
                   help="Cap test sessions (useful for quick checks)")
    p.add_argument("--json-out",      default=None,
                   help="Write full results as JSON to this path")
    return p.parse_args()


def main():
    args = parse_args()

    eval_cfg = dict(DEFAULT_CFG)
    eval_cfg.update({
        "top_n":             args.top_n,
        "embedding_dim":     args.embedding_dim,
        "hidden_dim":        args.hidden_dim,
        "num_layers":        args.num_layers,
        "device":            args.device,
        "max_eval_sessions": args.max_sessions,
    })

    results = run_eval(
        model_key=args.model_key,
        vocab_key=args.vocab_key,
        checkpoint=args.checkpoint,
        vocab_path=args.vocab,
        data_path=args.data,
        eval_cfg=eval_cfg,
    )

    # Single-line output for easy script parsing
    print(f"\nsession_HR@{args.top_n}: {results['session_HR']:.4f}")

    if args.json_out:
        with open(args.json_out, "w") as f:
            json.dump(results, f, indent=2)
        log.info(f"Results written to {args.json_out}")


if __name__ == "__main__":
    main()
