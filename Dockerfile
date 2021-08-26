FROM golang:1.16 as build

WORKDIR /go/src/github.com/webdevops/azure-resourcegraph-exporter

# Get deps (cached)
COPY ./go.mod /go/src/github.com/webdevops/azure-resourcegraph-exporter
COPY ./go.sum /go/src/github.com/webdevops/azure-resourcegraph-exporter
RUN go mod download

# Compile
COPY ./ /go/src/github.com/webdevops/azure-resourcegraph-exporter
RUN make test
RUN make lint
RUN make build
RUN ./azure-resourcegraph-exporter --help

#############################################
# FINAL IMAGE
#############################################
FROM gcr.io/distroless/static
ENV LOG_JSON=1
COPY --from=build /go/src/github.com/webdevops/azure-resourcegraph-exporter/azure-resourcegraph-exporter /
USER 1000:1000
EXPOSE 8080
ENTRYPOINT ["/azure-resourcegraph-exporter"]
