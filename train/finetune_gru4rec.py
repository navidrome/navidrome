"""
Fine-tuning script for GRU4Rec on new session data from the same 30Music catalog.

The item vocabulary is fixed at pretraining time and never changes — only songs
from the training catalog are served in Navidrome. Fine-tuning adapts the model
to new listening sessions on that same fixed catalog.

Model artifacts are stored in MinIO (never on local disk). Set env vars:
    MINIO_URL       e.g. http://minio:9000
    MINIO_USER      your-access-key
    MINIO_PASSWORD  your-secret-key
    MINIO_BUCKET    (optional) defaults to "gru4rec-models"

Expects two inputs:
  1. Pretrained vocab key in MinIO:
        pretrain/{date}/{run_id}/vocab.pkl

     OR a local pretrain_vocab.pkl (item2idx saved after the original prepare_data() call):
        data = prepare_data(cfg)
        pickle.dump({"item2idx": data["item2idx"]}, open("pretrain_vocab.pkl", "wb"))

  2. finetune_data.pkl  — prepare_data() output from new session data:
        {
            "item2idx":   dict[track_id -> int],
            "train_seqs": list[{session_id, user_idx, item_idxs, playratios}],
            "test_seqs":  list[{session_id, user_idx, item_idxs, playratios}],
            "num_items":  int,
            "num_users":  int,
        }

Usage:
    # Vocab and checkpoint from MinIO:
    python finetune_gru4rec.py \\
        --pretrain-model-key pretrain/2026-04-18/{run_id}/model.pt \\
        --pretrain-vocab-key pretrain/2026-04-18/{run_id}/vocab.pkl \\
        --finetune-data      finetune_data.pkl

    # Checkpoint and vocab from local files (e.g. first run):
    python finetune_gru4rec.py \\
        --checkpoint     best_gru4rec.pt \\
        --pretrain-vocab pretrain_vocab.pkl \\
        --finetune-data  finetune_data.pkl
"""

import io
import os
import sys
import json
import time
import pickle
import logging
import argparse
import tempfile
from datetime import datetime, timezone

import requests

import torch
import mlflow
from torch.utils.data import DataLoader

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gru4rec import (
    GRU4Rec,
    SessionDataset,
    collate_fn,
    train_epoch,
    evaluate,
    collect_environment_info,
    log_environment_to_mlflow,
    get_gpu_memory_stats,
)
try:
    from minio_store import get_client, push_run_artifacts, download_model, download_vocab
    MINIO_AVAILABLE = True
except ImportError:
    MINIO_AVAILABLE = False

logging.basicConfig(level=logging.INFO, format="%(asctime)s | %(message)s", datefmt="%H:%M:%S")
log = logging.getLogger(__name__)


# ============================================================
# DEFAULT CONFIG
# ============================================================

FINETUNE_CFG = {
    # ---- Model arch (must match pretrained checkpoint) ----
    "embedding_dim":        64,
    "hidden_dim":           128,
    "num_layers":           1,
    "dropout":              0.2,
    "embedding_dropout":    0.25,
    "use_user_context":     False,

    # ---- Fine-tuning hyperparams ----
    "epochs":               10,
    "batch_size":           1024,
    "lr":                   1e-4,       # lower than pretraining
    "weight_decay":         1e-5,
    "use_playratio_weight": False,
    "lr_step_size":         5,
    "lr_gamma":             0.5,
    "patience":             3,
    "label_smoothing":      0.0,

    # ---- Evaluation ----
    "top_n":                20,
    "eval_every_n_epochs":  2,
    "eval_batch_size":      2048,
    "max_eval_sessions":    None,
    "full_eval_at_end":     True,

    # ---- Hardware ----
    "device":               "cuda" if torch.cuda.is_available() else "cpu",
    "num_workers":          4,

    # ---- MLflow ----
    "mlflow_tracking_uri":  os.environ.get("MLFLOW_TRACKING_URI", "http://129.114.27.204:8000"),
    "mlflow_experiment":    "30music-session-recommendation-finetune",

    # ---- Cache / Swift ----
    "cache_dir":            ".cache_gru4rec",
    "finetune_dataset_version": "",   # e.g. "v20260419-finetune-001"; if set, data is pulled from Swift
}


