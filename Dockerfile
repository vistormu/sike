# syntax=docker/dockerfile:1
# the image the homelab runs: a static binary on distroless, as nonroot.
# no node step: site.css is committed and embedded. the sqlite driver is pure
# go (modernc), so no cgo, and the binary needs nothing from the image but ca
# certificates (ntfy over https); the tz database is embedded (main.go).
# github builds and publishes it on a v* tag (.github/workflows/image.yml).
ARG GO_VERSION=1.26.7

FROM golang:${GO_VERSION} AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-mod=readonly
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
ARG VERSION=dev
RUN go build -trimpath -ldflags="-s -w -X sike/cmd.version=${VERSION}" -o /out/sike . \
    && mkdir /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/sike /sike
# the db lives in /data: mount a volume there, writable by nonroot (65532)
COPY --from=build --chown=65532:65532 /out/data /data
ENV SIKE_ADDR=:8090 SIKE_DB=/data/sike.db
VOLUME /data
EXPOSE 8090
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/sike", "health"]
USER nonroot
ENTRYPOINT ["/sike"]
CMD ["run"]
