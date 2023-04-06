# To decrease the size of the image you could choose only mandatory plugins and scanners for your processes.
# For example, Semgrep is a really huge 3rd party dependency ~400MB. 

# Here we are building a main binary file and plugins from Golang code
FROM golang:1.19-buster AS build-scanio-plugins

WORKDIR /usr/src/scanio

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd cmd
COPY plugins plugins
COPY internal internal
COPY pkg pkg
COPY main.go main.go

RUN go build -o /usr/bin/scanio .
RUN go build -o /usr/bin/github ./plugins/github 
RUN go build -o /usr/bin/gitlab ./plugins/gitlab 
RUN go build -o /usr/bin/bitbucket ./plugins/bitbucket 
RUN go build -o /usr/bin/semgrep ./plugins/semgrep 
RUN go build -o /usr/bin/bandit ./plugins/bandit 
RUN go build -o /usr/bin/trufflehog ./plugins/trufflehog/

# Installing Trufflehog Go by unpacking binary
# ENV TRUFFLEHOG_VERSION 3.31.3
RUN export TRUFFLEHOG_VER="$(curl -s -qI https://github.com/trufflesecurity/trufflehog/releases/latest | awk -F '/' '/^location/ {print  substr($NF, 1, length($NF)-1)}' | awk -F 'v' '{print $2}')" && \
    export TRUFFLEHOG_SHA="$(curl -Ls https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VER}/trufflehog_${TRUFFLEHOG_VER}_checksums.txt | grep trufflehog_${TRUFFLEHOG_VER}_linux_amd64.tar.gz | awk '{print $1}')"  && \
    curl -LOs "https://github.com/trufflesecurity/trufflehog/releases/download/v${TRUFFLEHOG_VER}/trufflehog_${TRUFFLEHOG_VER}_linux_amd64.tar.gz" && \
    echo "${TRUFFLEHOG_SHA}  trufflehog_${TRUFFLEHOG_VER}_linux_amd64.tar.gz" | sha256sum -c - && \
    tar -xzf trufflehog_${TRUFFLEHOG_VER}_linux_amd64.tar.gz 


FROM python:alpine3.17
# Here we are preparing a container with all 3rd party dependencies for Scanio 

# RUN addgroup -g 101 scanio && \
#     adduser -h /home/scanio -s /bin/bash --uid 1001 -G scanio -D scanio && \
#     chown -R scanio:scanio $SCANIO_PLUGINS_FOLDER && \
#     chown -R scanio:scanio $SCANIO_HOME

# USER scanio:scanio

RUN apk update &&\
    apk upgrade

RUN apk add --no-cache \
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

RUN python3 -m pip install semgrep

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    curl -LO "https://dl.k8s.io/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256" && \
    (echo "$(cat kubectl.sha256)  kubectl" | sha256sum -c ) && \
    chmod +x ./kubectl && \
    mv kubectl /usr/local/bin

RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && \
        chmod 700 get_helm.sh && \
        ./get_helm.sh

ENV SCANIO_HOME=/data
ENV SCANIO_PLUGINS_FOLDER=/scanio-plugins
ENV JOB_HELM_CHART_PATH=/scanio-helm/scanio-job

RUN mkdir -p $SCANIO_HOME
RUN mkdir -p $SCANIO_PLUGINS_FOLDER

# Copying built binaries
COPY --from=build-scanio-plugins /usr/bin/scanio /bin/scanio
COPY --from=build-scanio-plugins /usr/bin/github $SCANIO_PLUGINS_FOLDER/github
COPY --from=build-scanio-plugins /usr/bin/gitlab $SCANIO_PLUGINS_FOLDER/gitlab
COPY --from=build-scanio-plugins /usr/bin/bitbucket $SCANIO_PLUGINS_FOLDER/bitbucket
COPY --from=build-scanio-plugins /usr/bin/semgrep $SCANIO_PLUGINS_FOLDER/semgrep
COPY --from=build-scanio-plugins /usr/bin/bandit $SCANIO_PLUGINS_FOLDER/bandit
COPY --from=build-scanio-plugins /usr/bin/trufflehog $SCANIO_PLUGINS_FOLDER/trufflehog

# Copy TrufflehogGo binary
COPY --from=build-scanio-plugins /usr/src/scanio/trufflehog /usr/local/bin

COPY helm /scanio-helm
COPY Dockerfile /Dockerfile

ENTRYPOINT ["/bin/scanio"]
CMD ["--help"]