# ============================================================
# SWIFT DATA LOADING
# ============================================================

def prepare_finetune_data(cfg: dict) -> dict:
    """
    Downloads fine-tune sequences from the Swift bucket.

    Swift URL:
        https://chi.uc.chameleoncloud.org:7480/swift/v1/AUTH_7c0a7a1952e44c94aa75cae1ff5dc9b4/navidrome-bucket-proj05/datasets/{version}/
    Expected files: train_sequences.pkl, test_sequences.pkl, item2idx.json, user2idx.json
    """
    version   = cfg["finetune_dataset_version"]
    base_url  = (
        f"https://chi.uc.chameleoncloud.org:7480/swift/v1/"
        f"AUTH_7c0a7a1952e44c94aa75cae1ff5dc9b4/navidrome-bucket-proj05/datasets/{version}"
    )
    cache_dir = cfg.get("cache_dir", ".cache_gru4rec")
    os.makedirs(cache_dir, exist_ok=True)

    def download(fname):
        local = os.path.join(cache_dir, f"{version}_{fname}")
        if os.path.exists(local):
            log.info(f"[cache] Using cached {fname}")
            return local
        log.info(f"Downloading {fname} from Swift ...")
        r = requests.get(f"{base_url}/{fname}", timeout=300, stream=True)
        r.raise_for_status()
        with open(local, "wb") as fh:
            for chunk in r.iter_content(chunk_size=8192):
                fh.write(chunk)
        log.info(f"Downloaded {fname} ({os.path.getsize(local)/1e6:.1f} MB)")
        return local

    train_path    = download("train_sequences.pkl")
    test_path     = download("test_sequences.pkl")
    item2idx_path = download("item2idx.json")
    user2idx_path = download("user2idx.json")

    log.info("Loading train_sequences.pkl ...")
    with open(train_path, "rb") as fh:
        train_seqs = pickle.load(fh)

    log.info("Loading test_sequences.pkl ...")
    with open(test_path, "rb") as fh:
        test_seqs = pickle.load(fh)

    log.info("Loading item2idx.json ...")
    with open(item2idx_path) as fh:
        item2idx_raw = json.load(fh)
    item2idx = {int(k): int(v) for k, v in item2idx_raw.items()}

    log.info("Loading user2idx.json ...")
    with open(user2idx_path) as fh:
        user2idx_raw = json.load(fh)
    user2idx = {int(k): int(v) for k, v in user2idx_raw.items()}

    log.info(
        f"Fine-tune dataset {version} loaded: "
        f"{len(train_seqs):,} train seqs, "
        f"{len(test_seqs):,} test seqs, "
        f"{len(item2idx):,} items, "
        f"{len(user2idx):,} users"
    )

    return {
        "item2idx":   item2idx,
        "user2idx":   user2idx,
        "train_seqs": train_seqs,
        "test_seqs":  test_seqs,
        "num_items":  len(item2idx),
        "num_users":  len(user2idx),
    }


# ============================================================
# SEQUENCE REMAPPING
# ============================================================

def remap_sequences(seqs: list, ft_item2idx: dict, pretrain_item2idx: dict) -> list:
    """
    Reindex fine-tune sequences to use the pretrained item indices.

    Fine-tune data has its own item2idx built by prepare_data() which may assign
    different indices to the same track IDs. We reverse-map back to track IDs
    then look them up in the fixed pretrained item2idx.

    Items not present in the pretrained catalog are silently dropped.
    Sequences that shrink below length 2 are discarded.
    """
    idx2track = {v: k for k, v in ft_item2idx.items()}

    remapped      = []
    dropped_seqs  = 0
    dropped_items = 0

    for seq in seqs:
        new_items = []
        for ft_idx in seq["item_idxs"]:
            track_id = idx2track.get(ft_idx)
            if track_id in pretrain_item2idx:
                new_items.append(pretrain_item2idx[track_id])
            else:
                dropped_items += 1

        if len(new_items) >= 2:
            remapped.append({
                "session_id": seq["session_id"],
                "user_idx":   seq["user_idx"],
                "item_idxs":  new_items,
                "playratios": seq["playratios"][: len(new_items)],
            })
        else:
            dropped_seqs += 1

    if dropped_items:
        log.warning(
            f"Dropped {dropped_items} interactions not in pretrained catalog "
            f"(expected 0 if catalog is identical)"
        )
    if dropped_seqs:
        log.warning(f"Dropped {dropped_seqs} sequences with < 2 items after remapping")

    return remapped


