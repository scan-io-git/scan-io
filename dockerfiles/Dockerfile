FROM golang:1.19-buster AS build

WORKDIR /usr/src/scanio
COPY cmd cmd
COPY plugins plugins
COPY shared shared
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go

RUN go mod download
RUN go build -o /usr/bin/scanio .
RUN go build -o /usr/bin/github ./plugins/github 
RUN go build -o /usr/bin/gitlab ./plugins/gitlab 
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

ENV HOME=/root
ENV SCANIO_HOME=$HOME/.scanio

RUN mkdir -p $SCANIO_HOME/plugins
COPY --from=build /usr/bin/scanio /bin/scanio
COPY --from=build /usr/bin/github $SCANIO_HOME/plugins/github
COPY --from=build /usr/bin/gitlab $SCANIO_HOME/plugins/gitlab
COPY --from=build /usr/bin/semgrep $SCANIO_HOME/plugins/semgrep
COPY --from=build /usr/bin/bandit $SCANIO_HOME/plugins/bandit

# ENTRYPOINT ["/bin/scanio"]
# CMD ["--help"]

CMD ["/bin/bash"]
