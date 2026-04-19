"""
MinIO storage utilities for GRU4Rec model artifacts.

Connects via boto3 (S3-compatible). Configure with env vars:
    MINIO_URL       e.g. http://minio:9000
    MINIO_USER      access key
    MINIO_PASSWORD  secret key
    MINIO_BUCKET    (optional) defaults to "gru4rec-models"

Bucket layout:
    gru4rec-models/
      pretrain/{YYYY-MM-DD}/{mlflow_run_id}/
        model.pt        — model weights (state_dict)
        vocab.pkl       — item2idx saved at training time
        metadata.json   — full metrics + config snapshot
      finetune/{YYYY-MM-DD}/{mlflow_run_id}/
        model.pt
        metadata.json

Each object is also tagged with a lightweight subset of metadata so the
MinIO UI / S3 ListObjectsV2 queries can filter without downloading files.
"""

import io
import json
import logging
import os
import pickle
import tempfile
from datetime import datetime, timezone

import boto3
from botocore.exceptions import ClientError

log = logging.getLogger(__name__)

DEFAULT_BUCKET = "gru4rec-models"


# ============================================================
# CONNECTION
# ============================================================

def get_client():
    """Return a boto3 S3 client pointed at MinIO.

    Reads MINIO_URL, MINIO_USER, MINIO_PASSWORD from the environment.
    """
    url  = os.environ.get("MINIO_URL")
    user = os.environ.get("MINIO_USER")
    pwd  = os.environ.get("MINIO_PASSWORD")

    if not all([url, user, pwd]):
        raise EnvironmentError(
            "Set MINIO_URL, MINIO_USER, and MINIO_PASSWORD environment variables "
            "before using minio_store."
        )

    return boto3.client(
        "s3",
        endpoint_url=url,
        aws_access_key_id=user,
        aws_secret_access_key=pwd,
        region_name="us-east-1",   # required by boto3, ignored by MinIO
    )


def ensure_bucket(s3, bucket: str):
    """Create the bucket if it does not already exist."""
    try:
        s3.head_bucket(Bucket=bucket)
    except ClientError as e:
        if e.response["Error"]["Code"] in ("404", "NoSuchBucket"):
            s3.create_bucket(Bucket=bucket)
            log.info(f"[minio] Created bucket: {bucket}")
        else:
            raise


# ============================================================
# KEY HELPERS
# ============================================================

def _run_prefix(run_type: str, run_id: str) -> str:
    """
    e.g. pretrain/2026-04-18/abc123def456/
    """
    date = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    return f"{run_type}/{date}/{run_id}"


def _tags(metadata: dict) -> list:
    """
    Convert metadata dict to S3 TagSet list.
    Tags are capped at 10 (S3 limit); we pick the most useful subset.
    Values must be strings <= 256 chars.
    """
    priority_keys = [
        "run_type", "mlflow_run_id", "timestamp",
        "session_HR", "num_items",
        "embedding_dim", "hidden_dim", "num_layers",
        "epochs_trained",
    ]
    tags = []
    for k in priority_keys:
        if k in metadata:
            tags.append({"Key": k, "Value": str(metadata[k])[:256]})
    return tags


# ============================================================
# UPLOAD
# ============================================================

def upload_model(
    s3,
    state_dict,
    run_type: str,
    run_id: str,
    metadata: dict,
    bucket: str = None,
) -> str:
    """Serialise and upload a model state_dict to MinIO.

    Returns the full object key so callers can log it.
    """
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)
    ensure_bucket(s3, bucket)

    key = f"{_run_prefix(run_type, run_id)}/model.pt"

    buf = io.BytesIO()
    import torch
    torch.save(state_dict, buf)
    buf.seek(0)

    s3.upload_fileobj(
        buf, bucket, key,
        ExtraArgs={"ContentType": "application/octet-stream"},
    )
    s3.put_object_tagging(
        Bucket=bucket, Key=key,
        Tagging={"TagSet": _tags(metadata)},
    )

    log.info(f"[minio] Uploaded model → s3://{bucket}/{key}")
    return key


def upload_vocab(s3, vocab: dict, run_type: str, run_id: str, bucket: str = None) -> str:
    """Serialise and upload item2idx vocab to MinIO."""
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)
    ensure_bucket(s3, bucket)

    key = f"{_run_prefix(run_type, run_id)}/vocab.pkl"

    buf = io.BytesIO()
    pickle.dump(vocab, buf)
    buf.seek(0)

    s3.upload_fileobj(
        buf, bucket, key,
        ExtraArgs={"ContentType": "application/octet-stream"},
    )

    log.info(f"[minio] Uploaded vocab → s3://{bucket}/{key}")
    return key


def upload_metadata(
    s3,
    metadata: dict,
    run_type: str,
    run_id: str,
    bucket: str = None,
) -> str:
    """Upload the full metadata dict as metadata.json."""
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)
    ensure_bucket(s3, bucket)

    key = f"{_run_prefix(run_type, run_id)}/metadata.json"

    body = json.dumps(metadata, indent=2, default=str).encode()
    s3.put_object(
        Bucket=bucket, Key=key,
        Body=body,
        ContentType="application/json",
    )

    log.info(f"[minio] Uploaded metadata → s3://{bucket}/{key}")
    return key


