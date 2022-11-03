FROM golang:1.19-buster AS build

WORKDIR /usr/src/scanio

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd cmd
COPY plugins plugins
COPY libs libs
COPY shared shared
COPY main.go main.go

RUN go build -o /usr/bin/scanio .
RUN go build -o /usr/bin/github ./plugins/github 
RUN go build -o /usr/bin/gitlab ./plugins/gitlab 
RUN go build -o /usr/bin/bitbucket ./plugins/bitbucket 
RUN go build -o /usr/bin/semgrep ./plugins/semgrep 
RUN go build -o /usr/bin/bandit ./plugins/bandit 

# FROM debian:buster
FROM ubuntu

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    apt-get install -y python3 python3-pip && \
    python3 -m pip install semgrep bandit

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

# ENTRYPOINT ["/bin/scanio"]
# CMD ["--help"]

CMD ["/bin/bash"]
