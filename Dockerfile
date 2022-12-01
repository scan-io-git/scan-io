FROM golang:1.19-buster AS build

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

# FROM debian:buster
FROM ubuntu:20.04

RUN apt-get update && \
    apt-get install -y ca-certificates curl && \
    apt-get install -y python3 python3-pip && \
    python3 -m pip install semgrep bandit

RUN curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list && \
    apt-get update && \
    apt-get install -y kubectl

RUN curl https://baltocdn.com/helm/signing.asc | gpg --dearmor | tee /usr/share/keyrings/helm.gpg > /dev/null && \
    apt-get install apt-transport-https --yes && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/helm.gpg] https://baltocdn.com/helm/stable/debian/ all main" | tee /etc/apt/sources.list.d/helm-stable-debian.list && \
    apt-get update && \
    apt-get install helm

# RUN adduser --home /home/scanio --shell /bin/bash --uid 1001 --disabled-password scanio
# USER scanio

ENV SCANIO_PLUGINS_FOLDER=/scanio-plugins
ENV SCANIO_HOME=/data

RUN mkdir -p $SCANIO_PLUGINS_FOLDER
COPY --from=build /usr/bin/scanio /bin/scanio
COPY --from=build /usr/bin/github $SCANIO_PLUGINS_FOLDER/github
COPY --from=build /usr/bin/gitlab $SCANIO_PLUGINS_FOLDER/gitlab
COPY --from=build /usr/bin/bitbucket $SCANIO_PLUGINS_FOLDER/bitbucket
COPY --from=build /usr/bin/semgrep $SCANIO_PLUGINS_FOLDER/semgrep
COPY --from=build /usr/bin/bandit $SCANIO_PLUGINS_FOLDER/bandit

COPY helm /scanio-helm
ENV JOB_HELM_CHART_PATH=/scanio-helm/scanio-job

# ENTRYPOINT ["/bin/scanio"]
# CMD ["--help"]

CMD ["/bin/bash"]