# ============================================================
# DOWNLOAD
# ============================================================

def download_model(s3, model_key: str, bucket: str = None) -> str:
    """Download a model from MinIO to a local temp file.

    Returns the path to the temp file. Caller is responsible for
    deleting it when done (or use as a context manager).
    """
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)

    tmp = tempfile.NamedTemporaryFile(suffix=".pt", delete=False)
    s3.download_fileobj(bucket, model_key, tmp)
    tmp.flush()
    tmp.close()

    log.info(f"[minio] Downloaded model s3://{bucket}/{model_key} → {tmp.name}")
    return tmp.name


def download_vocab(s3, vocab_key: str, bucket: str = None) -> dict:
    """Download and deserialise a vocab pickle from MinIO."""
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)

    buf = io.BytesIO()
    s3.download_fileobj(bucket, vocab_key, buf)
    buf.seek(0)

    log.info(f"[minio] Downloaded vocab s3://{bucket}/{vocab_key}")
    return pickle.load(buf)


def download_metadata(s3, run_type: str, run_id: str,
                       date: str = None, bucket: str = None) -> dict:
    """Download and parse metadata.json for a given run."""
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)
    date   = date or datetime.now(timezone.utc).strftime("%Y-%m-%d")
    key    = f"{run_type}/{date}/{run_id}/metadata.json"

    buf = io.BytesIO()
    s3.download_fileobj(bucket, key, buf)
    buf.seek(0)
    return json.loads(buf.read().decode())


# ============================================================
# DISCOVERY — find latest model
# ============================================================

def get_latest_model_key(s3, run_type: str = "finetune", bucket: str = None) -> str | None:
    """Return the MinIO key of the most recently uploaded model.pt for run_type.

    Searches finetune/ first; falls back to pretrain/ if nothing found and
    run_type was 'finetune'.  Returns None if the bucket is empty.

    Key layout: {run_type}/{YYYY-MM-DD}/{run_id}/model.pt
    """
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)

    def _latest_in(prefix):
        paginator = s3.get_paginator("list_objects_v2")
        best_key, best_ts = None, None
        for page in paginator.paginate(Bucket=bucket, Prefix=prefix):
            for obj in page.get("Contents", []):
                if obj["Key"].endswith("/model.pt"):
                    if best_ts is None or obj["LastModified"] > best_ts:
                        best_key = obj["Key"]
                        best_ts  = obj["LastModified"]
        return best_key

    key = _latest_in(f"{run_type}/")
    if key is None and run_type == "finetune":
        log.info("[minio] No finetune model found, falling back to pretrain/")
        key = _latest_in("pretrain/")

    if key:
        log.info(f"[minio] Latest {run_type} model: {key}")
    else:
        log.warning(f"[minio] No model.pt found under {run_type}/ in bucket {bucket}")

    return key


# ============================================================
# POPULARITY (cold-start)
# ============================================================

def upload_popularity(s3, local_path: str, key: str, bucket: str = None) -> str:
    """Upload a popularity.npy file to MinIO under an explicit key.

    Key convention: shared/{dataset_version}/popularity.npy
    Returns the object key.
    """
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)
    ensure_bucket(s3, bucket)

    with open(local_path, "rb") as fh:
        s3.upload_fileobj(
            fh, bucket, key,
            ExtraArgs={"ContentType": "application/octet-stream"},
        )

    log.info(f"[minio] Uploaded popularity → s3://{bucket}/{key}")
    return key


def download_popularity(s3, key: str, local_path: str, bucket: str = None):
    """Download popularity.npy from MinIO to local_path."""
    bucket = bucket or os.environ.get("MINIO_BUCKET", DEFAULT_BUCKET)

    import os as _os
    _os.makedirs(_os.path.dirname(local_path) or ".", exist_ok=True)

    with open(local_path, "wb") as fh:
        s3.download_fileobj(bucket, key, fh)

    log.info(f"[minio] Downloaded popularity s3://{bucket}/{key} → {local_path}")


# ============================================================
# CONVENIENCE: upload all artifacts in one call
# ============================================================

def push_run_artifacts(
    state_dict,
    run_type: str,
    run_id: str,
    metadata: dict,
    vocab: dict = None,
    bucket: str = None,
) -> dict:
    """Upload model (+ optional vocab) + metadata in one call.

    Returns a dict of object keys for logging:
        {"model_key": ..., "vocab_key": ..., "metadata_key": ...}
    """
    s3 = get_client()

    model_key    = upload_model(s3, state_dict, run_type, run_id, metadata, bucket)
    metadata_key = upload_metadata(s3, metadata, run_type, run_id, bucket)
    vocab_key    = None
    if vocab is not None:
        vocab_key = upload_vocab(s3, vocab, run_type, run_id, bucket)

    return {
        "model_key":    model_key,
        "metadata_key": metadata_key,
        "vocab_key":    vocab_key,
    }
