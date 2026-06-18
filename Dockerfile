# The Dockerfile facilitates multi-architecture builds. However, be cautious as trufflehog currently only support linux/arm64 and linux/amd64 architectures. 
# Always verify the compatibility of third-party versions before building.
# Important: As of now, Semgrep does not support ARM architectures - see https://github.com/returntocorp/semgrep/issues/2252 for details!

# Default Plugins' List
# Dependencies will be installed if the docker file supports it, othervise ignored and only compile binaries of plugins
ARG PLUGINS="github,gitlab,bitbucket,semgrep,bandit,trufflehog"

# Custom binary name and in-image path root. Filesystem relabel only;
# does not change the SCANIO_ env prefix, scanio: config key, or magic cookie.
ARG APP_NAME=scanio

# Stage 1: Build Scanio core and plugins
FROM golang:1.25.9-alpine3.23 AS build-scanio

WORKDIR /usr/src/scanio

# Copy go.mod and go.sum for dependency resolution
COPY go.mod go.sum ./
RUN go mod download

# Copy build inputs only — runtime assets (rules/, templates/, config.yml) are
# brought in by the runtime stage below.
COPY Makefile VERSION main.go ./
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
COPY plugins/ plugins/

# Set target architecture for multi-arch builds
ARG TARGETOS
ARG TARGETARCH
ARG PLUGINS
ARG APP_NAME

# Install make and other build dependencies
RUN apk update && \
    apk upgrade && \
    apk add --no-cache \
    make \
    jq

# Build the core and plugins using the Makefile
RUN echo "Building binaries and plugins for '$TARGETOS/$TARGETARCH'"
RUN make build PLUGINS="$PLUGINS" CORE_BINARY=/usr/bin/$APP_NAME PLUGINS_DIR=/usr/bin/plugins

# Stage 2: Prepare the runtime environment
FROM alpine:3.23.4 AS runtime

# Set target architecture for multi-arch builds
ARG TARGETOS
ARG TARGETARCH
ARG PLUGINS
ARG APP_NAME

RUN set -euxo pipefail && \
    echo "Building dependencies for '$TARGETOS/$TARGETARCH'" && \
    apk update && \
    apk upgrade && \
    apk add --no-cache bash python3 py3-pip openssh git && \
    apk add --no-cache --virtual .build-deps \
        jq \
        libc6-compat \
        gcc \
        openssl \
        ca-certificates \
        curl \
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
          pip install --no-cache-dir semgrep==1.161.0 ;; \
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
          pip install --no-cache-dir bandit==1.9.4 ;; \
        trufflehog) \
          echo "Installing TruffleHog binary..."; \
          TRUFFLEHOG_VER="3.95.2" && \
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

RUN mkdir -p /$APP_NAME /$APP_NAME/plugins /$APP_NAME/rules /$APP_NAME/templates \
          /$APP_NAME/projects /$APP_NAME/results /$APP_NAME/tmp /$APP_NAME/artifacts /$APP_NAME/log /data

# Copy built binaries and other necessary files from the build stage
COPY --from=build-scanio /usr/bin/$APP_NAME /bin/$APP_NAME
COPY --from=build-scanio /usr/bin/plugins/ /$APP_NAME/plugins/

# Copy additional resources
COPY rules /$APP_NAME/rules
COPY templates /$APP_NAME/templates
COPY VERSION /$APP_NAME/VERSION
COPY config.yml /$APP_NAME/config.yml

# Set PATH for venv manually
ENV PATH="/opt/venvs/semgrep/bin:/opt/venvs/trufflehog3/bin:/opt/venvs/bandit/bin:${PATH}"

# Write to config.yml customized values
RUN echo -e "\n\nscanio:" >> /$APP_NAME/config.yml && \
    echo -e "  home_folder: /$APP_NAME" >> /$APP_NAME/config.yml && \
    echo -e "  plugins_folder: /$APP_NAME/plugins" >> /$APP_NAME/config.yml && \
    echo -e "  projects_folder: /$APP_NAME/projects" >> /$APP_NAME/config.yml && \
    echo -e "  results_folder: /$APP_NAME/results" >> /$APP_NAME/config.yml && \
    echo -e "  temp_folder: /$APP_NAME/tmp" >> /$APP_NAME/config.yml && \
    echo -e "  artifacts_folder: /$APP_NAME/artifacts\n" >> /$APP_NAME/config.yml

# The compiled default config search paths are /scanio and ~/.scanio. Once the
# tree is relocated, point the binary at the moved config explicitly.
ENV SCANIO_CONFIG_PATH=/$APP_NAME/config.yml

RUN addgroup -S $APP_NAME && adduser -S -G $APP_NAME $APP_NAME && \
    chown -R $APP_NAME:$APP_NAME /$APP_NAME /data

# Exec-form ENTRYPOINT cannot interpolate $APP_NAME, so generate a wrapper that
# execs the renamed binary with clean arg and signal forwarding.
RUN printf '#!/bin/sh\nexec "/bin/%s" "$@"\n' "$APP_NAME" > /usr/local/bin/docker-entrypoint && \
    chmod +x /usr/local/bin/docker-entrypoint

USER $APP_NAME

ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
CMD ["--help"]