# ============================================================
# FINE-TUNING RUN
# ============================================================

def run_finetuning(
    ft_cfg: dict,
    checkpoint_path: str,       # local .pt file OR None
    pretrain_model_key: str,    # MinIO key OR None
    pretrain_vocab_path: str,   # local vocab pkl OR None
    pretrain_vocab_key: str,    # MinIO vocab key OR None
    finetune_data_path: str = None,   # local pickle (prepare_data() output) OR None
    ft_data: dict = None,             # pre-loaded data dict (from prepare_finetune_data) OR None
    data_version: str = "",     # dataset ID / version tag for lineage tracking
) -> dict:

    device = torch.device(ft_cfg["device"])
    s3     = get_client() if MINIO_AVAILABLE and (pretrain_vocab_key or pretrain_model_key) else None

    # 1. Load pretrained vocab (fixed catalog)
    if pretrain_vocab_key:
        pretrain_vocab = download_vocab(s3, pretrain_vocab_key)
    else:
        with open(pretrain_vocab_path, "rb") as f:
            pretrain_vocab = pickle.load(f)

    pretrain_item2idx = pretrain_vocab["item2idx"]
    num_items         = len(pretrain_item2idx)
    log.info(f"Pretrained catalog: {num_items} items")

    # 2. Load fine-tune data
    if ft_data is None:
        if finetune_data_path is None:
            raise ValueError("Either ft_data or finetune_data_path must be provided")
        with open(finetune_data_path, "rb") as f:
            ft_data = pickle.load(f)
    log.info(
        f"Fine-tune data: {len(ft_data['train_seqs'])} train, "
        f"{len(ft_data['test_seqs'])} test sessions"
    )

    # 3. Remap sequences to pretrained item indices
    train_seqs = remap_sequences(ft_data["train_seqs"], ft_data["item2idx"], pretrain_item2idx)
    test_seqs  = remap_sequences(ft_data["test_seqs"],  ft_data["item2idx"], pretrain_item2idx)
    log.info(f"After remapping: {len(train_seqs)} train, {len(test_seqs)} test sequences")

    if not train_seqs:
        raise RuntimeError(
            "No training sequences after remapping. "
            "Ensure fine-tune data comes from the same 30Music catalog."
        )

    # 4. Load pretrained model weights
    num_users = ft_data.get("num_users", 1)
    model     = GRU4Rec(num_items, num_users, ft_cfg).to(device)

    if pretrain_model_key:
        tmp_ckpt = download_model(s3, pretrain_model_key)
        try:
            model.load_state_dict(torch.load(tmp_ckpt, map_location=device))
        finally:
            os.unlink(tmp_ckpt)
        log.info(f"Loaded checkpoint from MinIO: {pretrain_model_key}")
    else:
        model.load_state_dict(torch.load(checkpoint_path, map_location=device))
        log.info(f"Loaded checkpoint from local: {checkpoint_path}")

    # 5. DataLoader
    train_dataset = SessionDataset(train_seqs, ft_cfg["use_playratio_weight"])
    train_loader  = DataLoader(
        train_dataset,
        batch_size=ft_cfg["batch_size"],
        shuffle=True,
        num_workers=ft_cfg["num_workers"],
        collate_fn=collate_fn,
        pin_memory=(device.type == "cuda"),
        persistent_workers=(ft_cfg["num_workers"] > 0),
        drop_last=True,
    )
    log.info(f"Train samples: {len(train_dataset)}")

    # 6. Optimiser
    optimizer = torch.optim.Adam(
        model.parameters(), lr=ft_cfg["lr"], weight_decay=ft_cfg["weight_decay"]
    )
    scheduler = torch.optim.lr_scheduler.StepLR(
        optimizer, step_size=ft_cfg["lr_step_size"], gamma=ft_cfg["lr_gamma"]
    )
    scaler = torch.cuda.amp.GradScaler(enabled=(device.type == "cuda"))

    if device.type == "cuda":
        torch.cuda.reset_peak_memory_stats()

    env_info = collect_environment_info(ft_cfg["device"])

    # 7. MLflow + training loop
    mlflow.set_tracking_uri(ft_cfg["mlflow_tracking_uri"])
    mlflow.set_experiment(ft_cfg["mlflow_experiment"])

    run_name = (
        f"finetune_gru4rec_e{ft_cfg['embedding_dim']}"
        f"_h{ft_cfg['hidden_dim']}_l{ft_cfg['num_layers']}"
    )

    best_session_hr    = 0.0
    best_state_dict    = None
    epochs_no_improve  = 0
    final_results      = {}
    patience           = ft_cfg["patience"]
    eval_every         = ft_cfg["eval_every_n_epochs"]
    max_eval           = ft_cfg["max_eval_sessions"]
    epochs_trained     = 0

    with mlflow.start_run(run_name=run_name) as run:
        mlflow.log_params({k: str(v) for k, v in ft_cfg.items()})
        mlflow.log_params({
            "num_items":           num_items,
            "train_sessions":      len(train_seqs),
            "test_sessions":       len(test_seqs),
            "train_samples":       len(train_dataset),
            "pretrain_model_key":  pretrain_model_key or os.path.basename(checkpoint_path or ""),
            "finetune_data_version": data_version or (os.path.basename(finetune_data_path) if finetune_data_path else ""),
        })
        mlflow.set_tags({"model_type": "GRU4Rec-InBatch-Finetune", "run_type": "finetune"})
        log_environment_to_mlflow(env_info)

        t_start = time.time()

        for epoch in range(1, ft_cfg["epochs"] + 1):
            t0 = time.time()

            avg_loss, samples = train_epoch(model, train_loader, optimizer, scaler, device, ft_cfg)
            scheduler.step()
            epochs_trained = epoch

            epoch_time = time.time() - t0
            throughput = samples / epoch_time if epoch_time > 0 else 0
            gpu_mem    = get_gpu_memory_stats()

            mlflow.log_metrics(
                {
                    "train_loss":                 avg_loss,
                    "epoch_time_sec":             round(epoch_time, 2),
                    "wall_time_sec":              round(time.time() - t_start, 2),
                    "throughput_samples_per_sec": round(throughput, 1),
                    "learning_rate":              optimizer.param_groups[0]["lr"],
                    "gpu_mem_allocated_mb":       gpu_mem["gpu_mem_allocated_mb"],
                    "gpu_mem_peak_mb":            gpu_mem["gpu_mem_peak_mb"],
                },
                step=epoch,
            )

            log.info(
                f"Epoch {epoch:02d}/{ft_cfg['epochs']} | "
                f"loss={avg_loss:.4f} | {epoch_time:.1f}s | {throughput:.0f} samp/s"
            )

            is_last = epoch == ft_cfg["epochs"]
            if epoch % eval_every == 0 or is_last:
                t_eval     = time.time()
                eval_limit = None if (is_last and ft_cfg["full_eval_at_end"]) else max_eval
                results    = evaluate(model, test_seqs, ft_cfg, device, max_sessions=eval_limit)
                eval_time  = time.time() - t_eval

                top_n = ft_cfg["top_n"]
                mlflow.log_metrics(
                    {
                        f"session_HR{top_n}":        results["session_HR"],
                        f"session_MRR{top_n}":       results["session_MRR"],
                        f"strict_HR{top_n}":         results["strict_HR"],
                        f"strict_MRR{top_n}":        results["strict_MRR"],
                        f"session_precision{top_n}": results["session_precision"],
                        f"session_recall{top_n}":    results["session_recall"],
                        "coverage":                  results["coverage"],
                        "eval_time_sec":             round(eval_time, 2),
                        "latency_mean_ms":           results["latency_mean_ms"],
                        "latency_p95_ms":            results["latency_p95_ms"],
                    },
                    step=epoch,
                )

                log.info(
                    f"  Session HR{top_n}={results['session_HR']:.4f}  "
                    f"MRR={results['session_MRR']:.4f}  "
                    f"(eval {eval_time:.1f}s)"
                )

                if results["session_HR"] > best_session_hr:
                    best_session_hr   = results["session_HR"]
                    epochs_no_improve = 0
                    best_state_dict   = {k: v.cpu().clone() for k, v in model.state_dict().items()}
                    log.info(f"  -> New best session HR{top_n}: {best_session_hr:.4f}")

                    if MINIO_AVAILABLE:
                        _meta = {
                            "run_type":              "finetune",
                            "mlflow_run_id":         run.info.run_id,
                            "timestamp":             datetime.now(timezone.utc).isoformat(),
                            "session_HR":            round(best_session_hr, 6),
                            "session_MRR":           round(results.get("session_MRR", 0), 6),
                            "strict_HR":             round(results.get("strict_HR", 0), 6),
                            "num_items":             num_items,
                            "embedding_dim":         ft_cfg["embedding_dim"],
                            "hidden_dim":            ft_cfg["hidden_dim"],
                            "num_layers":            ft_cfg["num_layers"],
                            "epoch":                 epoch,
                            "pretrain_source":       pretrain_model_key or checkpoint_path or "",
                            "finetune_data_version": data_version or (os.path.basename(finetune_data_path) if finetune_data_path else ""),
                            "gpu_name":              env_info.get("gpu_name", ""),
                            "git_sha":               env_info.get("git_sha", ""),
                        }
                        try:
                            _keys = push_run_artifacts(
                                state_dict=best_state_dict,
                                run_type="finetune",
                                run_id=run.info.run_id,
                                metadata=_meta,
                            )
                            mlflow.set_tags({
                                "minio_model_key":    _keys["model_key"],
                                "minio_metadata_key": _keys["metadata_key"],
                            })
                            log.info(f"  [minio] model    → {_keys['model_key']}")
                            log.info(f"  [minio] metadata → {_keys['metadata_key']}")
                        except Exception as _e:
                            log.warning(f"  [minio] Upload skipped — {_e}")
                else:
                    epochs_no_improve += 1
                    if epochs_no_improve >= patience:
                        log.info(f"  Early stopping at epoch {epoch}")
                        break

                final_results = results

        total_time = time.time() - t_start
        mlflow.log_metrics({
            "total_finetune_seconds": round(total_time, 2),
            "total_finetune_minutes": round(total_time / 60, 2),
            "best_session_HR":        best_session_hr,
            "epochs_trained":         epochs_trained,
        })

        log.info(
            f"\nDone. {total_time:.1f}s ({total_time/60:.1f} min) | "
            f"Best HR{ft_cfg['top_n']}: {best_session_hr:.4f} | "
            f"MLflow run: {run.info.run_id}"
        )

    final_results["best_session_HR"]       = best_session_hr
    final_results["total_finetune_seconds"] = total_time
    final_results["mlflow_run_id"]          = run.info.run_id
    return final_results


