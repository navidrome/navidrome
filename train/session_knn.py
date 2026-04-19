"""
30Music Dataset — Preprocessing & Session-based KNN Recommendation
===================================================================

All experiment settings are controlled by the single `cfg` dictionary below.
To run a different experiment, edit the keys in `cfg`, save, and re-run:

    python music_rec_pipeline.py

The cfg dict controls:
  - dataset paths and sampling
  - preprocessing thresholds
  - model selection and hyperparameters
  - evaluation settings
  - MLflow tracking

Inspired by the PyTorch Lightning config pattern used in BLIP-2 fine-tuning,
where a single cfg dict drives the entire training loop.
"""

import os
import json
import time
import logging
from urllib.parse import unquote_plus
from collections import defaultdict, Counter

import numpy as np
import pandas as pd
import mlflow

# ============================================================
# CONFIGURATION — edit this dict, save, and re-run
# ============================================================

cfg = {
    # ---- Dataset ----
    "dataset_root": "data/",  # root directory of the unzipped 30Music dataset
    "sample_sessions":      50000,      # max sessions to parse (None = full dataset)
    "sample_events":        200_000,     # max play events to parse (None = full)

    # ---- Preprocessing ----
    "min_session_length":   3,           # discard sessions shorter than this
    "max_session_length":   100,         # discard abnormally long sessions
    "min_item_support":     5,           # items must appear in >= N sessions
    "min_user_sessions":    3,           # users must have >= N sessions
    "skip_ratio_threshold": 0.25,        # playratio <= this = skip

    # ---- Model ----
    "model":                "popularity",      # "popularity" | "sknn"
    "sknn_k":               100,         # number of nearest-neighbor sessions
    "sknn_sample_size":     500,         # max candidates to compare per prediction
    "similarity":           "jaccard",   # similarity metric (jaccard | cosine)

    # ---- Evaluation ----
    "top_n":                20,          # recommendation list length
    "test_fraction":        0.2,         # temporal split ratio

    # ---- MLflow ----
    "mlflow_tracking_uri":  "http://129.114.27.248:8000",
    "mlflow_experiment":    "30music-session-recommendation",
}

# ---- Derived paths (do not edit) ----
ENTITIES_DIR = os.path.join(cfg["dataset_root"], "entities")
RELATIONS_DIR = os.path.join(cfg["dataset_root"], "relations")

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s | %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger(__name__)


# ============================================================
# PARSING
# ============================================================

def find_idomaar_file(directory, pattern):
    """Find a file matching pattern in directory (case-insensitive)."""
    if not os.path.exists(directory):
        raise FileNotFoundError(f"Directory not found: {directory}")
    for fname in os.listdir(directory):
        if pattern.lower() in fname.lower() and fname.endswith(".idomaar"):
            return os.path.join(directory, fname)
    for fname in os.listdir(directory):
        if pattern.lower() in fname.lower():
            return os.path.join(directory, fname)
    raise FileNotFoundError(
        f"No file matching '{pattern}' in {directory}. Files: {os.listdir(directory)}"
    )


def parse_sessions(filepath, cfg):
    """
    Parse sessions.idomaar.

    The format is unusual: the properties field contains two concatenated
    JSON objects — session-level stats and per-track play details.
    We extract the ordered (track_id, playratio) sequence from each.
    """
    max_rows = cfg["sample_sessions"]
    sessions = []
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

            raw_props = parts[3] if len(parts) > 3 else ""
            raw_linked = parts[4] if len(parts) > 4 else ""

            user_id = None
            track_sequence = []
            linked = None

            # First try: linked_entities in parts[4]
            if raw_linked:
                try:
                    linked = json.loads(raw_linked)
                except json.JSONDecodeError:
                    pass

            # Second try: split embedded JSON in properties field
            if linked is None and raw_props:
                brace_depth = 0
                split_pos = -1
                for ci, ch in enumerate(raw_props):
                    if ch == "{":
                        brace_depth += 1
                    elif ch == "}":
                        brace_depth -= 1
                        if brace_depth == 0 and ci < len(raw_props) - 1:
                            split_pos = ci + 1
                            break

                if split_pos > 0:
                    second_json = raw_props[split_pos:].strip()
                    if second_json:
                        try:
                            linked = json.loads(second_json)
                        except json.JSONDecodeError:
                            pass

            if linked is None:
                parse_errors += 1
                continue

            subjects = linked.get("subjects", [])
            if subjects:
                user_id = subjects[0].get("id")

            for obj in linked.get("objects", []):
                if obj.get("type") == "track":
                    track_sequence.append({
                        "track_id": obj["id"],
                        "playstart": obj.get("playstart", 0),
                        "playtime": obj.get("playtime", 0),
                        "playratio": obj.get("playratio"),
                        "action": obj.get("action", "play"),
                    })

            track_sequence.sort(key=lambda x: x.get("playstart", 0))

            if user_id is not None and len(track_sequence) > 0:
                sessions.append({
                    "session_id": session_id,
                    "user_id": user_id,
                    "timestamp": session_ts,
                    "num_tracks": len(track_sequence),
                    "tracks": track_sequence,
                })

    log.info(f"Parsed {len(sessions)} sessions ({parse_errors} parse errors)")
    return sessions


