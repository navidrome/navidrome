# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.148.1/containers/go/.devcontainer/base.Dockerfile

# [Choice] Go version: 1, 1.15, 1.14
ARG VARIANT="1"
FROM mcr.microsoft.com/vscode/devcontainers/go:${VARIANT}

# [Option] Install Node.js
ARG INSTALL_NODE="true"
ARG NODE_VERSION="lts/*"
RUN if [ "${INSTALL_NODE}" = "true" ]; then su vscode -c "source /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# Install additional OS packages
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends ffmpeg

# Install TagLib from cross-taglib releases
ARG CROSS_TAGLIB_VERSION="2.1.1-1"
ARG TARGETARCH
RUN DOWNLOAD_ARCH="linux-${TARGETARCH}" \
    && wget -q "https://github.com/navidrome/cross-taglib/releases/download/v${CROSS_TAGLIB_VERSION}/taglib-${DOWNLOAD_ARCH}.tar.gz" -O /tmp/cross-taglib.tar.gz \
    && tar -xzf /tmp/cross-taglib.tar.gz -C /usr --strip-components=1 \
    && mv /usr/include/taglib/* /usr/include/ \
    && rmdir /usr/include/taglib \
    && rm /tmp/cross-taglib.tar.gz /usr/provenance.json

ENV CGO_CFLAGS_ALLOW="--define-prefix"

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1