# ============================================================
# CLI
# ============================================================

def parse_args():
    p = argparse.ArgumentParser(description="Fine-tune GRU4Rec on new Navidrome session data")

    # Checkpoint source — MinIO key OR local file
    src = p.add_mutually_exclusive_group(required=True)
    src.add_argument("--pretrain-model-key", help="MinIO key for pretrained model.pt")
    src.add_argument("--checkpoint",         help="Local pretrained model weights (.pt)")

    # Vocab source — MinIO key OR local file
    voc = p.add_mutually_exclusive_group(required=True)
    voc.add_argument("--pretrain-vocab-key", help="MinIO key for pretrained vocab.pkl")
    voc.add_argument("--pretrain-vocab",     help="Local pretrained vocab pickle (item2idx)")

    # Fine-tune data source — Swift version tag OR local pickle (mutually exclusive)
    ft_data_src = p.add_mutually_exclusive_group(required=True)
    ft_data_src.add_argument(
        "--finetune-data-version",
        help="Swift dataset version to pull (e.g. v20260419-finetune-001). "
             "Downloads train_sequences.pkl, test_sequences.pkl, item2idx.json, user2idx.json "
             "from the Swift bucket.",
    )
    ft_data_src.add_argument(
        "--finetune-data",
        help="Local fine-tune data pickle (prepare_data() output)",
    )

    # Arch — must match the pretrained checkpoint
    p.add_argument("--embedding-dim", type=int,   default=FINETUNE_CFG["embedding_dim"])
    p.add_argument("--hidden-dim",    type=int,   default=FINETUNE_CFG["hidden_dim"])
    p.add_argument("--num-layers",    type=int,   default=FINETUNE_CFG["num_layers"])

    # Training
    p.add_argument("--epochs",     type=int,   default=FINETUNE_CFG["epochs"])
    p.add_argument("--lr",         type=float, default=FINETUNE_CFG["lr"])
    p.add_argument("--batch-size", type=int,   default=FINETUNE_CFG["batch_size"])
    p.add_argument("--top-n",      type=int,   default=FINETUNE_CFG["top_n"])
    p.add_argument("--patience",   type=int,   default=FINETUNE_CFG["patience"])

    # Infra
    p.add_argument("--device",       default=FINETUNE_CFG["device"])
    p.add_argument("--mlflow-uri",   default=FINETUNE_CFG["mlflow_tracking_uri"])
    p.add_argument("--experiment",   default=FINETUNE_CFG["mlflow_experiment"])
    p.add_argument("--cache-dir",    default=FINETUNE_CFG["cache_dir"],
                   help="Local directory for caching downloaded Swift files")
    p.add_argument("--data-version", default="",
                   help="Dataset version/ID tag for lineage (e.g. v20260419-live). "
                        "Defaults to the finetune data filename.")
    return p.parse_args()