# ============================================================
# PREPROCESSING
# ============================================================

def sessions_to_dataframe(sessions, cfg):
    """Convert parsed sessions into session_df and interaction_df."""
    session_rows = []
    interaction_rows = []

    for s in sessions:
        session_rows.append({
            "session_id": s["session_id"],
            "user_id": s["user_id"],
            "timestamp": s["timestamp"],
            "num_tracks": s["num_tracks"],
        })
        for pos, t in enumerate(s["tracks"]):
            playratio = t.get("playratio")
            skipped = playratio is not None and playratio <= cfg["skip_ratio_threshold"]

            interaction_rows.append({
                "session_id": s["session_id"],
                "user_id": s["user_id"],
                "position": pos,
                "track_id": t["track_id"],
                "playtime": t.get("playtime", 0),
                "playratio": playratio,
                "skipped": skipped,
            })

    session_df = pd.DataFrame(session_rows)
    interaction_df = pd.DataFrame(interaction_rows)

    session_df["timestamp"] = pd.to_numeric(session_df["timestamp"], errors="coerce").astype("Int64")
    session_df["user_id"] = pd.to_numeric(session_df["user_id"], errors="coerce").astype("Int64")
    interaction_df["track_id"] = pd.to_numeric(interaction_df["track_id"], errors="coerce").astype("Int64")
    interaction_df["user_id"] = pd.to_numeric(interaction_df["user_id"], errors="coerce").astype("Int64")

    return session_df, interaction_df


def filter_data(session_df, interaction_df, cfg):
    """
    Apply quality filters controlled by cfg:
      min_session_length, max_session_length, min_item_support, min_user_sessions
    Removes skipped tracks, then iteratively filters until stable.
    """
    log.info(f"Before filtering: {len(session_df)} sessions, {len(interaction_df)} interactions")

    # Remove skips
    engaged_df = interaction_df[~interaction_df["skipped"]].copy()
    log.info(f"After removing skips: {len(engaged_df)} interactions "
             f"(removed {len(interaction_df) - len(engaged_df)} skips)")

    # Recompute session lengths
    session_lengths = engaged_df.groupby("session_id").size().reset_index(name="engaged_length")
    session_df = session_df.merge(session_lengths, on="session_id", how="left")
    session_df["engaged_length"] = session_df["engaged_length"].fillna(0).astype(int)

    # Filter by session length
    valid_sessions = session_df[
        (session_df["engaged_length"] >= cfg["min_session_length"])
        & (session_df["engaged_length"] <= cfg["max_session_length"])
    ]["session_id"]
    engaged_df = engaged_df[engaged_df["session_id"].isin(valid_sessions)]
    log.info(f"After session length filter "
             f"[{cfg['min_session_length']}, {cfg['max_session_length']}]: "
             f"{valid_sessions.nunique()} sessions")

    # Iterative item/user filtering
    for iteration in range(5):
        prev_size = len(engaged_df)

        item_counts = engaged_df["track_id"].value_counts()
        valid_items = item_counts[item_counts >= cfg["min_item_support"]].index
        engaged_df = engaged_df[engaged_df["track_id"].isin(valid_items)]

        sess_lens = engaged_df.groupby("session_id").size()
        valid_sids = sess_lens[sess_lens >= cfg["min_session_length"]].index
        engaged_df = engaged_df[engaged_df["session_id"].isin(valid_sids)]

        user_session_counts = engaged_df.groupby("user_id")["session_id"].nunique()
        valid_users = user_session_counts[user_session_counts >= cfg["min_user_sessions"]].index
        engaged_df = engaged_df[engaged_df["user_id"].isin(valid_users)]

        if len(engaged_df) == prev_size:
            log.info(f"Filtering converged at iteration {iteration + 1}")
            break

    session_df = session_df[session_df["session_id"].isin(engaged_df["session_id"].unique())]

    log.info(f"After filtering: {session_df['session_id'].nunique()} sessions, "
             f"{engaged_df['track_id'].nunique()} unique tracks, "
             f"{engaged_df['user_id'].nunique()} unique users, "
             f"{len(engaged_df)} interactions")

    return session_df, engaged_df


