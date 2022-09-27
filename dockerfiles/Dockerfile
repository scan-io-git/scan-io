FROM golang:1.19-buster AS build

WORKDIR /src
COPY cmd cmd
COPY plugins plugins
COPY shared shared
COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go

RUN go mod download
RUN go build -o /bin/scanio .
RUN go build -o /bin/github ./plugins/github 
RUN go build -o /bin/semgrep ./plugins/semgrep 

FROM debian:buster

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    apt-get install -y python3 python3-pip && \
    python3 -m pip install semgrep

RUN adduser --home /home/scanio --shell /bin/bash --uid 1001 --disabled-password scanio
USER scanio

RUN mkdir -p /home/scanio/.scanio/plugins
COPY --from=build /bin/scanio /bin/scanio
COPY --from=build /bin/github /home/scanio/.scanio/plugins/github
COPY --from=build /bin/semgrep /home/scanio/.scanio/plugins/semgrep

# ENTRYPOINT ["/bin/scanio"]
# CMD ["--help"]

CMD bash
