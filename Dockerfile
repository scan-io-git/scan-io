# Optimize the image size by including only essential plugins and scanners specific to your processes.
# Note: Semgrep is a large third-party dependency, approximately 400MB in size.

# The Dockerfile facilitates multi-architecture builds. However, be cautious as trufflehog and helm currently only support linux/arm64 and linux/amd64 architectures. Always verify the compatibility of third-party versions before building.
# Important: As of now, Semgrep does not support ARM architectures - see https://github.com/returntocorp/semgrep/issues/2252 for details!

# Stage 1: Build Scanio core and plugins
FROM golang:1.19.8-alpine3.17 AS build-scanio

WORKDIR /usr/src/scanio

# Copy go.mod and go.sum for dependency resolution
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Set target architecture for multi-arch builds
ARG TARGETOS
ARG TARGETARCH

# Install make and other build dependencies
RUN apk update && \
    apk upgrade && \
    apk add --no-cache \
    make \
    jq

# Build the core and plugins using the Makefile
RUN echo "Building binaries and plugins for $TARGETOS/$TARGETARCH"
RUN make build CORE_BINARY=/usr/bin/scanio PLUGINS_DIR=/usr/bin/plugins

# Stage 2: Prepare the runtime environment
FROM python:3.11-alpine3.17

# RUN addgroup -g 101 scanio && \
#     adduser -h /home/scanio -s /bin/bash --uid 1001 -G scanio -D scanio && \
#     chown -R scanio:scanio $SCANIO_PLUGINS_FOLDER && \
#     chown -R scanio:scanio $SCANIO_HOME

# USER scanio:scanio

# Set target architecture for multi-arch builds
ARG TARGETOS
ARG TARGETARCH
RUN echo "Building dependencies for $TARGETOS/$TARGETARCH"

# Install dependencies
RUN apk update && \
    apk upgrade && \
    apk add --no-cache \
        bash \
        jq \
        openssh \
        libc6-compat 

RUN apk add --no-cache --virtual .build-deps \
                git \
                gcc \
                make \ 
                openssl \
                git \
                ca-certificates \
                curl \
                musl-dev

# Install Python dependencies
# To resolve a problem with same dependencies trufflehog3 has to be installed first
RUN python3 -m pip install trufflehog3
# Installing Semgrep 
RUN python3 -m pip install semgrep
# Installing Bandit 
RUN python3 -m pip install bandit

# Install Trufflehog Go
RUN export TRUFFLEHOG_VER="$(curl -s -qI https://github.com/trufflesecurity/trufflehog/releases/latest | awk -F '/' '/^location/ {print  substr($NF, 1, length($NF)-1)}' | awk -F 'v' '{print $2}')" && \
    export TRUFFLEHOG_SHA="$(curl -Ls https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VER}/trufflehog_${TRUFFLEHOG_VER}_checksums.txt | grep trufflehog_${TRUFFLEHOG_VER}_${TARGETOS}_${TARGETARCH}.tar.gz | awk '{print $1}')"  && \
    curl -LOs "https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VER}/trufflehog_${TRUFFLEHOG_VER}_${TARGETOS}_${TARGETARCH}.tar.gz" && \
    echo "${TRUFFLEHOG_SHA}  trufflehog_${TRUFFLEHOG_VER}_${TARGETOS}_${TARGETARCH}.tar.gz" | sha256sum -c - && \
    tar -xzf trufflehog_${TRUFFLEHOG_VER}_${TARGETOS}_${TARGETARCH}.tar.gz && \
    rm -rf trufflehog_${TRUFFLEHOG_VER}_${TARGETOS}_${TARGETARCH}.tar.gz && \
    mv trufflehog /usr/local/bin 

# Install Kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/${TARGETOS}/${TARGETARCH}/kubectl" && \
    curl -LO "https://dl.k8s.io/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/${TARGETOS}/${TARGETARCH}/kubectl.sha256" && \
    (echo "$(cat kubectl.sha256)  kubectl" | sha256sum -c ) && \
    rm -rf kubectl.sha256 && \
    chmod +x ./kubectl && \
    mv kubectl /usr/local/bin

# Install Helm
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
    chmod 700 get_helm.sh && \
    ./get_helm.sh && \
    rm -rf get_helm.sh

# Set environment variables // move to config file
ENV JOB_HELM_CHART_PATH=/scanio-helm/scanio-job

# Create necessary directories
RUN mkdir -p /scanio /data

# Copy built binaries and other necessary files from the build stage
COPY --from=build-scanio /usr/bin/scanio /bin/scanio
COPY --from=build-scanio /usr/bin/plugins/ /scanio/plugins/

# Copy additional resources
COPY rules /scanio/rules
COPY helm /scanio/helm
COPY Dockerfile /scanio/Dockerfile
COPY templates /scanio/templates
COPY VERSION /scanio/VERSION
COPY config.yml /scanio/config.yml

# Write to config.yml customized values
RUN echo "scanio:" >> /scanio/config.yml && \
    echo "  home_folder: /scanio" >> /scanio/config.yml && \
    echo "  plugins_folder: /scanio/plugins" >> /scanio/config.yml

ENTRYPOINT ["/bin/scanio"]
CMD ["--help"]