def build_session_sequences(interaction_df):
    """Build dict: session_id -> ordered list of track_ids."""
    sequences = {}
    for session_id, group in interaction_df.sort_values("position").groupby("session_id"):
        sequences[session_id] = group["track_id"].tolist()
    return sequences


def temporal_train_test_split(session_df, sequences, cfg):
    """Split sessions by timestamp: earlier for training, later for testing."""
    session_df = session_df.sort_values("timestamp")
    split_idx = int(len(session_df) * (1 - cfg["test_fraction"]))

    train_ids = set(session_df.iloc[:split_idx]["session_id"])
    test_ids = set(session_df.iloc[split_idx:]["session_id"])

    train_seq = {sid: seq for sid, seq in sequences.items() if sid in train_ids}
    test_seq = {sid: seq for sid, seq in sequences.items() if sid in test_ids}

    log.info(f"Train: {len(train_seq)} sessions | Test: {len(test_seq)} sessions")
    return train_seq, test_seq


# ============================================================
# MODELS
# ============================================================

class PopularityRecommender:
    """Baseline: always recommend the most globally popular items."""

    def __init__(self):
        self.popular_items = []

    def fit(self, train_sequences):
        counter = Counter()
        for seq in train_sequences.values():
            counter.update(seq)
        self.popular_items = counter.most_common()
        log.info(f"Popularity baseline fitted on {len(counter)} items")

    def predict(self, current_items, top_n=20):
        current_set = set(current_items)
        return [
            (item, count)
            for item, count in self.popular_items
            if item not in current_set
        ][:top_n]


class SessionKNN:
    """
    Session-based k-Nearest Neighbors Recommender.

    Uses an inverted index for fast candidate retrieval, then scores
    candidates by similarity-weighted voting from the top-K most
    similar historical sessions.
    """

    def __init__(self, cfg):
        self.k = cfg["sknn_k"]
        self.sample_size = cfg["sknn_sample_size"]
        self.similarity_fn = cfg["similarity"]
        self.item_to_sessions = defaultdict(set)
        self.session_items = {}
        self.item_popularity = Counter()

    def fit(self, train_sequences):
        log.info(f"Fitting S-KNN (k={self.k}, sample={self.sample_size}, "
                 f"sim={self.similarity_fn}) on {len(train_sequences)} sessions...")
        t0 = time.time()

        for session_id, item_list in train_sequences.items():
            item_set = set(item_list)
            self.session_items[session_id] = item_set
            for item in item_set:
                self.item_to_sessions[item].add(session_id)
                self.item_popularity[item] += 1

        elapsed = time.time() - t0
        log.info(f"Indexing done in {elapsed:.1f}s — "
                 f"{len(self.item_to_sessions)} unique items, "
                 f"{len(self.session_items)} sessions indexed")

    def _similarity(self, set_a, set_b):
        if self.similarity_fn == "cosine":
            intersection = len(set_a & set_b)
            denom = (len(set_a) ** 0.5) * (len(set_b) ** 0.5)
            return intersection / denom if denom > 0 else 0.0
        else:  # jaccard (default)
            intersection = len(set_a & set_b)
            union = len(set_a | set_b)
            return intersection / union if union > 0 else 0.0

    def predict(self, current_items, top_n=20):
        current_set = set(current_items)
        if not current_set:
            return [(item, count) for item, count in self.item_popularity.most_common(top_n)]

        # Find candidates via inverted index
        candidate_sessions = set()
        for item in current_items:
            candidate_sessions.update(self.item_to_sessions.get(item, set()))

        if not candidate_sessions:
            return [(item, count) for item, count in self.item_popularity.most_common(top_n)]

        # Sample for speed
        if len(candidate_sessions) > self.sample_size:
            candidate_sessions = set(
                np.random.choice(list(candidate_sessions), size=self.sample_size, replace=False)
            )

        # Compute similarities
        similarities = []
        for cand_sid in candidate_sessions:
            sim = self._similarity(current_set, self.session_items[cand_sid])
            if sim > 0:
                similarities.append((cand_sid, sim))

        similarities.sort(key=lambda x: x[1], reverse=True)
        top_neighbors = similarities[: self.k]

        # Weighted voting
        item_scores = defaultdict(float)
        for neighbor_sid, sim in top_neighbors:
            for item in self.session_items[neighbor_sid]:
                if item not in current_set:
                    item_scores[item] += sim

        ranked = sorted(item_scores.items(), key=lambda x: x[1], reverse=True)
        return ranked[:top_n]


