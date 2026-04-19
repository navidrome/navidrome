# Use the PyTorch-Notebook with CUDA 12 support for H100 compatibility
FROM quay.io/jupyter/pytorch-notebook:cuda12-python-3.12

USER root

# 1. Install system dependencies for extraction and H100 monitoring
RUN apt-get update && apt-get install -y --no-install-recommends \
    nvtop \
    tar \
    gzip \
    wget \
    gnupg \
    build-essential \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# 2. Install gdown to handle the Google Drive dataset download
RUN python3 -m pip install --no-cache-dir gdown

# 3. Upgrade CUDA Toolkit for Hopper (H100) Architecture support
RUN wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb && \
    dpkg -i cuda-keyring_1.1-1_all.deb && \
    apt-get update && \
    apt-get install -y cuda-toolkit-12-8 && \
    rm cuda-keyring_1.1-1_all.deb 

ENV CUDA_HOME=/usr/local/cuda
ENV PATH="${CUDA_HOME}/bin:${PATH}"
ENV LD_LIBRARY_PATH="${CUDA_HOME}/lib64:${LD_LIBRARY_PATH}"

# 4. Define the working directory inside the container
WORKDIR /home/${NB_USER}/work

# 5. Download and Extract Dataset directly into the build
# ID from your link: 1n2aaVgfUgz_8Uif4hCmI5meyX4jDhI6u
RUN gdown 1n2aaVgfUgz_8Uif4hCmI5meyX4jDhI6u -O ThirtyMusic.tar.gz && \
    mkdir -p data/ThirtyMusic && \
    tar -xzf ThirtyMusic.tar.gz -C data/ThirtyMusic --strip-components=1 && \
    rm ThirtyMusic.tar.gz

# 6. Copy your entire repository context
# This works because we will run the build from the repo root
COPY . .

# 7. Install Python dependencies
# We use the relative path from the repo root
RUN python -m pip install --no-cache-dir --upgrade pip wheel setuptools && \
    python -m pip install --no-cache-dir -r train/requirements.txt && \
    python -m pip install --no-cache-dir bash_kernel && \
    python -m bash_kernel.install --sys-prefix

# 8. CRITICAL: Fix permissions as ROOT before switching users
# This prevents the exit code: 1 error you saw
RUN chown -R ${NB_UID}:${NB_GID} /home/${NB_USER}/work && \
    fix-permissions "${CONDA_DIR}" && \
    fix-permissions "/home/${NB_USER}"

# 9. Switch to the non-root user for security and Jupyter standards
USER ${NB_UID}

# Set the MLflow Tracking URI for the H100 environment
ENV MLFLOW_TRACKING_URI="http://129.114.27.248:8000"

# Default command to start Jupyter
CMD ["start-notebook.sh", "--NotebookApp.token=''"]
