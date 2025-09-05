# The Dockerfile facilitates multi-architecture builds. However, be cautious as trufflehog currently only support linux/arm64 and linux/amd64 architectures. 
# Always verify the compatibility of third-party versions before building.
# Important: As of now, Semgrep does not support ARM architectures - see https://github.com/returntocorp/semgrep/issues/2252 for details!

# Default Plugins' List
# Dependencies will be installed if the docker file supports it, othervise ignored and only compile binaries of plugins
ARG PLUGINS="github,gitlab,bitbucket,semgrep,bandit,trufflehog"

# Stage 1: Build Scanio core and plugins
FROM golang:1.24.2-alpine3.21 AS build-scanio

WORKDIR /usr/src/scanio

# Copy go.mod and go.sum for dependency resolution
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Set target architecture for multi-arch builds
ARG TARGETOS
ARG TARGETARCH
ARG PLUGINS

# Install make and other build dependencies
RUN apk update && \
    apk upgrade && \
    apk add --no-cache \
    make \
    jq

# Build the core and plugins using the Makefile
RUN echo "Building binaries and plugins for '$TARGETOS/$TARGETARCH'"
RUN make build PLUGINS=$PLUGINS CORE_BINARY=/usr/bin/scanio PLUGINS_DIR=/usr/bin/plugins

# Stage 2: Prepare the runtime environment
FROM alpine:3.21.3 as runtime

# RUN addgroup -g 101 scanio && \
#     adduser -h /home/scanio -s /bin/bash --uid 1001 -G scanio -D scanio && \
#     chown -R scanio:scanio $SCANIO_PLUGINS_FOLDER && \
#     chown -R scanio:scanio $SCANIO_HOME

# USER scanio:scanio

# Set target architecture for multi-arch builds
ARG TARGETOS
ARG TARGETARCH
ARG PLUGINS

RUN set -euxo pipefail && \
    echo "Building dependencies for '$TARGETOS/$TARGETARCH'" && \
    apk update && \
    apk upgrade && \
    apk add --no-cache bash python3 py3-pip openssh && \
    apk add --no-cache --virtual .build-deps \
        jq \
        libc6-compat \
        gcc \
        openssl \
        ca-certificates \
        curl \
        git \
        musl-dev && \
    PLUGIN_VENVS_DIR="/opt/venvs" && \
    mkdir -p "$PLUGIN_VENVS_DIR" && \
    echo "Installing plugins: $PLUGINS" && \
    for plugin in $(echo "$PLUGINS" | tr ',' ' '); do \
      case "$plugin" in \
        semgrep) \
          echo "Installing Semgrep..."; \
          python3 -m venv "$PLUGIN_VENVS_DIR/semgrep" && \
          . "$PLUGIN_VENVS_DIR/semgrep/bin/activate" && \
          pip install --no-cache-dir semgrep==1.120.1 ;; \
        trufflehog3) \
          echo "Installing Trufflehog3..."; \
          apk add --no-cache git; \
          python3 -m venv "$PLUGIN_VENVS_DIR/trufflehog3" && \
          . "$PLUGIN_VENVS_DIR/trufflehog3/bin/activate" && \
          pip install --no-cache-dir trufflehog3==3.0.10 ;; \
        bandit) \
          echo "Installing Bandit..."; \
          python3 -m venv "$PLUGIN_VENVS_DIR/bandit" && \
          . "$PLUGIN_VENVS_DIR/bandit/bin/activate" && \
          pip install --no-cache-dir bandit==1.8.3 ;; \
        trufflehog) \
          echo "Installing TruffleHog binary..."; \
          TRUFFLEHOG_VER="3.88.27" && \
          TARFILE="trufflehog_${TRUFFLEHOG_VER}_${TARGETOS}_${TARGETARCH}.tar.gz" && \
          CHECKSUMFILE="trufflehog_${TRUFFLEHOG_VER}_checksums.txt" && \
          curl -LOs "https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VER}/${CHECKSUMFILE}" && \
          curl -LOs "https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VER}/${TARFILE}" && \
          grep "${TARFILE}" "${CHECKSUMFILE}" | sha256sum -c - && \
          tar -xzf "${TARFILE}" && \
          mv trufflehog /usr/local/bin/ && \
          rm -f "${TARFILE}" "${CHECKSUMFILE}" ;; \
        *) echo "No dependencies installed for plugin: $plugin" ;; \
      esac; \
    done && \
    apk del .build-deps && \
    find /usr -name '*.o' -delete && \
    find /usr -name '*.a' -delete && \
    rm -rf /var/cache/apk/* && \
    find /usr -name '__pycache__' -exec rm -rf {} + && \
    rm -rf /root/.cache/pip

# Set PATH for venv manually
ENV PATH="/opt/venvs/semgrep/bin:/opt/venvs/trufflehog3/bin:/opt/venvs/bandit/bin:$PATH"

# Create necessary directories
RUN mkdir -p /scanio /data

# Copy built binaries and other necessary files from the build stage
COPY --from=build-scanio /usr/bin/scanio /bin/scanio
COPY --from=build-scanio /usr/bin/plugins/ /scanio/plugins/

# Copy additional resources
COPY rules /scanio/rules
COPY templates /scanio/templates
COPY VERSION /scanio/VERSION
COPY config.yml /scanio/config.yml

# Write to config.yml customized values
RUN echo -e "\n\nscanio:" >> /scanio/config.yml && \
    echo -e "  home_folder: /scanio" >> /scanio/config.yml && \
    echo -e "  plugins_folder: /scanio/plugins" >> /scanio/config.yml && \
    echo -e "  projects_folder: /scanio/projects" >> /scanio/config.yml && \
    echo -e "  results_folder: /scanio/results" >> /scanio/config.yml && \
    echo -e "  temp_folder: /scanio/tmp\n" >> /scanio/config.yml

ENTRYPOINT ["/bin/scanio"]
CMD ["--help"]