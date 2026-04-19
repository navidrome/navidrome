# Lightweight CPU-only image for KNN session recommendation
# No ROCm, no PyTorch — just Python + pandas + mlflow + psutil
#
# Code and data are NOT baked in — they are mounted at runtime:
#   -v /home/cc/navidrome_mlops/train:/app/train   (script)
#   -v /home/cc/knn/data:/app/data                 (dataset)
#
# Dockerfile lives at: navidrome_mlops/train/docker_knn/Dockerfile
FROM python:3.11-slim

# ---- system deps (minimal) ----
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    && rm -rf /var/lib/apt/lists/*

# ---- Python deps ----
RUN pip install --no-cache-dir \
    pandas \
    numpy \
    mlflow \
    psutil \
    tqdm

# ---- working directory ----
# train/ and data/ are bind-mounted here at runtime
WORKDIR /app

# ---- env ----
ENV MLFLOW_TRACKING_URI="http://129.114.27.248:8000"
ENV PYTHONUNBUFFERED=1

CMD ["python", "train/music_rec_pipeline_aligned.py"]