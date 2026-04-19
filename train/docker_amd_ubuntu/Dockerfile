FROM ubuntu:22.04

ARG ROCM_VERSION=7.0.1
ARG AMDGPU_VERSION=7.0.1.70001
ARG DEBIAN_FRONTEND=noninteractive

# 1. Install system prerequisites
RUN apt-get update && apt-get install -y --no-install-recommends \
    tar \
    gzip \
    wget \
    gnupg \
    build-essential \
    cmake \
    libncurses-dev \
    libsystemd-dev \
    libudev-dev \
    libdrm-dev \
    git \
    python3 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/*

# 2. Install AMD ROCm system stack
RUN wget https://repo.radeon.com/amdgpu-install/$ROCM_VERSION/ubuntu/jammy/amdgpu-install_$AMDGPU_VERSION-1_all.deb && \
    apt-get update --yes && \
    apt-get install --yes ./amdgpu-install_$AMDGPU_VERSION-1_all.deb && \
    amdgpu-install --usecase=rocm -y && \
    rm *.deb

# 3. Build nvtop from source with AMD support
RUN git clone https://github.com/Syllo/nvtop && \
    mkdir -p nvtop/build && cd nvtop/build && \
    cmake -DAMDGPU_SUPPORT=ON -DNVIDIA_SUPPORT=OFF .. && \
    make && make install && \
    cd ../.. && rm -rf nvtop

# 4. Install PyTorch with ROCm compatibility and Lightning
RUN python3 -m pip install --no-cache-dir --index-url https://download.pytorch.org/whl/rocm7.0 \
    'torch' \
    'torchaudio' \
    'torchvision'

RUN python3 -m pip install --no-cache-dir 'lightning'

# 5. Create a non-root user
RUN useradd -m -s /bin/bash appuser

# 6. Define the working directory
WORKDIR /home/appuser/work

# 7. Copy your entire repository context
COPY . .

# 8. Install your project's Python dependencies
RUN python3 -m pip install --no-cache-dir --upgrade pip wheel setuptools && \
    python3 -m pip install --no-cache-dir -r train/requirements.txt

# 9. Fix permissions
RUN chown -R appuser:appuser /home/appuser/work

# 10. Switch to non-root user
USER appuser

ENV MLFLOW_TRACKING_URI="http://129.114.25.55:8000"
