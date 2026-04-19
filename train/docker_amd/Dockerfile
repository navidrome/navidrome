FROM ubuntu:22.04
ARG ROCM_VERSION=7.0.1
ARG AMDGPU_VERSION=7.0.1.70001
ARG DEBIAN_FRONTEND=noninteractive

# Apt retry/timeout config to reduce transient network failures
RUN echo 'Acquire::Retries "5";' > /etc/apt/apt.conf.d/80-retries && \
    echo 'Acquire::http::Timeout "120";' >> /etc/apt/apt.conf.d/80-retries

# 1. System prerequisites
RUN apt-get update && apt-get install -y --no-install-recommends \
    tar gzip wget gnupg build-essential cmake \
    libncurses-dev libsystemd-dev libudev-dev libdrm-dev git \
    python3 python3-pip python3-venv \
    && rm -rf /var/lib/apt/lists/*

# 2a. Register ROCm installer (cached separately from the heavy install below)
RUN wget https://repo.radeon.com/amdgpu-install/$ROCM_VERSION/ubuntu/jammy/amdgpu-install_$AMDGPU_VERSION-1_all.deb && \
    apt-get update --yes && \
    apt-get install --yes ./amdgpu-install_$AMDGPU_VERSION-1_all.deb

# 2b. Run the actual ROCm install (separate layer — retries don't re-download .deb)
RUN amdgpu-install --usecase=rocm -y && rm *.deb

# 3. nvtop from source with AMD support
RUN git clone https://github.com/Syllo/nvtop && \
    mkdir -p nvtop/build && cd nvtop/build && \
    cmake -DAMDGPU_SUPPORT=ON -DNVIDIA_SUPPORT=OFF .. && \
    make && make install && \
    cd ../.. && rm -rf nvtop

ENV PATH="/opt/rocm/bin/:$PATH"
ENV NCCL_P2P_DISABLE=1

# 4. PyTorch with ROCm (--pre matches the ideal for latest compatible wheels)
RUN python3 -m pip install --pre --no-cache-dir \
    --index-url https://download.pytorch.org/whl/rocm7.0 \
    'torch' 'torchaudio' 'torchvision'

RUN python3 -m pip install --pre --no-cache-dir lightning

# 5. ROCm GPU monitoring libs (keep separate from project deps)
RUN python3 -m pip install --pre --no-cache-dir pyrsmi amdsmi

# 6. Non-root user
RUN useradd -m -s /bin/bash appuser

WORKDIR /home/appuser/work

COPY . .

# 7. Project dependencies
RUN python3 -m pip install --no-cache-dir --upgrade pip wheel setuptools && \
    python3 -m pip install --no-cache-dir -r requirements.txt

RUN chown -R appuser:appuser /home/appuser/work

USER appuser

ENV MLFLOW_TRACKING_URI="http://129.114.27.204:8000"
ENV MINIO_URL="http://129.114.27.204:9000"
ENV MINIO_BUCKET="artifacts"

# MINIO_USER and MINIO_PASSWORD are passed at runtime — never baked into the image:
# docker run --env-file .env ...  OR  docker run -e MINIO_USER=... -e MINIO_PASSWORD=... ...