# ============================================================
# EVALUATION
# ============================================================

def evaluate(model, test_sequences, cfg):
    """
    Dual evaluation protocol for session-based music recommendation.

    For each test session [A, B, C, D, E], at each split point (e.g. prefix=[A,B]):

    1. STRICT (next-item): Does the prediction list contain C (the exact next track)?
       - Standard protocol used in published session-rec papers (Jannach et al. 2018)
       - Allows comparison against published baselines on 30Music
       - Overly harsh for music: predicting D or E gets zero credit

    2. SESSION (session-membership): Does the prediction list contain ANY of [C,D,E]?
       - Better reflects real music use: any remaining track is a good recommendation
       - HR_session: did we hit at least one remaining track?
       - MRR_session: reciprocal rank of the highest-ranked remaining track
       - Precision_session: what fraction of the top-N are remaining session tracks?
       - Recall_session: what fraction of remaining session tracks did we retrieve?

    Both are reported so you can compare against papers (strict) while also
    measuring what actually matters for a music player (session).

    Serving speed is also tracked per prediction call (mean, p50, p95, p99, max,
    and overall throughput in queries/sec) and included in the returned results dict.
    """
    top_n = cfg["top_n"]
    log.info(f"Evaluating on {len(test_sequences)} test sessions (top-{top_n})...")
    t0 = time.time()

    # Strict next-item counters
    strict_hits = 0
    strict_mrr_sum = 0.0

    # Session-membership counters
    session_hits = 0
    session_mrr_sum = 0.0
    session_precision_sum = 0.0
    session_recall_sum = 0.0

    total_predictions = 0
    all_recommended_items = set()

    # Serving speed: per-call wall time in milliseconds (time.perf_counter for
    # high resolution; avoids GIL / OS scheduling noise better than time.time)
    predict_latencies_ms = []

    for session_id, full_sequence in test_sequences.items():
        for split_point in range(1, len(full_sequence)):
            prefix = full_sequence[:split_point]
            next_item = full_sequence[split_point]                  # strict ground truth
            remaining = set(full_sequence[split_point:])            # session ground truth

            t_pred = time.perf_counter()
            predictions = model.predict(prefix, top_n=top_n)
            predict_latencies_ms.append((time.perf_counter() - t_pred) * 1000)

            predicted_items = [item for item, _ in predictions]
            all_recommended_items.update(predicted_items)
            total_predictions += 1

            # ---- Strict: exact next item ----
            if next_item in predicted_items:
                strict_hits += 1
                strict_mrr_sum += 1.0 / (predicted_items.index(next_item) + 1)

            # ---- Session: any remaining item ----
            hit_positions = [
                i for i, item in enumerate(predicted_items) if item in remaining
            ]

            if hit_positions:
                session_hits += 1
                session_mrr_sum += 1.0 / (hit_positions[0] + 1)  # best (earliest) hit

            # Precision: fraction of top-N that are relevant
            n_relevant_in_topn = len(hit_positions)
            session_precision_sum += n_relevant_in_topn / top_n

            # Recall: fraction of remaining items we retrieved
            if len(remaining) > 0:
                session_recall_sum += n_relevant_in_topn / len(remaining)

    elapsed = time.time() - t0

    n = total_predictions if total_predictions > 0 else 1  # avoid division by zero

    # Latency percentiles (numpy operates on the full list in one pass)
    lat = np.array(predict_latencies_ms) if predict_latencies_ms else np.array([0.0])

    results = {
        # Strict next-item metrics (for paper comparison)
        "strict_HR":          strict_hits / n,
        "strict_MRR":         strict_mrr_sum / n,

        # Session-membership metrics (for real-world relevance)
        "session_HR":         session_hits / n,
        "session_MRR":        session_mrr_sum / n,
        "session_precision":  session_precision_sum / n,
        "session_recall":     session_recall_sum / n,

        # Serving speed — per predict() call
        "latency_mean_ms":    float(np.mean(lat)),
        "latency_p50_ms":     float(np.percentile(lat, 50)),
        "latency_p95_ms":     float(np.percentile(lat, 95)),
        "latency_p99_ms":     float(np.percentile(lat, 99)),
        "latency_max_ms":     float(np.max(lat)),
        "throughput_qps":     float(total_predictions / elapsed) if elapsed > 0 else 0.0,

        # General
        "coverage":           len(all_recommended_items),
        "total_predictions":  total_predictions,
    }

    log.info(f"Evaluation done in {elapsed:.1f}s over {total_predictions} predictions")
    log.info(f"")
    log.info(f"  Strict (next-item):      HR@{top_n}={results['strict_HR']:.4f}  "
             f"MRR@{top_n}={results['strict_MRR']:.4f}")
    log.info(f"  Session (any remaining):  HR@{top_n}={results['session_HR']:.4f}  "
             f"MRR@{top_n}={results['session_MRR']:.4f}  "
             f"P@{top_n}={results['session_precision']:.4f}  "
             f"R@{top_n}={results['session_recall']:.4f}")
    log.info(f"  Coverage: {results['coverage']} unique items recommended")
    log.info(f"  Serving speed:  mean={results['latency_mean_ms']:.2f}ms  "
             f"p50={results['latency_p50_ms']:.2f}ms  "
             f"p95={results['latency_p95_ms']:.2f}ms  "
             f"p99={results['latency_p99_ms']:.2f}ms  "
             f"max={results['latency_max_ms']:.2f}ms  "
             f"QPS={results['throughput_qps']:.1f}")

    return results


