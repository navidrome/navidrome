"""
Popularity-Based Cold Start for GRU4Rec
========================================

Blends GRU4Rec session scores with global popularity scores when a session
is too short to have reliable GRU signal.

Usage
-----
    from cold_start import ColdStartRecommender

    # Build once after data is loaded
    cs = ColdStartRecommender(train_seqs, num_items)

    # At inference time, replaces model.predict_top_n
    predictions = cs.predict(model, prefix_items, user_idxs, all_item_emb, top_n=20)
"""

import logging
import torch
import numpy as np
from collections import Counter

log = logging.getLogger(__name__)


class ColdStartRecommender:
    """
    Wraps a trained GRU4Rec model and blends its scores with popularity scores
    based on session length.

    Blend weight alpha = min(session_len / ramp_sessions, 1.0)
      - alpha=0.0  -> pure popularity  (session length 0)
      - alpha=1.0  -> pure GRU4Rec     (session length >= ramp_sessions)

    Parameters
    ----------
    train_seqs : list of dicts with key "item_idxs" (1-indexed item ids)
    num_items  : total number of items (vocab size, excluding padding index 0)
    ramp_sessions : number of interactions before fully trusting GRU4Rec
    popularity_temp : temperature for softmax over log-counts (lower = peakier)
    """

    def __init__(
        self,
        train_seqs: list,
        num_items: int,
        ramp_sessions: int = 3,
        popularity_temp: float = 1.0,
    ):
        self.num_items       = num_items
        self.ramp_sessions   = ramp_sessions
        self.popularity_temp = popularity_temp

        self._pop_scores = self._build_popularity(train_seqs)   # (num_items,) tensor

    # ------------------------------------------------------------------
    # Build
    # ------------------------------------------------------------------

    def _build_popularity(self, train_seqs: list) -> torch.Tensor:
        """
        Count item occurrences across all training sequences.
        Returns a (num_items,) float tensor of log-smoothed popularity scores,
        indexed by item_idx - 1  (item_idx 1 → index 0).
        """
        counts = Counter()
        for seq in train_seqs:
            for item_idx in seq["item_idxs"]:
                counts[item_idx] += 1

        raw = torch.zeros(self.num_items, dtype=torch.float)
        for item_idx, cnt in counts.items():
            arr_idx = item_idx - 1          # item_idx is 1-based
            if 0 <= arr_idx < self.num_items:
                raw[arr_idx] = cnt

        # log-smooth then temperature-scale; zero-count items get -inf after log
        log_counts = torch.log1p(raw)       # log(1 + count), safe for zeros
        pop_scores = log_counts / self.popularity_temp
        return pop_scores                   # unnormalized; will be softmax'd together with gru scores

    # ------------------------------------------------------------------
    # Inference
    # ------------------------------------------------------------------

    def _alpha(self, session_len: int) -> float:
        """Blend coefficient: 0 = pure popularity, 1 = pure GRU4Rec."""
        return min(session_len / self.ramp_sessions, 1.0)

    @torch.no_grad()
    def predict(
        self,
        model,
        prefix_items: torch.Tensor,     # (B, L) padded, right-padded, 0=pad
        user_idxs: torch.Tensor,        # (B,)
        all_item_emb: torch.Tensor,     # (num_items, D)  model.item_emb.weight[1:]
        top_n: int,
        exclude_sets: list = None,      # list of B sets of item_idxs to exclude
    ) -> list:
        """
        Returns list of B lists, each containing top_n item indices (1-based).
        """
        device = prefix_items.device
        B      = prefix_items.size(0)

        if exclude_sets is None:
            exclude_sets = [set() for _ in range(B)]

        pop = self._pop_scores.to(device)           # (num_items,)

        # Session lengths per sample (non-padding tokens)
        lengths = (prefix_items != 0).sum(dim=1)    # (B,)

        # GRU4Rec scores for all items — (B, num_items)
        raw_model    = model.module if hasattr(model, "module") else model
        session_repr = raw_model.encode_session(prefix_items, user_idxs)   # (B, D)
        gru_scores   = session_repr @ all_item_emb.T                       # (B, num_items)

        # Normalise both score distributions to comparable scale (softmax in log-space)
        gru_log   = torch.log_softmax(gru_scores, dim=-1)   # (B, num_items)
        pop_log   = torch.log_softmax(pop.unsqueeze(0).expand(B, -1), dim=-1)  # (B, num_items)

        # Per-sample blending
        results = []
        for b in range(B):
            alpha  = self._alpha(int(lengths[b].item()))
            scores = alpha * gru_log[b] + (1.0 - alpha) * pop_log[b]   # (num_items,)

            # Mask excluded items
            for item_idx in exclude_sets[b]:
                arr_idx = item_idx - 1
                if 0 <= arr_idx < self.num_items:
                    scores[arr_idx] = float("-inf")

            top_indices = torch.topk(scores, top_n).indices.cpu().tolist()
            results.append([i + 1 for i in top_indices])   # back to 1-based

        return results

    # ------------------------------------------------------------------
    # Diagnostics
    # ------------------------------------------------------------------

    def save_popularity(self, path: str):
        """
        Save popularity scores as a .npy file for the serving layer.
        Load with: ColdStartBlender.from_file(path)
        """
        import numpy as np
        np.save(path, self._pop_scores.numpy())
        log.info(f"[cold_start] Popularity scores saved -> {path}")

    def top_popular(self, n: int = 20) -> list:
        """Return the top-n most popular item indices (1-based)."""
        indices = torch.topk(self._pop_scores, n).indices.tolist()
        return [i + 1 for i in indices]

    def alpha_schedule(self) -> list:
        """Show the blend schedule for session lengths 0..ramp_sessions."""
        return [
            {"session_len": l, "alpha": round(self._alpha(l), 3),
             "gru_weight_%": round(self._alpha(l) * 100), "pop_weight_%": round((1 - self._alpha(l)) * 100)}
            for l in range(self.ramp_sessions + 1)
        ]