def main():
    args = parse_args()

    ft_cfg = dict(FINETUNE_CFG)
    ft_cfg.update({
        "embedding_dim":       args.embedding_dim,
        "hidden_dim":          args.hidden_dim,
        "num_layers":          args.num_layers,
        "epochs":              args.epochs,
        "lr":                  args.lr,
        "batch_size":          args.batch_size,
        "top_n":               args.top_n,
        "patience":            args.patience,
        "device":              args.device,
        "mlflow_tracking_uri": args.mlflow_uri,
        "mlflow_experiment":   args.experiment,
        "cache_dir":           args.cache_dir,
    })

    log.info(json.dumps(ft_cfg, indent=2, default=str))

    # Pull fine-tune data from Swift if a version tag was given
    ft_data_dict = None
    if args.finetune_data_version:
        ft_cfg["finetune_dataset_version"] = args.finetune_data_version
        ft_data_dict = prepare_finetune_data(ft_cfg)

    results = run_finetuning(
        ft_cfg=ft_cfg,
        checkpoint_path=args.checkpoint,
        pretrain_model_key=args.pretrain_model_key,
        pretrain_vocab_path=args.pretrain_vocab,
        pretrain_vocab_key=args.pretrain_vocab_key,
        finetune_data_path=args.finetune_data,
        ft_data=ft_data_dict,
        data_version=args.data_version or args.finetune_data_version or "",
    )

    print(f"\nbest_session_HR@{ft_cfg['top_n']}: {results['best_session_HR']:.4f}")
    print(f"mlflow_run_id: {results['mlflow_run_id']}")


if __name__ == "__main__":
    main()