# ============================================================
# MAIN — reads everything from cfg
# ============================================================

def main():

    # Print config for reproducibility (like BLIP-2 script does)
    log.info("=" * 60)
    log.info("[cfg] " + json.dumps(cfg, indent=2, default=str))
    log.info("=" * 60)

    # ---- MLflow setup ----
    mlflow.set_tracking_uri(cfg["mlflow_tracking_uri"])
    mlflow.set_experiment(cfg["mlflow_experiment"])
    log.info(f"MLflow: {cfg['mlflow_tracking_uri']}  experiment: {cfg['mlflow_experiment']}")

    # ---- Verify dataset ----
    for d, label in [(ENTITIES_DIR, "ENTITIES"), (RELATIONS_DIR, "RELATIONS")]:
        if os.path.exists(d):
            log.info(f"{label}/: {os.listdir(d)}")
        else:
            log.error(f"Directory not found: {d}")
            return

    # ---- Parse ----
    sessions_file = find_idomaar_file(RELATIONS_DIR, "sessions")
    log.info(f"Parsing sessions from: {sessions_file}")
    sessions = parse_sessions(sessions_file, cfg)
    if not sessions:
        log.error("No sessions parsed! Check file format.")
        return

    # ---- Preprocess ----
    session_df, interaction_df = sessions_to_dataframe(sessions, cfg)

    raw_stats = {
        "raw_sessions": session_df.shape[0],
        "raw_interactions": interaction_df.shape[0],
        "raw_users": int(interaction_df["user_id"].nunique()),
        "raw_tracks": int(interaction_df["track_id"].nunique()),
    }
    log.info(f"Raw: {raw_stats}")

    valid_ratios = interaction_df["playratio"].dropna()
    data_stats = {
        "skip_rate": round(float(interaction_df["skipped"].mean()), 4),
        "playratio_mean": round(float(valid_ratios.mean()), 4) if len(valid_ratios) > 0 else 0,
        "playratio_median": round(float(valid_ratios.median()), 4) if len(valid_ratios) > 0 else 0,
    }

    session_df, interaction_df = filter_data(session_df, interaction_df, cfg)
    sequences = build_session_sequences(interaction_df)
    train_sequences, test_sequences = temporal_train_test_split(session_df, sequences, cfg)

    if len(train_sequences) == 0 or len(test_sequences) == 0:
        log.error("Empty train or test set! Increase sample_sessions or relax filters.")
        return

    filtered_stats = {
        "filtered_sessions": session_df["session_id"].nunique(),
        "filtered_users": int(interaction_df["user_id"].nunique()),
        "filtered_tracks": int(interaction_df["track_id"].nunique()),
        "filtered_interactions": len(interaction_df),
        "train_sessions": len(train_sequences),
        "test_sessions": len(test_sequences),
    }

    # ---- Build model from cfg ----
    if cfg["model"] == "popularity":
        model = PopularityRecommender()
        run_name = "popularity_baseline"
        model_tag = "PopularityBaseline"
    elif cfg["model"] == "sknn":
        model = SessionKNN(cfg)
        run_name = f"sknn_k{cfg['sknn_k']}_s{cfg['sknn_sample_size']}_{cfg['similarity']}"
        model_tag = "SessionKNN"
    else:
        log.error(f"Unknown model: {cfg['model']}")
        return

    # ---- Train & evaluate inside a single MLflow run ----
    with mlflow.start_run(run_name=run_name) as run:

        # Log the full cfg as params (flat)
        mlflow.log_params({k: str(v) for k, v in cfg.items()})
        mlflow.log_params(raw_stats)
        mlflow.log_params(data_stats)
        mlflow.log_params(filtered_stats)
        mlflow.set_tags({
            "model_type": model_tag,
            "dataset": "30Music",
        })

        # Fit
        t0 = time.time()
        model.fit(train_sequences)
        fit_time = time.time() - t0

        # Evaluate
        results = evaluate(model, test_sequences, cfg)

        # Log metrics
        top_n = cfg["top_n"]
        mlflow.log_metrics({
            # Strict (paper-comparable)
            f"strict_HR_at_{top_n}":   results["strict_HR"],
            f"strict_MRR_at_{top_n}":  results["strict_MRR"],

            # Session-membership (real-world relevance)
            f"session_HR_at_{top_n}":        results["session_HR"],
            f"session_MRR_at_{top_n}":       results["session_MRR"],
            f"session_precision_at_{top_n}": results["session_precision"],
            f"session_recall_at_{top_n}":    results["session_recall"],

            # Serving speed
            "latency_mean_ms":    results["latency_mean_ms"],
            "latency_p50_ms":     results["latency_p50_ms"],
            "latency_p95_ms":     results["latency_p95_ms"],
            "latency_p99_ms":     results["latency_p99_ms"],
            "latency_max_ms":     results["latency_max_ms"],
            "throughput_qps":     results["throughput_qps"],

            # General
            "coverage": results["coverage"],
            "total_predictions": results["total_predictions"],
            "fit_time_seconds": round(fit_time, 2),
        })

        # Extra model-specific metrics
        if cfg["model"] == "sknn":
            mlflow.log_metrics({
                "num_indexed_items": len(model.item_to_sessions),
                "num_indexed_sessions": len(model.session_items),
            })

        log.info(f"MLflow run ID: {run.info.run_id}")

    # ---- Summary ----
    log.info("\n" + "=" * 60)
    log.info("RESULTS")
    log.info("=" * 60)
    log.info(f"  Model:              {model_tag}  (cfg['model'] = {cfg['model']})")
    log.info(f"")
    log.info(f"  Strict (next-item):")
    log.info(f"    HR@{top_n}:           {results['strict_HR']:.4f}")
    log.info(f"    MRR@{top_n}:          {results['strict_MRR']:.4f}")
    log.info(f"")
    log.info(f"  Session (any remaining):")
    log.info(f"    HR@{top_n}:           {results['session_HR']:.4f}")
    log.info(f"    MRR@{top_n}:          {results['session_MRR']:.4f}")
    log.info(f"    Precision@{top_n}:    {results['session_precision']:.4f}")
    log.info(f"    Recall@{top_n}:       {results['session_recall']:.4f}")
    log.info(f"")
    log.info(f"  Serving speed:")
    log.info(f"    mean latency:     {results['latency_mean_ms']:.2f} ms/query")
    log.info(f"    p50 latency:      {results['latency_p50_ms']:.2f} ms")
    log.info(f"    p95 latency:      {results['latency_p95_ms']:.2f} ms")
    log.info(f"    p99 latency:      {results['latency_p99_ms']:.2f} ms")
    log.info(f"    max latency:      {results['latency_max_ms']:.2f} ms")
    log.info(f"    throughput:       {results['throughput_qps']:.1f} queries/sec")
    log.info(f"")
    log.info(f"  Coverage:           {results['coverage']}")
    log.info(f"  Fit time:           {fit_time:.1f}s")
    log.info(f"\n  MLflow UI: {cfg['mlflow_tracking_uri']}")
    log.info(f"  Experiment: {cfg['mlflow_experiment']}")


if __name__ == "__main__":
    main()